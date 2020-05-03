package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccDataTemplate_basic(t *testing.T) {
	name := fmt.Sprintf("test-basic-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))

	config := testAccDataHelmTemplateConfigBasic(testResourceName, namespace, name, "7.1.0")

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t, "") },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "6"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/secrets.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-configmap.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/tests.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-svc.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-statefulset.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/test-runner.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest_bundle"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
			),
		}},
	})
}

func TestAccDataTemplate_templates(t *testing.T) {
	name := fmt.Sprintf("test-templates-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))

	config := testAccDataHelmTemplateConfigTemplates(testResourceName, namespace, name, "7.1.0")

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t, "") },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "3"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-statefulset.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-svc.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/master-configmap.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest_bundle"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
			),
		}},
	})
}

func testAccDataHelmTemplateConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = "https://kubernetes-charts.storage.googleapis.com"
  			chart       = "mariadb"
			version     = %q

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}

			set {
				name = "master.persistence.enabled"
				value = false # persistent volumes are giving non-related issues when testing
			}
			set {
				name = "replication.enabled"
				value = false
			}
		}
	`, resource, name, ns, version)
}

func testAccDataHelmTemplateConfigTemplates(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = "https://kubernetes-charts.storage.googleapis.com"
  			chart       = "mariadb"
			version     = %q

			templates = [
				"templates/master-statefulset.yaml",
				"templates/master-svc.yaml",
				"templates/master-configmap.yaml",
			]

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}

			set {
				name = "master.persistence.enabled"
				value = false # persistent volumes are giving non-related issues when testing
			}
			set {
				name = "replication.enabled"
				value = false
			}
		}
	`, resource, name, ns, version)
}
