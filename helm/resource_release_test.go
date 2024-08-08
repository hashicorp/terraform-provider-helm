// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/pkg/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestAccResourceRelease_basic(t *testing.T) {
	name := randName("basic")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "description", "Test"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.app_version", "1.19.5"),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.first_deployed", regexp.MustCompile("[0-9]+")),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.last_deployed", regexp.MustCompile("[0-9]+")),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.notes", regexp.MustCompile(`^1. Get the application URL by running these commands:\n  export POD_NAME=.*`)),
				),
			},
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "description", "Test"),
				),
			},
		},
	})
}

// NOTE this is a regression test for: https://github.com/hashicorp/terraform-provider-helm/issues/1236
func TestAccResourceRelease_emptyVersion(t *testing.T) {
	name := randName("basic")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resourceName := "helm_release.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigEmptyVersion(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr(resourceName, "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.version", "2.0.0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.app_version", "1.19.5"),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.first_deployed", regexp.MustCompile("[0-9]+")),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.last_deployed", regexp.MustCompile("[0-9]+")),
					resource.TestMatchResourceAttr("helm_release.test", "metadata.0.notes", regexp.MustCompile(`^1. Get the application URL by running these commands:\n  export POD_NAME=.*`)),
				),
			},
		},
	})
}

// "upgrade_install" without a previously installed release (effectively equivalent to TestAccResourceRelease_basic)
func TestAccResourceRelease_upgrade_with_install_coldstart(t *testing.T) {
	name := randName("basic")
	namespace := createRandomNamespace(t)
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),

		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigWithUpgradeStrategy(testResourceName, namespace, name, "1.2.3", true),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "description", "Test"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.app_version", "1.19.5"),
			),
		}, {
			Config: testAccHelmReleaseConfigWithUpgradeStrategy(testResourceName, namespace, name, "1.2.3", true),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				resource.TestCheckResourceAttr("helm_release.test", "description", "Test"),
			),
		}},
	})
}

// "upgrade" install wherein we pretend that someone else (e.g. a CI/CD system) has done the first install
func TestAccResourceRelease_upgrade_with_install_warmstart(t *testing.T) {
	name := randName("basic")
	namespace := createRandomNamespace(t)
	// Delete namespace automatically created by helm after checks
	defer deleteNamespace(t, namespace)

	// preinstall the first revision of our chart directly via the helm CLI
	args := []string{"install", "-n", namespace, "--create-namespace", name, "./test-chart-1.2.3.tgz"}
	cmd := exec.Command("helm", args...)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("could not preinstall helm chart: %v -- %s", err, stdout)
	}

	// upgrade-install on top of the existing release, creating a new revision
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{{
			Config: testAccHelmReleaseConfigWithUpgradeStrategyWarmstart(namespace, name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
				resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
			)}},
	})
}

func TestAccResourceRelease_import(t *testing.T) {
	name := randName("import")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
			{
				Config:                  testAccHelmReleaseConfigBasic("imported", namespace, "import", "1.2.3"),
				ImportStateId:           fmt.Sprintf("%s/%s", namespace, name),
				ResourceName:            "helm_release.imported",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"set", "set.#", "repository"},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.imported", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.imported", "metadata.0.version", "1.2.0"),
					resource.TestCheckResourceAttr("helm_release.imported", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.imported", "description", "Test"),
					resource.TestCheckNoResourceAttr("helm_release.imported", "repository"),

					// Default values
					resource.TestCheckResourceAttr("helm_release.imported", "verify", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "timeout", "300"),
					resource.TestCheckResourceAttr("helm_release.imported", "wait", "true"),
					resource.TestCheckResourceAttr("helm_release.imported", "wait_for_jobs", "true"),
					resource.TestCheckResourceAttr("helm_release.imported", "pass_credentials", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "disable_webhooks", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "atomic", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "render_subchart_notes", "true"),
					resource.TestCheckResourceAttr("helm_release.imported", "disable_crd_hooks", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "force_update", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "reset_values", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "reuse_values", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "recreate_pods", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "max_history", "0"),
					resource.TestCheckResourceAttr("helm_release.imported", "skip_crds", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "cleanup_on_fail", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "dependency_update", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "replace", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "disable_openapi_validation", "false"),
					resource.TestCheckResourceAttr("helm_release.imported", "create_namespace", "false"),
				),
			},
		},
	})
}

func TestAccResourceRelease_inconsistentVersionRegression(t *testing.T) {
	// NOTE this is a regression test, see: https://github.com/hashicorp/terraform-provider-helm/issues/1150
	name := randName("basic")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "v1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "version", "1.2.3"),
				),
			},
			{
				Config:   testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "v1.2.3"),
				PlanOnly: true,
			},
		},
	})
}

