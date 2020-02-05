package helm

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

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
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
	client           kubernetes.Interface = nil
	helmdir          string
)

func TestMain(m *testing.M) {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"helm": testAccProvider,
	}

	if dir, err := ioutil.TempDir(os.TempDir(), "helmhome"); err != nil {
		panic(err)
	} else {
		helmdir = dir
	}

	ec := m.Run()

	os.RemoveAll(helmdir)

	os.Exit(ec)
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T, namespace string) {
	err := setK8Client()

	if err != nil {
		t.Fatal(err)
	}

	err = testAccProvider.Configure(terraform.NewResourceConfigRaw(nil))

	if err != nil {
		t.Fatal(err)
	}

	if namespace != "" {
		createNamespace(t, namespace)
	}

	home, err := ioutil.TempDir(helmdir, "helm")

	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("HELM_REPOSITORY_CONFIG", filepath.Join(home, "config/repositories.yaml"))
	os.Setenv("HELM_REPOSITORY_CACHE", filepath.Join(home, "cache/helm/repository"))
	os.Setenv("HELM_REGISTRY_CONFIG", filepath.Join(home, "config/registry.json"))
	os.Setenv("HELM_PLUGINS", filepath.Join(home, "plugins"))
	os.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	//os.Setenv("HELM_DEBUG", "true")
	//os.Setenv("TF_LOG", "DEBUG")
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

	options := metav1.GetOptions{}

	_, err := client.CoreV1().Namespaces().Get(namespace, options)

	if err == nil {
		return
	}

	k8ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	t.Log("[DEBUG] Creating namespace", namespace)
	_, err = client.CoreV1().Namespaces().Create(k8ns)
	if err != nil {
		// No failure here, the concurrency tests will blow up if we fail. Tried
		// Locking in this method, but it causes the tests to hang
		t.Log("An error occurred while creating namespace", namespace, err)
	}
}
