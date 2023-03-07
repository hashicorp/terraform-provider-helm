// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	testNamespacePrefix = "terraform-acc-test"
	testResourceName    = "test"
	testChartsPath      = "./testdata/charts"
	testRepositoryDir   = "./testdata/repository"
)

var (
	accTest           bool
	testRepositoryURL string

	testAccProviders map[string]*schema.Provider
	testAccProvider  *schema.Provider
	client           kubernetes.Interface = nil
)

func TestMain(m *testing.M) {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"helm": testAccProvider,
	}

	home, err := ioutil.TempDir(os.TempDir(), "helm")

	if err != nil {
		panic(err)
	}

	err = os.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(home, "config/repositories.yaml"))
	if err != nil {
		panic(err)
	}

	err = os.Setenv("HELM_REPOSITORY_CACHE", filepath.Join(home, "cache/helm/repository"))
	if err != nil {
		panic(err)
	}

	err = os.Setenv("HELM_REGISTRY_CONFIG", filepath.Join(home, "config/registry.json"))
	if err != nil {
		panic(err)
	}

	err = os.Setenv("HELM_PLUGINS", filepath.Join(home, "plugins"))
	if err != nil {
		panic(err)
	}

	err = os.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	if err != nil {
		panic(err)
	}

	accTest = os.Getenv("TF_ACC") == "1"

	var stopRepositoryServer func()
	if accTest {
		_, err := exec.LookPath("helm")
		if err != nil {
			panic(`command "helm" needs to be available to run the test suite`)
		}

		// create the Kubernetes client
		c, err := createKubernetesClient()
		if err != nil {
			panic(err)
		}
		client = c

		// Build the test repository and start the server
		buildChartRepository()
		testRepositoryURL, stopRepositoryServer = startRepositoryServer()
		log.Println("Test repository is listening on", testRepositoryURL)
	}

	ec := m.Run()

	err = os.RemoveAll(home)
	if err != nil {
		panic(err)
	}

	if accTest {
		stopRepositoryServer()
		cleanupChartRepository()
	}

	os.Exit(ec)
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// buildChartRepository packages all the test charts and builds the repository index
func buildChartRepository() {
	log.Println("Building chart repository...")

	if _, err := os.Stat(testRepositoryDir); os.IsNotExist(err) {
		os.Mkdir(testRepositoryDir, os.ModePerm)
	}

	charts, err := ioutil.ReadDir(testChartsPath)
	if err != nil {
		panic(err)
	}

	// package all the charts
	for _, c := range charts {
		cmd := exec.Command("helm", "package", "-u",
			filepath.Join(testChartsPath, c.Name()),
			"-d", testRepositoryDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println(string(out))
			panic(err)
		}

		log.Printf("Created repository package for %q\n", c.Name())
	}

	// build the repository index
	cmd := exec.Command("helm", "repo", "index", testRepositoryDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		panic(err)
	}

	log.Println("Built chart repository index")
}

// cleanupChartRepository cleans up the repository of test charts
func cleanupChartRepository() {
	if _, err := os.Stat(testRepositoryDir); err == nil {
		err := os.RemoveAll(testRepositoryDir)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// startRepositoryServer starts a helm repository in a goroutine using
// a plain HTTP server on a random port and returns the URL
func startRepositoryServer() (string, func()) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	var shutdownFunc func()
	go func() {
		fileserver := http.Server{
			Handler: http.FileServer(http.Dir(testRepositoryDir)),
		}
		// NOTE we disable keep alive to prevent the server from chewing
		// up a lot of open connections as the test suite is run
		fileserver.SetKeepAlivesEnabled(false)
		shutdownFunc = func() { fileserver.Shutdown(context.Background()) }
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			panic(err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		testRepositoryURL = fmt.Sprintf("http://localhost:%d", port)
		wg.Done()
		err = fileserver.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	wg.Wait()

	return testRepositoryURL, shutdownFunc
}

func testAccPreCheck(t *testing.T) {
	if !accTest {
		t.Skip("TF_ACC=1 not set")
	}
	http.DefaultClient.CloseIdleConnections()
	ctx := context.TODO()
	diags := testAccProvider.Configure(ctx, terraform.NewResourceConfigRaw(nil))

	if diags.HasError() {
		t.Fatal(diags)
	}
}

func createKubernetesClient() (kubernetes.Interface, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	kubeconfig := os.Getenv("KUBE_CONFIG_PATH")
	if kubeconfig == "" {
		panic("Need to set KUBE_CONFIG_PATH")
	}
	rules.ExplicitPath = kubeconfig

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func createRandomNamespace(t *testing.T) string {
	if !accTest {
		t.Skip("TF_ACC=1 not set")
		return ""
	}

	namespace := fmt.Sprintf("%s-%s", testNamespacePrefix, acctest.RandString(10))
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Could not create test namespace %q: %s", namespace, err)
	}
	return namespace
}

func deleteNamespace(t *testing.T, namespace string) {
	if !accTest {
		t.Skip("TF_ACC=1 not set")
		return
	}

	gracePeriodSeconds := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}
	err := client.CoreV1().Namespaces().Delete(context.TODO(), namespace, deleteOptions)
	if err != nil {
		t.Fatalf("An error occurred while deleting namespace %q: %q", namespace, err)
	}
}

func randName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, acctest.RandString(10))
}