func TestAccResourceRelease_multiple_releases(t *testing.T) {
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	const resourceCount = 10
	basicHelmRelease := func(index int, namespace string) (string, resource.TestCheckFunc) {
		randomKey := acctest.RandString(10)
		randomValue := acctest.RandString(10)
		resourceName := fmt.Sprintf("test_%d", index)
		releaseName := fmt.Sprintf("test-%d", index)
		return fmt.Sprintf(`
			resource "helm_release" %q {
				name        = %q
				namespace   = %q
				repository  = %q
				chart       = "test-chart"

				set {
					name = %q
					value = %q
				}
			}`, resourceName, releaseName, namespace, testRepositoryURL, randomKey, randomValue),
			resource.TestCheckResourceAttr(
				fmt.Sprintf("helm_release.%s", resourceName), "metadata.0.name", releaseName,
			)
	}
	config := ""
	var resourceChecks []resource.TestCheckFunc
	for i := 0; i < resourceCount; i++ {
		releaseConfig, releaseCheck := basicHelmRelease(i, namespace)
		resourceChecks = append(resourceChecks, releaseCheck)
		config += releaseConfig
	}
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resourceChecks...,
				),
			},
		},
	})
}

func TestAccResourceRelease_concurrent(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func(name string) {
			defer wg.Done()
			namespace := createRandomNamespace(t)
			defer deleteNamespace(t, namespace)
			resource.Test(t, resource.TestCase{
				PreCheck: func() { testAccPreCheck(t) },
				ProviderFactories: map[string]func() (*schema.Provider, error){
					"helm": func() (*schema.Provider, error) {
						return Provider(), nil
					},
				},
				CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
				Steps: []resource.TestStep{
					{
						Config: testAccHelmReleaseConfigBasic(name, namespace, name, "1.2.3"),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(
								fmt.Sprintf("helm_release.%s", name), "metadata.0.name", name,
							),
						),
					},
				},
			})
		}(fmt.Sprintf("concurrent-%d-%s", i, acctest.RandString(10)))
	}
	wg.Wait()
}

func TestAccResourceRelease_update(t *testing.T) {
	name := randName("update")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "version", "1.2.3"),
				),
			},
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "2.0.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "2.0.0"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "version", "2.0.0"),
				),
			},
		},
	})
}

func TestAccResourceRelease_emptyValuesList(t *testing.T) {
	name := randName("test-empty-values-list")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3", []string{""},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{}"),
				),
			},
		},
	})
}

func TestAccResourceRelease_updateValues(t *testing.T) {
	name := randName("test-update-values")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3", []string{"foo: bar"},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"bar\"}"),
				),
			},
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3", []string{"foo: baz"},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"baz\"}"),
				),
			},
		},
	})
}

func TestAccResourceRelease_cloakValues(t *testing.T) {
	name := randName("test-update-values")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigSensitiveValue(
					testResourceName, namespace, name, "test-chart", "1.2.3", "foo", "bar",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values",
						"{\"foo\":\"(sensitive value)\"}"),
				),
			},
		},
	})
}

func TestAccResourceRelease_updateMultipleValues(t *testing.T) {
	name := randName("test-update-multiple-values")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name,
					"test-chart", "1.2.3", []string{"foo: bar"},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"bar\"}"),
				),
			},
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name,
					"test-chart", "1.2.3", []string{"foo: bar", "foo: baz"},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.values", "{\"foo\":\"baz\"}"),
				),
			},
		},
	})
}

func TestAccResourceRelease_repository_url(t *testing.T) {
	name := randName("test-repository-url")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
					resource.TestCheckResourceAttrSet("helm_release.test", "version"),
				),
			},
			{
				Config: testAccHelmReleaseConfigRepositoryURL(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttrSet("helm_release.test", "metadata.0.version"),
					resource.TestCheckResourceAttrSet("helm_release.test", "version"),
				),
			},
		},
	})
}

func TestAccResourceRelease_updateAfterFail(t *testing.T) {
	name := randName("test-update-after-fail")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	malformed := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = %q
		namespace   = %q
		repository  = %q
		chart       = "test-chart"

		set {
			name = "serviceAccount.name"
			value = "invalid-$%%!-character"
		}

		set {
			name = "service.type"
			value = "ClusterIP"
		}
	}`, name, namespace, testRepositoryURL)

	fixed := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = %q
		namespace   = %q
		repository  = %q
		chart       = "test-chart"

		set {
			name = "serviceAccount.name"
			value = "valid-name"
		}

		set {
			name = "service.type"
			value = "ClusterIP"
		}
	}`, name, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:             malformed,
				ExpectError:        regexp.MustCompile("invalid resource name"),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fixed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func TestAccResourceRelease_updateExistingFailed(t *testing.T) {
	name := randName("test-update-existing-failed")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3",
					[]string{"serviceAccount:\n  name: valid-name"},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3",
					[]string{"service:\n  type: invalid%-$type"},
				),
				ExpectError:        regexp.MustCompile("Unsupported value"),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "status", "FAILED"),
				),
			},
			{
				Config: testAccHelmReleaseConfigValues(
					testResourceName, namespace, name, "test-chart", "1.2.3",
					[]string{"service:\n  type: invalid%-$type"},
				),
				ExpectError:        regexp.MustCompile("Unsupported value"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceRelease_updateSetValue(t *testing.T) {
	name := randName("test-update-set-value")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	// Ensure that 'set' 'type' arguments don't disappear when updating a 'set' 'value' argument.
	// use  checkResourceAttrExists rather than testCheckResourceAttrSet as the latter also checks if the value is not ""
	// and the default for 'type' is an empty string when not explicitly set.
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigSet(
					testResourceName, namespace, name, "1.2.3", "initial",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkResourceAttrExists("helm_release.test", "set.0.type"),
					checkResourceAttrExists("helm_release.test", "set.1.type"),
				),
			},
			{
				Config: testAccHelmReleaseConfigSet(
					testResourceName, namespace, name, "1.2.3", "updated",
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkResourceAttrExists("helm_release.test", "set.0.type"),
					checkResourceAttrExists("helm_release.test", "set.1.type"),
				),
			},
		},
	})
}

