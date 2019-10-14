package helm

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
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
	testAccHelmHome  string
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"helm": testAccProvider,
	}

	var err error
	testAccHelmHome, err = ioutil.TempDir("", "terraform-acc-test-helm-")
	if err != nil {
		log.Printf("[ERROR] Failed to create new temporary directory for use as helm home: %s", err)
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestAccProviderHelmInitEnabled(t *testing.T) {
	err := testAccProviderHelmInit(t, true)
	if err != nil {
		t.Fatal("helm home should have been initialized", err)
	}
}

func TestAccProviderHelmInitDisabled(t *testing.T) {
	err := testAccProviderHelmInit(t, false)
	if err != io.EOF {
		t.Fatal("helm home should not have been initialized", err)
	}
}

func testAccProviderHelmInit(t *testing.T, enabled bool) (err error) {
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skip(fmt.Sprintf(
			"Acceptance tests skipped unless env '%s' set", resource.TestEnvVar))
		return
	}

	helmHome, err := ioutil.TempDir("", "terraform-acc-test-helm-")
	if err != nil {
		t.Fatalf("Failed to create new temporary directory for use as helm home: %s", err)
	}
	defer os.RemoveAll(helmHome)

	log.Printf("[INFO] Test: Using %s as helm home", helmHome)

	testProvider := Provider().(*schema.Provider)
	err = testProvider.Configure(terraform.NewResourceConfigRaw(map[string]interface{}{
		"home":           helmHome,
		"init_helm_home": enabled,
	}))
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(helmHome)
	defer f.Close()
	if err != nil {
		t.Fatal("Failed to access helm home directory")
	}

	_, err = f.Readdirnames(1)
	return
}

func testAccPreCheck(t *testing.T) {
	log.Printf("[INFO] Test: Using %s as helm home", testAccHelmHome)
	os.Setenv("HELM_HOME", testAccHelmHome)

	err := testAccProvider.Configure(terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}
