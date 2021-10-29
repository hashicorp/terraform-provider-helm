package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataTemplate_basic(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateConfigBasic(testResourceName, namespace, name, "1.2.3", false),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "5"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/deployment.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/service.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/serviceaccount.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/configmaps.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/tests/test-connection.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
			),
		}},
	})
}

func TestAccDataTemplate_templates(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)
	expectedTemplate := fmt.Sprintf(`---
# Source: test-chart/templates/configmaps.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: %[1]s-test-chart-one
  labels:
    helm.sh/chart: test-chart-1.2.3
    app.kubernetes.io/name: test-chart
    app.kubernetes.io/instance: %[1]s
    app.kubernetes.io/version: "1.19.5"
    app.kubernetes.io/managed-by: Helm
data:
  test: one
---
# Source: test-chart/templates/configmaps.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: %[1]s-test-chart-two
  labels:
    helm.sh/chart: test-chart-1.2.3
    app.kubernetes.io/name: test-chart
    app.kubernetes.io/instance: %[1]s
    app.kubernetes.io/version: "1.19.5"
    app.kubernetes.io/managed-by: Helm
data:
  test: two
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateConfigTemplates(testResourceName, namespace, name, "1.2.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "1"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.templates/configmaps.yaml", expectedTemplate),
			),
		}},
	})
}

func TestAccDataTemplate_crds(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateConfigBasic(testResourceName, namespace, name, "1.2.3", true),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "7"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.multiple.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.single.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttr(datasourceAddress, "crds.%", "2"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "crds.multiple.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "crds.single.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "crd"),
			),
		}},
	})
}

func testAccDataHelmTemplateConfigBasic(resource, ns, name, version string, includeCrds bool) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = %q
  			chart       = "test-chart"
			version     = %q
      include_crds = %t

			set {
				name = "foo"
				value = "bar"
			}

			set {
				name = "fizz"
				value = 1337
			}
		}
  `, resource, name, ns, testRepositoryURL, version, includeCrds)
}

func testAccDataHelmTemplateConfigTemplates(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = %q
  			chart       = "test-chart"
			version     = %q

			set {
				name = "foo"
				value = "bar"
			}

			set {
				name = "fizz"
				value = 1337
			}

			show_only = [
				"templates/configmaps.yaml",
			]
		}
	`, resource, name, ns, testRepositoryURL, version)
}