func TestAccResourceRelease_validation(t *testing.T) {
	invalidName := "this-helm-release-name-is-longer-than-53-characters-long"
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:      testAccHelmReleaseConfigBasic(testResourceName, namespace, invalidName, "1.2.3"),
				ExpectError: regexp.MustCompile("expected length of name to be in the range.*"),
			},
		},
	})
}

func checkResourceAttrExists(name, key string) resource.TestCheckFunc {
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

		if _, ok := is.Attributes[key]; ok {
			return nil
		}
		return fmt.Errorf("%s: Attribute '%s' expected to be set", name, key)
	}
}

func TestAccResourceRelease_postrender(t *testing.T) {
	// TODO: Add Test Fixture to return real YAML here

	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigPostrender(testResourceName, namespace, testResourceName, "echo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
			{
				Config:      testAccHelmReleaseConfigPostrender(testResourceName, namespace, testResourceName, "echo", "this will not work!", "Wrong", "Code"),
				ExpectError: regexp.MustCompile("error validating data"),
			},
			{
				Config:      testAccHelmReleaseConfigPostrender(testResourceName, namespace, testResourceName, "foobardoesnotexist"),
				ExpectError: regexp.MustCompile("unable to find binary"),
			},
			{
				Config: testAccHelmReleaseConfigPostrender(testResourceName, namespace, testResourceName, "true", "Hello", "World", "!"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func TestAccResourceRelease_namespaceDoesNotExist(t *testing.T) {
	name := randName("test-namespace-does-not-exist")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	broken := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = %q
		namespace   = "does-not-exist"
		repository  = %q
		chart       = "test-chart"
	}`, name, testRepositoryURL)

	fixed := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = %q
		namespace   = %q
		repository  = %q
		chart       = "test-chart"
	}`, name, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:             broken,
				ExpectError:        regexp.MustCompile(`failed to create: namespaces "does-not-exist" not found`),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fixed,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func TestAccResourceRelease_invalidName(t *testing.T) {
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	broken := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = "1nva&lidname$"
		namespace   = %q
		repository  = %q
		chart       = "test-chart"
	}`, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:             broken,
				ExpectError:        regexp.MustCompile("invalid release name"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceRelease_createNamespace(t *testing.T) {
	name := randName("create-namespace")
	namespace := randName("helm-created-namespace")
	defer deleteNamespace(t, namespace)

	config := fmt.Sprintf(`
	resource "helm_release" "test" {
		name             = %q
		namespace        = %q
		repository       = %q
		chart            = "test-chart"
		create_namespace = true
	}`, name, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func TestAccResourceRelease_LocalVersion(t *testing.T) {
	name := randName("create-namespace")
	namespace := randName("helm-created-namespace")
	defer deleteNamespace(t, namespace)

	// this test insures that the version is not changed when using a local chart

	config1 := fmt.Sprintf(`
	resource "helm_release" "test" {
		name             = %q
		namespace        = %q
		chart            = "testdata/charts/test-chart"
		create_namespace = true
	}`, name, namespace)

	config2 := fmt.Sprintf(`
	resource "helm_release" "test" {
		name             = %q
		namespace        = %q
		version 		 = "1.0.0"
		chart            = "testdata/charts/test-chart"
		create_namespace = true
	}`, name, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				),
			},
			{
				Config: config2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				),
			},
		},
	})
}

func testAccHelmReleaseConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
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
		}
	`, resource, name, ns, testRepositoryURL, version)
}

func testAccHelmReleaseConfigEmptyVersion(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
  			chart       = "test-chart"
			version     = ""
		}
	`, resource, name, ns, testRepositoryURL)
}

func testAccHelmReleaseConfigWithUpgradeStrategy(resource, ns, name, version string, upgrade_install bool) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = "%s"
  			chart       = "test-chart"
			version     = %q

			upgrade_install = %t

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
	`, resource, name, ns, testRepositoryURL, version, upgrade_install)
}

func testAccHelmReleaseConfigWithUpgradeStrategyWarmstart(ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "test" {
 			name        = %q
			namespace   = %q
			description = "Test"
  			chart       = "./test-chart-1.2.3.tgz"
			version     = "0.1.0"

			upgrade_install = true
			
			set {
				name  = "foo"
				value = "bar"
			}
		}
	`, name, ns)
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
			repository = %q
			chart      = %q
			version    = %q
			values     = [ %s ]
		}
	`, resource, name, ns, testRepositoryURL, chart, version, strings.Join(vals, ","))
}

func testAccHelmReleaseConfigSensitiveValue(resource, ns, name, chart, version string, key, value string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name       = %q
			namespace  = %q
			repository = %q
			chart      = %q
			version    = %q
			set_sensitive {
				name  = %q
				value = %q
			  }
		}
	`, resource, name, ns, testRepositoryURL, chart, version, key, value)
}

func testAccHelmReleaseConfigSet(resource, ns, name, version, setValue string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			description = "Test"
			repository  = %q
  			chart       = "test-chart"
			version     = %q

			set {
				name = "foo"
				value = %q
			}

			set {
				name = "fizz"
				value = 1337
			}
		}
	`, resource, name, ns, testRepositoryURL, version, setValue)
}

