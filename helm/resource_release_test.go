package helm

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gopkg.in/yaml.v1"
	"k8s.io/helm/pkg/helm"
)

func TestAccResourceRelease_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, testNamespace, "test-basic", "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", "test-basic"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", testNamespace),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "mariadb"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, testNamespace, "test-basic", "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_concurrent(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(name string) {
			defer wg.Done()
			resource.Test(t, resource.TestCase{
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckHelmReleaseDestroy,
				Steps: []resource.TestStep{{
					Config: testAccHelmReleaseConfigBasic(name, testNamespace, name, "0.6.2"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							fmt.Sprintf("helm_release.%s", name), "metadata.0.name", name,
						),
					),
				}},
			})
		}(fmt.Sprintf("concurrent-%d", i))
	}

	wg.Wait()
}

func TestAccResourceRelease_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, testNamespace, "test-update", "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, testNamespace, "test-update", "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_emptyValuesList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, "test-empty-values-list", "stable/kibana", []string{""},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{}\n"),
			),
		}},
	})
}

func TestAccResourceRelease_updateValues(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, "test-update-values", "stable/kibana", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: bar\n"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, "test-update-values", "stable/kibana", []string{"foo: baz"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: baz\n"),
			),
		}},
	})
}

func TestAccResourceRelease_updateMultipleValues(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, "test-update-multiple-values",
				"stable/kibana", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: bar\n"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, "test-update-multiple-values",
				"stable/kibana", []string{"foo: bar", "foo: baz"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: baz\n"),
			),
		}},
	})
}

func TestAccResourceRelease_repository(t *testing.T) {

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepository(testNamespace, testResourceName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepository(testNamespace, testResourceName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_repository_url(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryURL(testNamespace, testResourceName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
				resource.TestCheckResourceAttrSet("helm_release.test", "version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryURL(testNamespace, testResourceName),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
				resource.TestCheckResourceAttrSet("helm_release.test", "version"),
			),
		}},
	})
}

func TestAccResourceRelease_updateAfterFail(t *testing.T) {
	malformed := `
	resource "helm_release" "test" {
		name        = "malformed"
		chart       = "stable/nginx-ingress"
		set {
			name = "controller.name"
			value = "invalid-$%!-character-for-k8s-label"
		}
	}
	`

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config:             malformed,
			ExpectError:        regexp.MustCompile("failed"),
			ExpectNonEmptyPlan: true,
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, testNamespace, testResourceName, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_updateExistingFailed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, testResourceName, "stable/mariadb",
				[]string{"master:\n  persistence:\n    enabled: false", "replication:\n  enabled: false"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, testResourceName, "stable/mariadb",
				[]string{"master:\n  persistence:\n    enabled: true", "replication:\n  enabled: false"},
			),
			ExpectError:        regexp.MustCompile("forbidden"),
			ExpectNonEmptyPlan: true,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "FAILED"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, testNamespace, testResourceName, "stable/mariadb",
				[]string{"master:\n  persistence:\n    enabled: true", "replication:\n  enabled: false"},
			),
			ExpectError:        regexp.MustCompile("forbidden"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccResourceRelease_updateVersionFromRelease(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	chartPath := filepath.Join(dir, "mariadb")
	defer os.RemoveAll(dir)
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy,
		Steps: []resource.TestStep{{
			PreConfig: func() {
				err := downloadTar("https://kubernetes-charts.storage.googleapis.com/mariadb-0.6.2.tgz", dir)
				if err != nil {
					t.Fatal(err)
				}
			},
			Config: fmt.Sprintf(`
			resource "helm_release" "test" {
				name      = %q
				namespace = %q
				chart     = %q
				set {
					name = "persistence.enabled"
					value = "false" # persistent volumes are giving non-related issues when testing
				}
			}
		`, testNamespace, testResourceName, chartPath),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "version", "0.6.2"),
			),
		}, {
			PreConfig: func() {
				err := downloadTar("https://kubernetes-charts.storage.googleapis.com/mariadb-0.6.3.tgz", dir)
				if err != nil {
					t.Fatal(err)
				}
			},
			Config: fmt.Sprintf(`
			resource "helm_release" "test" {
				name      = %q
				namespace = %q
				chart     = %q
				set {
					name = "persistence.enabled"
					value = "false" # persistent volumes are giving non-related issues when testing
				}
			}
		`, testNamespace, testResourceName, chartPath),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "version", "0.6.3"),
			),
		}},
	})
}

func testAccHelmReleaseConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
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
	`, resource, name, ns, version)
}

func testAccHelmReleaseConfigValues(resource, ns, name, chart string, values []string) string {
	vals := make([]string, len(values))
	for i, v := range values {
		vals[i] = strconv.Quote(v)
	}
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name      = %q
			namespace = %q
			chart     = %q
			values    = [ %s ]
		}
	`, resource, name, ns, chart, strings.Join(vals, ","))
}

func TestGetValues(t *testing.T) {
	d := resourceRelease().Data(nil)
	d.Set("values", []string{
		"foo: bar\nbaz: corge",
		"first: present\nbaz: grault",
		"second: present\nbaz: uier",
	})
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
	if base["first"] != "present" {
		t.Fatalf("error merging values from file, expected value file %q not read", "testdata/get_values_first.yaml")
	}
	if base["second"] != "present" {
		t.Fatalf("error merging values from file, expected value file %q not read", "testdata/get_values_second.yaml")
	}
	if base["baz"] != "uier" {
		t.Fatalf("error merging values from file, expected %q, got %q", "uier", base["baz"])
	}
}

func testAccHelmReleaseConfigRepository(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "stable_repo" {
			name = "stable-repo"
			url  = "https://kubernetes-charts.storage.googleapis.com"
		}

		resource "helm_release" "test" {
			name       = %q
			namespace  = %q
			repository = "${helm_repository.stable_repo.metadata.0.name}"
			chart      = "coredns"
		}
	`, name, ns)
}

func testAccHelmReleaseConfigRepositoryURL(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "test" {
			name       = %q
			namespace  = %q
			repository = "https://kubernetes-charts.storage.googleapis.com"
			chart      = "coredns"
		}
	`, name, ns)
}

func testAccCheckHelmReleaseDestroy(s *terraform.State) error {
	// Fix for a flaky test
	// Helm doesn't instantly delete it's releases causing this test to fail if not waited for a small period of time.
	// TODO: improve the workaround
	time.Sleep(30 * time.Second)

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

	if res == nil {
		return nil
	}

	if err != nil {
		return err
	}

	for _, r := range res.Releases {
		if r.Name == testResourceName {
			return fmt.Errorf("found %q release", testResourceName)
		}
	}

	if res.Count != 0 {
		return fmt.Errorf("%q namespace should be empty", testNamespace)
	}

	return nil
}

func testAccHelmReleaseConfigLocalDir(ns, name, path string) string {
	return fmt.Sprintf(`
		resource "helm_release" "test" {
			name      = %q
			namespace = %q
			chart     = %q
		}
	`, name, ns, path)
}

func downloadTar(url, dst string) error {
	rsp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	return unTar(dst, rsp.Body)
}

func unTar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	defer gzr.Close()
	if err != nil {
		return err
	}
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			dir := filepath.Dir(target)
			if _, err := os.Stat(dir); err != nil {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}
