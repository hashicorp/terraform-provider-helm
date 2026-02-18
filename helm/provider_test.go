// Copyright IBM Corp. 2017, 2026
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

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
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
	client            kubernetes.Interface = nil
	testMeta          *Meta
)

var providerFactory map[string]func() (tfprotov6.ProviderServer, error)

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	if len(providerFactory) != 0 {
		return providerFactory
	}

	providerFactory = map[string]func() (tfprotov6.ProviderServer, error){
		"helm": providerserver.NewProtocol6WithError(New("test")()),
	}

	return providerFactory
}

func TestMain(m *testing.M) {
	home, err := ioutil.TempDir(os.TempDir(), "helm")
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(home)

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
	}

	ec := m.Run()

	if accTest {
		stopRepositoryServer()
		cleanupChartRepository()
	}

	os.Exit(ec)
}

// todo
func TestProvider(t *testing.T) {
	ctx := context.Background()
	provider := New("test")()

	// Create the provider server
	providerServer, err := createProviderServer(provider)
	if err != nil {
		t.Fatalf("Failed to create provider server: %s", err)
	}
	// Perform config validation

	validateResponse, err := providerServer.ValidateProviderConfig(ctx, &tfprotov6.ValidateProviderConfigRequest{})
	if err != nil {
		t.Fatalf("Provider config validation failed, error: %v", err)
	}

	if hasError(validateResponse.Diagnostics) {
		t.Fatalf("Provider config validation failed, diagnostics: %v", validateResponse.Diagnostics)
	}
}

func createProviderServer(provider provider.Provider) (tfprotov6.ProviderServer, error) {
	providerServerFunc := providerserver.NewProtocol6WithError(provider)
	server, err := providerServerFunc()
	if err != nil {
	} else {
	}
	return server, err
}

func buildChartRepository() {
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

func cleanupChartRepository() {
	if _, err := os.Stat(testRepositoryDir); err == nil {
		err := os.RemoveAll(testRepositoryDir)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func startRepositoryServer() (string, func()) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	var shutdownFunc func()
	go func() {
		fileserver := http.Server{
			Handler: http.FileServer(http.Dir(testRepositoryDir)),
		}
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

func createAndConfigureProviderServer(provider provider.Provider, ctx context.Context) (tfprotov6.ProviderServer, error) {
	log.Println("Starting createAndConfigureProviderServer...")

	providerServerFunc := providerserver.NewProtocol6WithError(provider)
	providerServer, err := providerServerFunc()
	if err != nil {
		return nil, fmt.Errorf("Failed to create protocol6 provider: %w", err)
	}
	log.Println("Provider server function created successfully.")

	configResponse, err := providerServer.ConfigureProvider(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error configuring provider: %w", err)
	}
	log.Println("Provider configured successfully.")

	if hasError(configResponse.Diagnostics) {
		return nil, fmt.Errorf("Provider configuration failed, diagnostics: %#v", configResponse.Diagnostics[0])
	}

	if helmProvider, ok := provider.(*HelmProvider); ok {
		testMeta = helmProvider.meta
		if testMeta == nil {
			log.Println("testMeta is nil after type assertion.")
		} else {
			log.Printf("testMeta initialized: %+v", testMeta)
		}
	} else {
		return nil, fmt.Errorf("Failed to type assert provider to HelmProvider")
	}

	return providerServer, nil
}

func testAccPreCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping acceptance tests in short mode")
	}
	http.DefaultClient.CloseIdleConnections()

	ctx := context.TODO()

	provider := New("test")()

	// Create and configure the ProviderServer
	_, err := createAndConfigureProviderServer(provider, ctx)
	if err != nil {
		t.Fatalf("Pre-check failed: %v", err)
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

func hasError(diagnostics []*tfprotov6.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}

func DynamicValueEmpty() *tfprotov6.DynamicValue {
	return &tfprotov6.DynamicValue{
		MsgPack: nil,
		JSON:    nil,
	}
}