func TestGetValues(t *testing.T) {
	d := resourceRelease().Data(nil)
	err := d.Set("values", []string{
		"foo: bar\nbaz: corge",
		"first: present\nbaz: grault",
		"second: present\nbaz: uier",
	})
	if err != nil {
		t.Fatalf("error setting values: %v", err)
	}
	err = d.Set("set", []interface{}{
		map[string]interface{}{"name": "foo", "value": "qux"},
		map[string]interface{}{"name": "int", "value": "42"},
	})
	if err != nil {
		t.Fatalf("error setting values: %v", err)
	}

	values, err := getValues(d)
	if err != nil {
		t.Fatalf("error getValues: %s", err)
		return
	}

	if values["foo"] != "qux" {
		t.Fatalf("error merging values, expected %q, got %q", "qux", values["foo"])
	}
	if values["int"] != int64(42) {
		t.Fatalf("error merging values, expected %s, got %s", "42", values["int"])
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

func TestGetValuesString(t *testing.T) {
	d := resourceRelease().Data(nil)
	err := d.Set("set", []interface{}{
		map[string]interface{}{"name": "foo", "value": "42", "type": "string"},
	})
	if err != nil {
		t.Fatalf("error setting values: %s", err)
		return
	}

	values, err := getValues(d)
	if err != nil {
		t.Fatalf("error getValues: %s", err)
		return
	}

	if values["foo"] != "42" {
		t.Fatalf("error merging values, expected %q, got %s", "42", values["foo"])
	}
}

func TestUseChartVersion(t *testing.T) {

	type test struct {
		chartPath       string
		repositoryURL   string
		useChartVersion bool
	}

	tests := []test{
		// when chart is a local directory
		{chartPath: "./testdata/charts/test-chart", repositoryURL: "", useChartVersion: true},
		// when the repo is a local directory
		{chartPath: "testchart", repositoryURL: "./testdata/charts", useChartVersion: true},
		// when the repo is a repository URL
		{chartPath: "", repositoryURL: "https://charts.bitnami.com/bitnami", useChartVersion: false},
		// when chartPath is chart name and repo is repository URL
		{chartPath: "redis", repositoryURL: "https://charts.bitnami.com/bitnami", useChartVersion: false},
		// when the chart is a URL to an .tgz file, any other url link that is not a .tgz file will not reach useChartVersion
		{chartPath: "https://charts.bitnami.com/bitnami/redis-10.7.16.tgz", repositoryURL: "", useChartVersion: true},
		// when the repo is an OCI registry
		{chartPath: "redis", repositoryURL: "oci://registry-1.docker.io/bitnamicharts", useChartVersion: false},
		// when the chart is a URL to an OCI registry
		{chartPath: "oci://registry-1.docker.io/bitnamicharts/redis", repositoryURL: "", useChartVersion: false},
	}

	for i, tc := range tests {
		if result := useChartVersion(tc.chartPath, tc.repositoryURL); result != tc.useChartVersion {
			t.Fatalf("[%v] error in useChartVersion; expected useChartVersion(%q, %q) == %v, got %v", i, tc.chartPath, tc.repositoryURL, tc.useChartVersion, result)
		}
	}
}

func TestGetListValues(t *testing.T) {
	d := resourceRelease().Data(nil)
	testValue := []string{"1", "2", "3"}
	err := d.Set("set_list", []interface{}{
		map[string]interface{}{"name": "foo", "value": testValue},
	})
	if err != nil {
		t.Fatalf("error setting values: %s", err)
		return
	}

	values, err := getValues(d)
	if err != nil {
		t.Fatalf("error getValues: %s", err)
		return
	}

	for i, v := range testValue {
		val, _ := strconv.ParseInt(v, 10, 64)
		if values["foo"].([]interface{})[i] != val {
			t.Fatalf("error merging values, expected value of %v, got %v", v, values["foo"].([]interface{})[i])
		}
	}
}

func TestCloakSetValues(t *testing.T) {
	d := resourceRelease().Data(nil)
	err := d.Set("set_sensitive", []interface{}{
		map[string]interface{}{"name": "foo", "value": "42"},
	})
	if err != nil {
		t.Fatalf("error setting values: %v", err)
	}

	values := map[string]interface{}{
		"foo": "foo",
	}

	cloakSetValues(values, d)
	if values["foo"] != sensitiveContentValue {
		t.Fatalf("error cloak values, expected %q, got %s", sensitiveContentValue, values["foo"])
	}
}

func TestCloakSetValuesNested(t *testing.T) {
	d := resourceRelease().Data(nil)
	err := d.Set("set_sensitive", []interface{}{
		map[string]interface{}{"name": "foo.qux.bar", "value": "42"},
	})
	if err != nil {
		t.Fatalf("error setting values: %v", err)
	}

	qux := map[string]interface{}{
		"bar": "bar",
	}

	values := map[string]interface{}{
		"foo": map[string]interface{}{
			"qux": qux,
		},
	}

	cloakSetValues(values, d)
	if qux["bar"] != sensitiveContentValue {
		t.Fatalf("error cloak values, expected %q, got %s", sensitiveContentValue, qux["bar"])
	}
}

func TestCloakSetValuesNotMatching(t *testing.T) {
	d := resourceRelease().Data(nil)
	err := d.Set("set_sensitive", []interface{}{
		map[string]interface{}{"name": "foo.qux.bar", "value": "42"},
	})
	if err != nil {
		t.Fatalf("error setting values: %v", err)
	}

	values := map[string]interface{}{
		"foo": "42",
	}

	cloakSetValues(values, d)
	if values["foo"] != "42" {
		t.Fatalf("error cloak values, expected %q, got %s", "42", values["foo"])
	}
}

func testAccHelmReleaseConfigRepositoryURL(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" %q {
			name       = %q
			namespace  = %q
			repository = %q
			chart      = "test-chart"
		}
	`, resource, name, ns, testRepositoryURL)
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

	_, err = fmt.Fprintf(os.Stdout, "%q has been removed from your repositories\n", name)
	if err != nil {
		t.Fatalf("error printing stdout: %v", err)
	}

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

func testAccCheckHelmReleaseDependencyUpdate(namespace string, name string, expectedResources int) resource.TestCheckFunc {
	// NOTE this is a regression test to check that a charts dependencies have not been
	// deleted from the manifest on update.

	return func(s *terraform.State) error {
		m := testAccProvider.Meta()
		if m == nil {
			return fmt.Errorf("provider not properly initialized")
		}

		actionConfig, err := m.(*Meta).GetHelmConfiguration(namespace)
		if err != nil {
			return err
		}

		client := action.NewGet(actionConfig)
		res, err := client.Run(name)

		if res == nil {
			return fmt.Errorf("release %q not found", name)
		}

		if err != nil {
			return err
		}

		resources := releaseutil.SplitManifests(res.Manifest)
		if len(resources) != expectedResources {
			return fmt.Errorf("expected %v resources but got %v", expectedResources, len(resources))
		}

		return nil
	}
}

func testAccCheckHelmReleaseDestroy(namespace string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
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

func testAccHelmReleaseConfigPostrender(resource, ns, name, binaryPath string, args ...string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
  			chart       = "test-chart"
			version     = "1.2.3"

			postrender {
				binary_path = %q
				args = %s
			}

			set {
				name = "serviceAccount.create"
				value = false
			}
			set {
				name = "service.port"
				value = 1337
			}
		}
	`, resource, name, ns, testRepositoryURL, binaryPath, fmt.Sprintf(`["%s"]`, strings.Join(args, `","`)))
}

