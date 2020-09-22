package helm

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	testNamespace        = "terraform-acc-test"
	testResourceName     = "test"
	testRepositoryName   = "test-repository"
	testRepositoryURL    = "https://kubernetes-charts.storage.googleapis.com"
	testRepositoryURLAlt = "https://kubernetes-charts-incubator.storage.googleapis.com"
)

var (
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
		panic("Could not create temporary directory for helm config files")
	}

	err = os.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(home, "config/repositories.yaml"))
	if err != nil {
		panic("setenv failed for HELM_REPOSITORY_CONFIG")
	}

	err = os.Setenv("HELM_REPOSITORY_CACHE", filepath.Join(home, "cache/helm/repository"))
	if err != nil {
		panic("setenv failed for HELM_REPOSITORY_CACHE")
	}

	err = os.Setenv("HELM_REGISTRY_CONFIG", filepath.Join(home, "config/registry.json"))
	if err != nil {
		panic("setenv failed for HELM_REGISTRY_CONFIG")
	}

	err = os.Setenv("HELM_PLUGINS", filepath.Join(home, "plugins"))
	if err != nil {
		panic("setenv failed for HELM_PLUGINS")
	}

	err = os.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	if err != nil {
		panic("setenv failed for XDG_CACHE_HOME")
	}

	ec := m.Run()

	err = os.RemoveAll(home)
	if err != nil {
		panic("TempDir deletion failed")
	}

	os.Exit(ec)
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T, namespace string) {
	ctx := context.TODO()
	err := setK8Client()

	if err != nil {
		t.Fatal(err)
	}

	diags := testAccProvider.Configure(ctx, terraform.NewResourceConfigRaw(nil))
	if diags.HasError() {
		t.Fatal(diags)
	}

	if namespace != "" {
		createNamespace(t, namespace)
	}
}

func setK8Client() error {

	if client != nil {
		return nil
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()

	if err != nil {
		return err
	}

	if config == nil {
		config = &rest.Config{}
	}

	c, err := kubernetes.NewForConfig(config)

	if err != nil {
		return err
	}

	client = c

	return nil
}

func createNamespace(t *testing.T, namespace string) {
	// Nothing to cleanup with unit test
	if os.Getenv("TF_ACC") == "" {
		t.Log("TF_ACC Not Set")
		return
	}

	m := testAccProvider.Meta()
	if m == nil {
		t.Fatal("provider not properly initialized")
	}

	getOptions := metav1.GetOptions{}

	_, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, getOptions)

	if err == nil {
		return
	}

	k8ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	t.Log("[DEBUG] Creating namespace", namespace)

	createOptions := metav1.CreateOptions{}

	_, err = client.CoreV1().Namespaces().Create(context.TODO(), k8ns, createOptions)
	if err != nil {
		// No failure here, the concurrency tests will blow up if we fail. Tried
		// Locking in this method, but it causes the tests to hang
		t.Log("An error occurred while creating namespace", namespace, err)
	}
}
