package helm

import (
	"fmt"
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
	"github.com/pkg/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestAccResourceRelease_basic(t *testing.T) {
	name := fmt.Sprintf("test-basic-%s", acctest.RandString(10))
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "7.1.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "mariadb"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.1.0"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "7.1.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.1.0"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
				PreCheck:     func() { testAccPreCheck(t, namespace) },
				Providers:    testAccProviders,
				CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
				Steps: []resource.TestStep{{
					Config: testAccHelmReleaseConfigBasic(name, namespace, name, "7.1.0"),
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
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "7.0.1"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.0.1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "7.1.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.1.0"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "kibana", "3.2.5", []string{""},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{}"),
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
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "kibana", "3.2.5", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"bar\"}"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "kibana", "3.2.5", []string{"foo: baz"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"baz\"}"),
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
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name,
				"kibana", "3.2.5", []string{"foo: bar"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"bar\"}"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name,
				"kibana", "3.2.5", []string{"foo: bar", "foo: baz"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"baz\"}"),
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
		PreCheck:  func() { testAccPreCheck(t, namespace) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepository(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepository(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
		PreCheck:  func() { testAccPreCheck(t, namespace) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryDatasource(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryDatasource(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
			testAccPreCheck(t, namespace)
			testAccPreCheckHelmRepositoryDestroy(t, repo1)
			testAccPreCheckHelmRepositoryDestroy(t, repo2)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryMultipleDatasource(repo1, repo2, testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
		PreCheck:  func() { testAccPreCheck(t, namespace) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
				resource.TestCheckResourceAttrSet("helm_release.test", "version"),
			),
		}, {
			Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
		repository  = "https://kubernetes-charts.storage.googleapis.com"

		chart       = "nginx-ingress"
		set {
			name = "controller.name"
			value = "invalid-$%!-character-for-k8s-label"
		}
	}
	`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config:             malformed,
			ExpectError:        regexp.MustCompile("invalid resource name"),
			ExpectNonEmptyPlan: true,
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "7.1.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.1.0"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
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
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "mariadb", "7.1.0",
				[]string{"master:\n  persistence:\n    enabled: false",
					"replication:\n  enabled: false",
					"master:\n  service:\n    annotations: {}"},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "mariadb", "7.1.0",
				[]string{"master:\n  persistence:\n    enabled: true",
					"replication:\n  enabled: false",
					"master:\n  service:\n    annotations: {}"},
			),
			ExpectError:        regexp.MustCompile("forbidden"),
			ExpectNonEmptyPlan: true,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "status", "FAILED"),
			),
		}, {
			Config: testAccHelmReleaseConfigValues(
				testResourceName, namespace, name, "mariadb", "7.1.0",
				[]string{
					"master:\n  persistence:\n    enabled: true",
					"replication:\n  enabled: false",
					"master:\n  service:\n    annotations: {}"},
			),
			ExpectError:        regexp.MustCompile("forbidden"),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccResourceRelease_updateVersionFromRelease(t *testing.T) {
	namespace := fmt.Sprintf("%s-%s", testNamespace, acctest.RandString(10))
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t, namespace) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, testResourceName, "7.0.1"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.0.1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "version", "7.0.1"),
			),
		}, {
			Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, testResourceName, "7.1.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "7.1.0"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "version", "7.1.0"),
			),
		}},
	})
}

func testAccHelmReleaseConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name       = %q
			namespace  = %q
			repository = "https://kubernetes-charts.storage.googleapis.com"
  			chart      = "mariadb"
			version    = %q

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

func testAccHelmReleaseConfigValues(resource, ns, name, chart, version string, values []string) string {
	vals := make([]string, len(values))
	for i, v := range values {
		vals[i] = strconv.Quote(v)
	}
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name       = %q
			namespace  = %q
			repository = "https://kubernetes-charts.storage.googleapis.com"
			chart      = %q
			version    = %q
			values     = [ %s ]
		}
	`, resource, name, ns, chart, version, strings.Join(vals, ","))
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

	if err != nil {
		t.Fatalf("error parsing returned yaml: %s", err)
		return
	}

	if values["foo"] != "qux" {
		t.Fatalf("error merging values, expected %q, got %q", "qux", values["foo"])
	}
	if values["first"] != "present" {
		t.Fatalf("error merging values from file, expected value file %q not read", "testdata/get_values_first.yaml")
	}
	if values["second"] != "present" {
		t.Fatalf("error merging values from file, expected value file %q not read", "testdata/get_values_second.yaml")
	}
	if values["baz"] != "uier" {
		t.Fatalf("error merging values from file, expected %q, got %q", "uier", values["baz"])
	}
}

func testAccHelmReleaseConfigRepository(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_repository" "stable_repo" {
			name = "stable-repo-resource"
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
			name = "stable-repo-data"
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

	rc := settings.RepositoryConfig

	r, err := repo.LoadFile(rc)

	if isNotExist(err) || len(r.Repositories) == 0 || !r.Remove(name) {
		t.Log(fmt.Sprintf("no repo named %q found, nothing to do", name))
		return
	}

	if err := r.WriteFile(rc, 0644); err != nil {
		t.Fatalf("Failed to write repositories file: %s", err)
	}

	if err := removeRepoCache(settings.RepositoryCache, name); err != nil {
		t.Fatalf("Failed to remove repository cache: %s", err)
	}

	fmt.Fprintf(os.Stdout, "%q has been removed from your repositories\n", name)
	t.Log(fmt.Sprintf("%q has been removed from your repositories\n", name))
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}

func removeRepoCache(root, name string) error {
	idx := filepath.Join(root, helmpath.CacheIndexFile(name))
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove index file %s", idx)
	}
	return os.Remove(idx)
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

		actionConfig, err := m.(*Meta).GetHelmConfiguration(namespace)
		if err != nil {
			return err
		}

		client := action.NewList(actionConfig)
		res, err := client.Run()

		if res == nil {
			return nil
		}

		if err != nil {
			return err
		}

		for _, r := range res {
			if r.Name == testResourceName {
				return fmt.Errorf("found %q release", testResourceName)
			}

			if r.Namespace == namespace {
				return fmt.Errorf("%q namespace should be empty", namespace)
			}
		}

		return nil
	}
}

func deleteNamespace(t *testing.T, namespace string) {
	// Nothing to cleanup with unit test
	if os.Getenv("TF_ACC") == "" {
		t.Log("TF_ACC Not Set")
		return
	}

	m := testAccProvider.Meta()
	if m == nil {
		t.Fatal("provider not properly initialized")
	}

	debug("[DEBUG] Deleting namespace %q", namespace)
	gracePeriodSeconds := int64(0)
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}
	err := client.CoreV1().Namespaces().Delete(namespace, deleteOptions)
	if err != nil {
		t.Fatalf("An error occurred while deleting namespace %q: %q", namespace, err)
	}
}