func TestAccResourceRelease_LintFailValues(t *testing.T) {
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	broken := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = "foo"
		namespace   = %q
		repository  = %q
		chart       = "test-chart"
		lint        = true
		values = [
			"replicaCount:\n  - foo: qux"
		]
	}`, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:             broken,
				PlanOnly:           true,
				ExpectError:        regexp.MustCompile("malformed chart or values"),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceRelease_LintFailChart(t *testing.T) {
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	broken := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = "foo"
		namespace   = %q
		chart       = "broken-chart"
		repository  = %q
		lint        = true
	}`, namespace, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:             broken,
				PlanOnly:           true,
				ExpectError:        regexp.MustCompile(`function "BAD_FUNCTION" not defined`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceRelease_FailedDeployFailsApply(t *testing.T) {
	name := randName("test-failed-deploy-fails-apply")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	failed := fmt.Sprintf(`
	resource "helm_release" "test" {
		name        = %q
		chart       = "failed-deploy"
		repository  = %q
	}`, name, testRepositoryURL)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:   failed,
				PlanOnly: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusFailed.String()),
				),
				ExpectError:        regexp.MustCompile(`namespaces "doesnt-exist" not found`),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccResourceRelease_dependency(t *testing.T) {
	name := fmt.Sprintf("test-dependency-%s", acctest.RandString(10))
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	// remove the subcharts so we can use `dependency_update` to grab them
	if err := removeSubcharts("umbrella-chart"); err != nil {
		t.Fatalf("Failed to remove subcharts: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config:      testAccHelmReleaseConfigDependency(testResourceName, namespace, name, false),
				ExpectError: regexp.MustCompile("found in Chart.yaml, but missing in charts/ directory"),
			},
			{
				Config: testAccHelmReleaseConfigDependency(testResourceName, namespace, name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "dependency_update", "true"),
				),
			},
			{
				PreConfig: func() {
					if err := removeSubcharts("umbrella-chart"); err != nil {
						t.Fatalf("Failed to remove subcharts: %s", err)
					}
				},
				Config: testAccHelmReleaseConfigDependencyUpdate(testResourceName, namespace, name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHelmReleaseDependencyUpdate(namespace, name, 9),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "dependency_update", "true"),
				),
			},
			{
				PreConfig: func() {
					if err := removeSubcharts("umbrella-chart"); err != nil {
						t.Fatalf("Failed to remove subcharts: %s", err)
					}
				},
				Config: testAccHelmReleaseConfigDependencyUpdateWithLint(testResourceName, namespace, name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "dependency_update", "true"),
				),
			},
		},
	})
}

func TestAccResourceRelease_chartURL(t *testing.T) {
	name := randName("chart-url")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	chartURL := fmt.Sprintf("%s/%s", testRepositoryURL, "test-chart-1.2.3.tgz")
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_chartURL(testResourceName, namespace, name, chartURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				),
			},
		},
	})
}

