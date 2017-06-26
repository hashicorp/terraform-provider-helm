package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"k8s.io/helm/pkg/helm"
)

func TestAccResourceChart_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmChartDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmChartConfigBasic(testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.name", testReleaseName),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.namespace", testNamespace),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.chart", "mariadb"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.version", "0.6.2"),
			),
		}, {
			Config: testAccHelmChartConfigBasic(testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceChart_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmChartDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmChartConfigBasic(testNamespace, testReleaseName, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmChartConfigBasic(testNamespace, testReleaseName, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceChart_repository(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmChartDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmChartConfigRepository(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_chart.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmChartConfigRepository(testNamespace, testReleaseName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_chart.test", "metadata.0.version"),
			),
		}},
	})
}

func testAccHelmChartConfigBasic(ns, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_chart" "test" {
 			name      = %q
			namespace = %q
  			chart     = "stable/mariadb"
			version   = %q

			value {
				name = "foo"
				content = "qux"
			}

			value {
				name = "qux.bar"
				content = 1
			}
		}
	`, name, ns, version)
}

func testAccHelmChartConfigRepository(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "incubator" {
			name = "incubator"
			url  = "https://kubernetes-charts-incubator.storage.googleapis.com"
		}

		resource "helm_chart" "test" {
 			name       = %q
			namespace  = %q
			repository = "${helm_repository.incubator.metadata.0.name}"
  			chart      = "redis-cache"
		}
	`, name, ns)
}

func testAccCheckHelmChartDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(helm.Interface)

	res, err := client.ListReleases(
		helm.ReleaseListNamespace(testNamespace),
	)

	if err != nil {
		return err
	}

	for _, r := range res.Releases {
		if r.Name == testReleaseName {
			return fmt.Errorf("found %q release", testReleaseName)
		}
	}

	if res.Count != 0 {
		return fmt.Errorf("%q namespace should be empty", testNamespace)
	}

	return nil
}
