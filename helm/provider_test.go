package helm

import (
	"os"
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
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"helm": testAccProvider,
	}
}

// 	var err error
// 	testAccHelmHome, err = ioutil.TempDir("", "terraform-acc-test-helm-")
// 	if err != nil {
// 		log.Printf("[ERROR] Failed to create new temporary directory for use as helm home: %s", err)
// 	}
// }

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// func TestAccProviderHelmInitEnabled(t *testing.T) {
// 	err := testAccProviderHelmInit(t, true)
// 	if err != nil {
// 		t.Fatal("helm home should have been initialized", err)
// 	}
// }

// func TestAccProviderHelmInitDisabled(t *testing.T) {
// 	err := testAccProviderHelmInit(t, false)
// 	if err != io.EOF {
// 		t.Fatal("helm home should not have been initialized", err)
// 	}
// }

// func testAccProviderHelmInit(t *testing.T, enabled bool) (err error) {
// 	if os.Getenv(resource.TestEnvVar) == "" {
// 		t.Skip(fmt.Sprintf(
// 			"Acceptance tests skipped unless env '%s' set", resource.TestEnvVar))
// 		return
// 	}

// 	helmHome, err := ioutil.TempDir("", "terraform-acc-test-helm-")
// 	if err != nil {
// 		t.Fatalf("Failed to create new temporary directory for use as helm home: %s", err)
// 	}
// 	defer os.RemoveAll(helmHome)

// 	log.Printf("[INFO] Test: Using %s as helm home", helmHome)

// 	testProvider := Provider().(*schema.Provider)
// 	err = testProvider.Configure(terraform.NewResourceConfigRaw(map[string]interface{}{
// 		"home":           helmHome,
// 		"init_helm_home": enabled,
// 	}))
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	f, err := os.Open(helmHome)
// 	defer f.Close()
// 	if err != nil {
// 		t.Fatal("Failed to access helm home directory")
// 	}

// 	_, err = f.Readdirnames(1)
// 	return
// }

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

	os.Setenv("HELM_REPOSITORY_CONFIG", "/Users/amell/Library/Preferences/helm/repositories.yaml")
	os.Setenv("HELM_REPOSITORY_CACHE", "/Users/amell/Library/Caches/helm/repository")
	os.Setenv("HELM_REGISTRY_CONFIG", "/Users/amell/Library/Preferences/helm/registry.json")
	os.Setenv("HELM_PLUGINS", "/Users/amell/Library/helm/plugins")
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