func TestAccResourceRelease_helm_repo_add(t *testing.T) {
	name := randName("helm-repo-add")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	// add the repository with `helm repo add`
	cmd := exec.Command("helm", "--kubeconfig", os.Getenv("KUBE_CONFIG_PATH"), "repo", "add", "hashicorp-test", testRepositoryURL)
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_helm_repo_add(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
				),
			},
		},
	})
}

func TestAccResourceRelease_delete_regression(t *testing.T) {
	name := randName("outside-delete")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
			{
				PreConfig: func() {
					// delete the release outside of terraform
					cmd := exec.Command("helm", "--kubeconfig", os.Getenv("KUBE_CONFIG_PATH"), "delete", "--namespace", namespace, name)
					out, err := cmd.CombinedOutput()
					t.Log(string(out))
					if err != nil {
						t.Fatal(err)
					}
				},
				Config:             testAccHelmReleaseConfigBasic(testResourceName, namespace, name, "1.2.3"),
				ResourceName:       testResourceName,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func getReleaseJSONManifest(namespace, name string) (string, error) {
	cmd := exec.Command("helm", "--kubeconfig", os.Getenv("KUBE_CONFIG_PATH"), "get", "manifest", "--namespace", namespace, name)
	manifest, err := cmd.Output()
	if err != nil {
		return "", err
	}

	jsonManifest, err := convertYAMLManifestToJSON(string(manifest))
	if err != nil {
		return "", err
	}
	return jsonManifest, nil
}

func TestAccResourceRelease_manifest(t *testing.T) {
	name := randName("diff")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfigManifestExperimentEnabled(testResourceName, namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					func(state *terraform.State) error {
						// FIXME this is bordering on testing the implementation
						t.Logf("getting JSON manifest for release %q", name)
						m, err := getReleaseJSONManifest(namespace, name)
						if err != nil {
							t.Fatal(err.Error())
						}
						return resource.TestCheckResourceAttr("helm_release.test", "manifest", m)(state)
					},
				),
			},
		},
	})
}

func TestAccResourceRelease_manifestUnknownValues(t *testing.T) {
	name := "example"
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			// NOTE this is a regression test to apply a configuration which supplies
			// unknown values to the release at plan time, we simply expected to test here
			// that applying the config doesn't produce an inconsistent final plan error
			{
				Config: testAccHelmReleaseConfigManifestUnknownValues(testResourceName, namespace, name, "1.2.3"),
			},
		},
	})
}

func TestAccResourceRelease_set_list_chart(t *testing.T) {
	name := randName("helm-setlist-chart")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseSetListValues(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.0.value.0", ""),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.0", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.1", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.2", "3"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.3", ""),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.#", "4"),
				),
			},
		},
	})
}

func TestAccResourceRelease_update_set_list_chart(t *testing.T) {
	name := randName("helm-setlist-chart")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseSetListValues(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.0.value.0", ""),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.0", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.1", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.2", "3"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.3", ""),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.1.value.#", "4"),
				),
			},
			{
				Config: testAccHelmReleaseUpdateSetListValues(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.chart", "test-chart"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.0.value.0", "2"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.0.value.1", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "set_list.0.value.#", "2"),
				),
			},
		},
	})
}

func setupOCIRegistry(t *testing.T, usepassword bool) (string, func()) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("Starting the OCI registry requires docker to be installed in the PATH")
	}

	helmPath, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("Starting the OCI registry requires helm to be installed in the PATH")
	}

	regitryContainerName := randName("registry")

	// start OCI registry
	// TODO run this in-process instead of starting a container
	// see here: https://pkg.go.dev/github.com/distribution/distribution/registry
	t.Log("Starting OCI registry")
	wd, _ := os.Getwd()
	runflags := []string{
		"run",
		"--detach",
		"--publish", "5000",
		"--name", regitryContainerName,
	}
	if usepassword {
		t.Log(wd)
		runflags = append(runflags, []string{
			"--volume", path.Join(wd, "testdata/oci_registry/auth.htpasswd") + ":/etc/docker/registry/auth.htpasswd",
			"--env", `REGISTRY_AUTH={htpasswd: {realm: localhost, path: /etc/docker/registry/auth.htpasswd}}`,
		}...)
	}
	runflags = append(runflags, "registry")
	cmd := exec.Command(dockerPath, runflags...)
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Errorf("Failed to start OCI registry: %v", err)
		return "", nil
	}
	// wait a few seconds for the server to start
	t.Log("Waiting for registry to start...")
	time.Sleep(5 * time.Second)

	// grab the randomly chosen port
	cmd = exec.Command(dockerPath, "port", regitryContainerName)
	out, err = cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Errorf("Failed to get port for OCI registry: %v", err)
		return "", nil
	}

	portOutput := strings.Split(string(out), "\n")[0]
	ociRegistryPort := strings.TrimSpace(strings.Split(strings.Split(portOutput, " -> ")[1], ":")[1])
	ociRegistryURL := fmt.Sprintf("oci://localhost:%s/helm-charts", ociRegistryPort)

	t.Log("OCI registry started at", ociRegistryURL)

	// package chart
	t.Log("packaging test-chart")
	cmd = exec.Command(helmPath, "package", "testdata/charts/test-chart")
	out, err = cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Errorf("Failed to package chart: %v", err)
		return "", nil
	}

	if usepassword {
		// log into OCI registry
		t.Log("logging in to test-chart to OCI registry")
		cmd = exec.Command(helmPath, "registry", "login",
			fmt.Sprintf("localhost:%s", ociRegistryPort),
			"--username", "hashicorp",
			"--password", "terraform")
		out, err = cmd.CombinedOutput()
		t.Log(string(out))
		if err != nil {
			t.Errorf("Failed to login to OCI registry: %v", err)
			return "", nil
		}
	}

	// push chart to OCI registry
	t.Log("pushing test-chart to OCI registry")
	cmd = exec.Command(helmPath, "push",
		"test-chart-1.2.3.tgz",
		ociRegistryURL)
	out, err = cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Errorf("Failed to push chart: %v", err)
		return "", nil
	}

	return ociRegistryURL, func() {
		t.Log("stopping OCI registry")
		cmd := exec.Command(dockerPath, "rm",
			"--force", regitryContainerName)
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		if err != nil {
			t.Errorf("Failed to stop OCI registry: %v", err)
		}
	}
}

