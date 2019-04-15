package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	testNamespace         = "terraform-acc-test"
	testResourceName      = "test"
	testRepositoryName    = "test-repository"
	testRepositoryURL     = "https://kubernetes-charts.storage.googleapis.com"
	testRepositoryURLAlt  = "https://kubernetes-charts-incubator.storage.googleapis.com"
	kubeConfigTestFixture = "test-fixtures/kube-config.yaml"
)

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"helm": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_configure(t *testing.T) {
	checkSkipAccTest(t)
	resetEnv := unsetEnv(t)
	defer resetEnv()

	os.Setenv("KUBECONFIG", kubeConfigTestFixture)
	os.Setenv("KUBE_CTX", "gcp")

	c, err := config.NewRawConfig(map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	rc := terraform.NewResourceConfig(c)
	p := Provider()
	err = p.Configure(rc)
	if err != nil {
		t.Fatal(err)
	}
}

func TestProvider_configureWithConfigContent(t *testing.T) {
	checkSkipAccTest(t)
	resetEnv := unsetEnv(t)
	defer resetEnv()

	os.Setenv("KUBECONFIG", "dont/accidentally/load/me")

	configbytes, err := ioutil.ReadFile(kubeConfigTestFixture)
	if err != nil {
		t.Errorf("Error reading test fixture %s: %s", kubeConfigTestFixture, err)
	}
	configContent := string(configbytes)

	c, err := config.NewRawConfig(map[string]interface{}{
		"kubernetes": []map[string]interface{}{{
			"config_content": configContent,
			"config_context": "gcp",
		}},
	})
	if err != nil {
		t.Fatalf("Error setting up test config: %s", err)
	}
	rc := terraform.NewResourceConfig(c)
	p := Provider()
	err = p.Configure(rc)
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func checkSkipAccTest(t *testing.T) {
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skip(fmt.Sprintf(
			"Acceptance tests skipped unless env '%s' set",
			resource.TestEnvVar))
	}
}

func unsetEnv(t *testing.T) func() {
	e := getEnv()
	for envKey := range e {
		if err := os.Unsetenv(envKey); err != nil {
			t.Fatalf("Error unsetting env var \"%s\": %s", envKey, err)
		}
	}
	return func() {
		for envKey, envValue := range e {
			if err := os.Setenv(envKey, envValue); err != nil {
				t.Fatalf("Error resetting env var \"%s\": %s", envKey, err)
			}
		}
	}
}

func getEnv() map[string]string {
	envMap := make(map[string]string)
	for _, envKey := range envVars {
		envMap[envKey] = os.Getenv(envKey)
	}
	return envMap
}

var envVars = []string{
	"HELM_HOME",
	"HELM_HOST",
	"HELM_NO_PLUGINS",
	"KUBE_CONFIG",
	"KUBECONFIG",
	"KUBE_CTX",
	"KUBE_CTX_AUTH_INFO",
	"KUBE_CTX_CLUSTER",
	"KUBE_HOST",
	"KUBE_USER",
	"KUBE_PASSWORD",
	"KUBE_CLIENT_CERT_DATA",
	"KUBE_CLIENT_KEY_DATA",
	"KUBE_CLUSTER_CA_CERT_DATA",
}
