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

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	yaml "gopkg.in/yaml.v2"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/repo"
)

func TestAccResourceRelease_basic(t *testing.T) {
	name := fmt.Sprintf("test-basic-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "mariadb"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "0.6.2"),
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
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	// This test case cannot be parallelized by using `resource.ParallelTest()` as calling `t.Parallel()` more than
	// once in a single test case resuls in the following error:
	// `panic: testing: t.Parallel called multiple times`
	t.Parallel()

	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(name string) {
			defer wg.Done()
			resource.Test(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
				Steps: []resource.TestStep{{
					Config: testAccHelmReleaseConfigBasic(name, namespace, name, "0.6.2"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							fmt.Sprintf("helm_release.%s", name), "metadata.0.name", name,
						),
					),
				}},
			})
		}(fmt.Sprintf("concurrent-%d-%s", i, acctest.RandString(10)))
	}

	wg.Wait()
}

func TestAccResourceRelease_update(t *testing.T) {
	name := fmt.Sprintf("test-update-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "0.6.2"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_emptyValuesList(t *testing.T) {
	name := fmt.Sprintf("test-empty-values-list-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "stable/kibana", []string{""},
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
	name := fmt.Sprintf("test-update-values-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "stable/kibana", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: bar\n"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "stable/kibana", []string{"foo: baz"},
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
	name := fmt.Sprintf("test-update-multiple-values-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name,
				"stable/kibana", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "foo: bar\n"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name,
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
	name := fmt.Sprintf("test-repository-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepository(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepository(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_repositoryDatasource(t *testing.T) {
	name := fmt.Sprintf("test-repository-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryDatasource(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryDatasource(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_repositoryMultipleDatasources(t *testing.T) {
	name := fmt.Sprintf("test-repository-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	repo1 := "test-acc-repo-1"
	repo2 := "test-acc-repo-2"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckHelmRepositoryDestroy(t, repo1)
			testAccPreCheckHelmRepositoryDestroy(t, repo2)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryMultipleDatasource(repo1, repo2, testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}},
	})
}

func TestAccResourceRelease_repository_url(t *testing.T) {
	name := fmt.Sprintf("test-repository-url-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
				resource.TestCheckResourceAttrSet("helm_release.test", "version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
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
	name := fmt.Sprintf("test-update-after-fail-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

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

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config:             malformed,
			ExpectError:        regexp.MustCompile("failed"),
			ExpectNonEmptyPlan: true,
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "0.6.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "0.6.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}},
	})
}

func TestAccResourceRelease_updateExistingFailed(t *testing.T) {
	name := fmt.Sprintf("test-update-existing-failed-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "stable/mariadb",
				[]string{"master:\n  persistence:\n    enabled: false", "replication:\n  enabled: false"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "DEPLOYED"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "stable/mariadb",
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
				testResourceName, namespace, name, "stable/mariadb",
				[]string{"master:\n  persistence:\n    enabled: true", "replication:\n  enabled: false"},
			),
			ExpectError:        regexp.MustCompile("forbidden"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccResourceRelease_updateVersionFromRelease(t *testing.T) {
	name := fmt.Sprintf("test-update-version-from-release-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	chartPath := filepath.Join(dir, "mariadb")
	defer os.RemoveAll(dir)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			PreConfig: func() {
				err := downloadTar("https://kubernetes-charts.storage.googleapis.com/mariadb-0.6.2.tgz", dir)
				if err != nil {
					t.Fatal(err)
				}
			},
			Config: fmt.Sprintf(`
			resource "helm_release" %q {
				name      = %q
				namespace = %q
				chart     = %q
				set {
					name = "persistence.enabled"
					value = "false" # persistent volumes are giving non-related issues when testing
				}
			}
		`, testResourceName, name, namespace, chartPath),
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
			resource "helm_release" %q {
				name      = %q
				namespace = %q
				chart     = %q
				set {
					name = "persistence.enabled"
					value = "false" # persistent volumes are giving non-related issues when testing
				}
			}
		`, testResourceName, name, namespace, chartPath),
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

func testAccHelmReleaseConfigRepository(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "stable_repo" {
			name = "stable-repo"
			url  = "https://kubernetes-charts.storage.googleapis.com"
		}

		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = "${helm_repository.stable_repo.metadata.0.name}"
			chart      = "coredns"
		}
	`, resource, name, ns)
}

func testAccHelmReleaseConfigRepositoryDatasource(resource, ns, name string) string {
	return fmt.Sprintf(`
		data "helm_repository" "stable_repo" {
			name = "stable-repo"
			url  = "https://kubernetes-charts.storage.googleapis.com"
		}

		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = "${data.helm_repository.stable_repo.metadata.0.name}"
			chart      = "coredns"
		}
	`, resource, name, ns)
}

func testAccHelmReleaseConfigRepositoryMultipleDatasource(repo1, repo2, resource, ns, name string) string {
	return fmt.Sprintf(`
		data "helm_repository" "stable_repo" {
			name = %q
			url  = "https://kubernetes-charts.storage.googleapis.com"
		}

		data "helm_repository" "stable_repo_2" {
			name = %q
			url  = "https://kubernetes-charts.storage.googleapis.com"
		}

		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = "${data.helm_repository.stable_repo.metadata.0.name}"
			chart      = "coredns"
		}

		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = "${data.helm_repository.stable_repo_2.metadata.0.name}"
			chart      = "coredns"
		}
	`, repo1, repo2, resource, name, ns, resource+"_2", name+"-2", ns)
}

func testAccHelmReleaseConfigRepositoryURL(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = "https://kubernetes-charts.storage.googleapis.com"
			chart      = "coredns"
		}
	`, resource, name, ns)
}

func testAccPreCheckHelmRepositoryDestroy(t *testing.T, name string) {
	settings := testAccProvider.Meta().(*Meta).Settings

	repoFile := settings.Home.RepositoryFile()
	r, err := repo.LoadRepositoriesFile(repoFile)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Remove(name) {
		t.Log(fmt.Sprintf("no repo named %q found, nothing to do", name))
		return
	}
	if err := r.WriteFile(repoFile, 0644); err != nil {
		t.Fatalf("Failed to write repositories file: %s", err)
	}

	if _, err := os.Stat(settings.Home.CacheIndex(name)); err == nil {
		err = os.Remove(settings.Home.CacheIndex(name))
		if err != nil {
			t.Fatalf("Failed to remove repository cache: %s", err)
		}
	}

	t.Log(fmt.Sprintf("%q has been removed from your repositories\n", name))
}

func testAccCheckHelmReleaseDestroy(namespace string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Fix for a flaky test
		// Helm doesn't instantly delete its releases causing this test to fail if not waited for a small period of time.
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
			helm.ReleaseListNamespace(namespace),
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
			return fmt.Errorf("%q namespace should be empty", namespace)
		}

		return nil
	}
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

func deleteNamespace(t *testing.T, namespace string) {
	// Nothing to cleanup with unit test
	if os.Getenv("TF_ACC") == "" {
		return
	}

	m := testAccProvider.Meta()
	if m == nil {
		t.Fatal("provider not properly initialized")
	}

	debug("[DEBUG] Deleting namespace %q", namespace)
	gracePeriodSeconds := int64(0)
	deleteOptions := meta_v1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}
	err := m.(*Meta).K8sClient.CoreV1().Namespaces().Delete(namespace, &deleteOptions)
	if err != nil {
		t.Fatalf("An error occurred while deleting namespace %q: %q", namespace, err)
	}
}