func TestAccResourceRelease_OCI_repository(t *testing.T) {
	name := randName("oci")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	ociRegistryURL, shutdown := setupOCIRegistry(t, false)
	defer shutdown()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_OCI(testResourceName, namespace, name, ociRegistryURL, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
			{
				Config: testAccHelmReleaseConfig_OCI_updated(testResourceName, namespace, name, ociRegistryURL, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "set.0.name", "replicaCount"),
					resource.TestCheckResourceAttr("helm_release.test", "set.0.value", "2"),
				),
			},
			{
				Config: testAccHelmReleaseConfig_OCI_chartName(testResourceName, namespace, name, fmt.Sprintf("%s/%s", ociRegistryURL, "test-chart"), "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "chart", fmt.Sprintf("%s/%s", ociRegistryURL, "test-chart")),
				),
			},
		},
	})
}

func TestAccResourceRelease_OCI_registry_login(t *testing.T) {
	name := randName("oci")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	ociRegistryURL, shutdown := setupOCIRegistry(t, false)
	defer shutdown()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_OCI_login_provider(os.Getenv("KUBE_CONFIG_PATH"), testResourceName, namespace, name, ociRegistryURL, "1.2.3", "hashicorp", "terraform"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func testAccHelmReleaseConfig_OCI_login_provider(kubeconfig, resource, ns, name, repo, version, username, password string) string {
	return fmt.Sprintf(`
		provider "helm" {
			kubernetes {
				config_path = %q
			}
			registry {
		  		url      = %q
		  		username = %q
		  		password = %q
			}
	  	}
		resource "helm_release" "%s" {
 			name        = "%s"
			namespace   = %q
			version     = %q
			repository  = %[2]q
			chart       = "test-chart"
		}`, kubeconfig, repo, username, password, resource, name, ns, version)
}

func TestAccResourceRelease_OCI_login(t *testing.T) {
	name := randName("oci")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	ociRegistryURL, shutdown := setupOCIRegistry(t, true)
	defer shutdown()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_OCI_login_multiple(testResourceName, namespace, name, ociRegistryURL, "1.2.3", "hashicorp", "terraform"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test1", "metadata.0.name", name+"1"),
					resource.TestCheckResourceAttr("helm_release.test1", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test1", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test1", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test2", "metadata.0.name", name+"2"),
					resource.TestCheckResourceAttr("helm_release.test2", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test2", "metadata.0.version", "1.2.3"),
					resource.TestCheckResourceAttr("helm_release.test2", "status", release.StatusDeployed.String()),
				),
			},
		},
	})
}

func TestAccResourceRelease_recomputeMetadata(t *testing.T) {
	name := randName("basic")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"helm": func() (*schema.Provider, error) {
				return Provider(), nil
			},
		},
		ExternalProviders: map[string]resource.ExternalProvider{
			"local": {
				Source: "hashicorp/local",
			},
		},
		CheckDestroy: testAccCheckHelmReleaseDestroy(namespace),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseRecomputeMetadata(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "2.0.0"),
					resource.TestCheckResourceAttr("helm_release.test", "set.%", "0"),
				),
			},
			{
				Config: testAccHelmReleaseRecomputeMetadataSet(testResourceName, namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.0.version", "2.0.0"),
					resource.TestCheckResourceAttr("helm_release.test", "status", release.StatusDeployed.String()),
					resource.TestCheckResourceAttr("helm_release.test", "set.0.name", "test"),
					resource.TestCheckResourceAttr("helm_release.test", "set.0.value", "test"),
				),
			},
			{
				Config:   testAccHelmReleaseRecomputeMetadataSet(testResourceName, namespace, name),
				PlanOnly: true,
			},
		},
	})
}

func testAccHelmReleaseConfig_OCI(resource, ns, name, repo, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
			version     = %q
			chart       = "test-chart"
		}
	`, resource, name, ns, repo, version)
}

func testAccHelmReleaseConfig_OCI_login_multiple(resource, ns, name, repo, version, username, password string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s1" {
 			name        = "%s1"
			namespace   = %q
			repository  = %q
			version     = %q
			chart       = "test-chart"

			repository_username = %q
			repository_password = %q
		}
		resource "helm_release" "%[1]s2" {
			name       = "%[2]s2"
		   namespace   = %[3]q
		   repository  = %[4]q
		   version     = %[5]q
		   chart       = "test-chart"

		   repository_username = %[6]q
		   repository_password = %[7]q
	   }
	`, resource, name, ns, repo, version, username, password)
}

