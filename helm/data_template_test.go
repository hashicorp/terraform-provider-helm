package helm

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"testing"
)

func testCheckResourceAttrNotEmpty(name, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()

		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s in %s", name, ms.Path)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s in %s", name, ms.Path)
		}

		if value, ok := is.Attributes[key]; !ok || len(value) == 0 {
			if !ok {
				return fmt.Errorf("%s: Attribute '%s' not found", name, key)
			}

			return fmt.Errorf("%s: Attribute '%s' expected to be not empty", name, key)
		}

		return nil
	}
}

func TestAccDataTemplate_basic(t *testing.T) {
	r := acctest.RandString(10)

	name := fmt.Sprintf("test-basic-%s", r)
	namespace := fmt.Sprintf("%s-%s", testNamespace, r)

	config := testAccHelmDataTemplateConfigBasic(testResourceName, namespace, name, "0.6.2")

	expectedNotesTemplate := `MariaDB can be accessed via port 3306 on the following DNS name from within your cluster:
%[2]s-mariadb.%[1]s.svc.cluster.local

To connect to your database:

1. Run a pod that you can use as a client:

    kubectl run %[2]s-mariadb-client --rm --tty -i --image bitnami/mariadb --command -- bash

2. Connect using the mysql cli, then provide your password:
    $ mysql -h %[2]s-mariadb`

	expectedNotes := fmt.Sprintf(expectedNotesTemplate, namespace, name)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceAddress, "name", name),
					resource.TestCheckResourceAttr(datasourceAddress, "namespace", namespace),
					testCheckResourceAttrNotEmpty(datasourceAddress, "rendered"),
					resource.TestCheckResourceAttr(datasourceAddress, "notes", expectedNotes),
					resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "5"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/secrets.yaml"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/configmap.yaml"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/pvc.yaml"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/svc.yaml"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/deployment.yaml"),
				),
			},
		},
	})
}

func TestAccDataTemplate_explicitTemplates(t *testing.T) {
	r := acctest.RandString(10)

	name := fmt.Sprintf("test-explicitTemplates-%s", r)
	namespace := fmt.Sprintf("%s-%s", testNamespace, r)

	config := testAccHelmDataTemplateConfigExplicitTemplates(testResourceName, namespace, name, "0.6.2")

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testCheckResourceAttrNotEmpty(datasourceAddress, "rendered"),
					resource.TestCheckResourceAttr(datasourceAddress, "notes", ""),
					resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "2"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/svc.yaml"),
					testCheckResourceAttrNotEmpty(datasourceAddress, "manifests.mariadb/templates/deployment.yaml"),
				),
			},
		},
	})
}

func testAccHelmDataTemplateConfigBasic(dataSource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name      = %q
			namespace = %q
  			chart     = "stable/mariadb"
			version   = %q

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}
		}
	`, dataSource, name, ns, version)
}

func testAccHelmDataTemplateConfigExplicitTemplates(dataSource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name      = %q
			namespace = %q
  			chart     = "stable/mariadb"
			version   = %q

			templates = [
				"templates/deployment.yaml",
				"templates/svc.yaml",
			]

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}
		}
	`, dataSource, name, ns, version)
}
