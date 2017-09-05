package helm

import (
	"fmt"
	"regexp"
	"testing"

	yaml "gopkg.in/yaml.v1"

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
		Providers: testAccProviders,
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

func TestAccResourceChart_updateAfterFail(t *testing.T) {
	malformed := `
	resource "helm_chart" "test" {
	  name        = "malformed"
	  chart       = "stable/nginx-ingress"
	  set {
	      name = "controller.podAnnotations.\"prometheus\\.io/scrape\""
	      value = "true"
	  }
	}
	`

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmChartDestroy,
		Steps: []resource.TestStep{{
			Config:             malformed,
			ExpectError:        regexp.MustCompile("failed"),
			ExpectNonEmptyPlan: true,
		}, {
			Config: testAccHelmChartConfigBasic(testNamespace, testReleaseName, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_chart.test", "metadata.0.status", "DEPLOYED"),
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

			set {
				name = "foo"
				value = "qux"
			}

			set {
				name = "qux.bar"
				value = 1
			}
		}
	`, name, ns, version)
}

func TestGetValues(t *testing.T) {
	d := resourceChart().Data(nil)
	d.Set("values", `foo: bar`)
	d.Set("set", []interface{}{
		map[string]interface{}{"name": "foo", "value": "qux"},
	})

	values, err := getValues(d)
	if err != nil {
		t.Fatalf("error getValues: %s", err)
		return
	}

	base := map[string]string{}
	err = yaml.Unmarshal([]byte(values), &base)
	if err != nil {
		t.Fatalf("error parsing returned yaml: %s", err)
		return
	}

	if base["foo"] != "qux" {
		t.Fatalf("error merging values, expected %q, got %q", "qux", base["foo"])
	}
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
	m := testAccProvider.Meta()
	if m == nil {
		return fmt.Errorf("provider not properly initialized")
	}

	client, err := m.(*Meta).GetHelmClient()
	if err != nil {
		return err
	}

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