func testAccHelmReleaseConfig_OCI_chartName(resource, ns, name, chartName, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			version     = %q
			chart       = %q
		}
	`, resource, name, ns, version, chartName)
}

func testAccHelmReleaseConfig_OCI_updated(resource, ns, name, repo, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
			version     = %q
			chart       = "test-chart"

			set { 
				name = "replicaCount"
				value = 2
			}
		}
	`, resource, name, ns, repo, version)
}

func testAccHelmReleaseConfigManifestExperimentEnabled(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		provider helm {
			experiments {
				manifest = true
			}
		}
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
			version     = %q
			chart       = "test-chart"
		}
	`, resource, name, ns, testRepositoryURL, version)
}

func testAccHelmReleaseConfigManifestUnknownValues(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		provider helm {
			experiments {
				manifest = true
			}
		}
		resource "random_string" "random_label" {
			length           = 16
			special          = false
		}
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			repository  = %q
			version     = %q
			chart       = "test-chart"
			set {
				name  = "podAnnotations.random"
				value = random_string.random_label.result
			}
			set_sensitive {
				name  = "podAnnotations.sensitive"
				value = random_string.random_label.result
			}
			values = [<<EOT
podAnnotations:
  test: ${random_string.random_label.result}
			EOT
			]
		}
	`, resource, name, ns, testRepositoryURL, version)
}

func testAccHelmReleaseConfigDependencyUpdateWithLint(resource, ns, name string, dependencyUpdate bool) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
  			chart       = "./testdata/charts/umbrella-chart"

			dependency_update = %t
			lint = true

			set {
				name = "fake"
				value = "fake"
			}
		}
	`, resource, name, ns, dependencyUpdate)
}

func testAccHelmReleaseConfig_helm_repo_add(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			chart       = "hashicorp-test/test-chart"
			version     = "1.2.3"
		}
	`, resource, name, ns)
}

func testAccHelmReleaseConfig_chartURL(resource, ns, name, url string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
			chart       = %q
			version     = "1.2.3"
		}
	`, resource, name, ns, url)
}

func testAccHelmReleaseConfigDependency(resource, ns, name string, dependencyUpdate bool) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
  			chart       = "./testdata/charts/umbrella-chart"

			dependency_update = %t
		}
	`, resource, name, ns, dependencyUpdate)
}

func testAccHelmReleaseConfigDependencyUpdate(resource, ns, name string, dependencyUpdate bool) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
 			name        = %q
			namespace   = %q
  			chart       = "./testdata/charts/umbrella-chart"

			dependency_update = %t

			set {
				name = "fake"
				value = "fake"
			}
		}
	`, resource, name, ns, dependencyUpdate)
}

func removeSubcharts(chartName string) error {
	chartsPath := fmt.Sprintf(`testdata/charts/%s/charts`, chartName)
	if _, err := os.Stat(chartsPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove charts directory %s", chartsPath)
	}
	return os.RemoveAll(chartsPath)
}

func TestResourceExampleInstanceStateUpgradeV0(t *testing.T) {
	expected := map[string]any{
		"wait_for_jobs":    false,
		"pass_credentials": false,
	}
	states := []map[string]any{
		{
			"wait_for_jobs":    nil,
			"pass_credentials": nil,
		},
		{},
	}

	for _, state := range states {
		actual, err := resourceReleaseStateUpgradeV0(context.Background(), state, nil)
		if err != nil {
			t.Fatalf("error migrating state: %s", err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", expected, actual)
		}
	}
}

func testAccHelmReleaseSetListValues(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
	 		name        = %q
			namespace   = %q
	  		chart       = "./testdata/charts/test-chart-v2"

			set_list {
				name = "nil_check"
				value = [""]
			}

			set_list {
				name = "set_list_test"
				value = [1, 2, 3, ""]
			}
		}
`, resource, name, ns)
}

func testAccHelmReleaseUpdateSetListValues(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
			name        = %q
			namespace   = %q
	  		chart       = "./testdata/charts/test-chart-v2"

			set_list {
				name = "set_list_test"
				value = [2, 1]
			}
		}
`, resource, name, ns)
}

func testAccHelmReleaseRecomputeMetadata(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
			name        = %q
			namespace   = %q
			chart       = "./testdata/charts/test-chart-v2"
		}

		resource "local_file" "example" {
			content  = yamlencode(helm_release.test.metadata)
			filename = "${path.module}/foo.bar"
		}
`, resource, name, ns)
}

func testAccHelmReleaseRecomputeMetadataSet(resource, ns, name string) string {
	return fmt.Sprintf(`
		resource "helm_release" "%s" {
			name        = %q
			namespace   = %q
  			chart       = "./testdata/charts/test-chart-v2"

			set {
				name  = "test"
				value = "test"
			}
		}

		resource "local_file" "example" {
			content  = yamlencode(helm_release.test.metadata)
			filename = "${path.module}/foo.bar"
		}
`, resource, name, ns)
}
