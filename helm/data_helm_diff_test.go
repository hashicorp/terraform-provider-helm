// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccDataDiff_newRelease(t *testing.T) {
	name := randName("diff-new")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataHelmDiffConfig_standalone(namespace, name, "1.2.3", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.helm_diff.test", "name", name),
					resource.TestCheckResourceAttr("data.helm_diff.test", "namespace", namespace),
					resource.TestCheckResourceAttr("data.helm_diff.test", "has_changes", "true"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "diff"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "proposed_manifest"),
					resource.TestCheckResourceAttr("data.helm_diff.test", "current_manifest", ""),
					resource.TestMatchResourceAttr("data.helm_diff.test", "diff", regexp.MustCompile("has been added")),
				),
			},
		},
	})
}

func TestAccDataDiff_existingReleaseNoChanges(t *testing.T) {
	name := randName("diff-nochange")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_basic(namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.name", name),
					resource.TestCheckResourceAttr("helm_release.test", "metadata.namespace", namespace),
					func(s *terraform.State) error {
						return testCheckHelmReleaseExists(t, namespace, name)
					},
				),
			},
			{
				Config: testAccDataHelmDiffConfig_withRelease(namespace, name, "1.2.3", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						return testCheckHelmReleaseExists(t, namespace, name)
					},
					resource.TestCheckResourceAttr("data.helm_diff.test", "name", name),
					resource.TestCheckResourceAttr("data.helm_diff.test", "namespace", namespace),
					resource.TestCheckResourceAttr("data.helm_diff.test", "has_changes", "false"),
					resource.TestCheckResourceAttr("data.helm_diff.test", "diff", ""),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "current_manifest"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "proposed_manifest"),
				),
			},
		},
	})
}

func TestAccDataDiff_existingReleaseWithChanges(t *testing.T) {
	name := randName("diff-change")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_basic(namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.name", name),
				),
			},
			{
				Config: testAccDataHelmDiffConfig_withRelease(namespace, name, "1.2.3", map[string]string{
					"service.type": "NodePort",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.helm_diff.test", "name", name),
					resource.TestCheckResourceAttr("data.helm_diff.test", "namespace", namespace),
					resource.TestCheckResourceAttr("data.helm_diff.test", "has_changes", "true"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "diff"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "current_manifest"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "proposed_manifest"),
					resource.TestMatchResourceAttr("data.helm_diff.test", "diff", regexp.MustCompile("has changed")),
				),
			},
		},
	})
}

func TestAccDataDiff_existingReleaseVersionChange(t *testing.T) {
	name := randName("diff-version")
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccHelmReleaseConfig_basic(namespace, name, "1.2.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("helm_release.test", "metadata.version", "1.2.3"),
				),
			},
			{
				Config: testAccDataHelmDiffConfig_withRelease(namespace, name, "2.0.0", nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.helm_diff.test", "has_changes", "true"),
					resource.TestCheckResourceAttrSet("data.helm_diff.test", "diff"),
				),
			},
		},
	})
}

func testAccHelmReleaseConfig_basic(namespace, name, version string) string {
	return fmt.Sprintf(`
		resource "helm_release" "test" {
			name             = %q
			namespace        = %q
			repository       = %q
			chart            = "test-chart"
			version          = %q
			disable_webhooks = true
		}
	`, name, namespace, testRepositoryURL, version)
}

func testAccDataHelmDiffConfig_withRelease(namespace, name, version string, values map[string]string) string {
	setBlocks := ""
	if len(values) > 0 {
		setBlocks = "\n\t\t\tset = ["
		for k, v := range values {
			setBlocks += fmt.Sprintf("\n\t\t\t\t{\n\t\t\t\t\tname  = %q\n\t\t\t\t\tvalue = %q\n\t\t\t\t},", k, v)
		}
		setBlocks += "\n\t\t\t]"
	}

	return fmt.Sprintf(`
		resource "helm_release" "test" {
			name             = %q
			namespace        = %q
			repository       = %q
			chart            = "test-chart"
			version          = %q
			disable_webhooks = true
		}

		data "helm_diff" "test" {
			name             = %q
			namespace        = %q
			repository       = %q
			chart            = "test-chart"
			version          = %q
			disable_webhooks = true%s
		}
	`, name, namespace, testRepositoryURL, version, name, namespace, testRepositoryURL, version, setBlocks)
}

func testAccDataHelmDiffConfig_standalone(namespace, name, version string, values map[string]string) string {
	setBlocks := ""
	if len(values) > 0 {
		setBlocks = "\n\t\t\tset = ["
		for k, v := range values {
			setBlocks += fmt.Sprintf("\n\t\t\t\t{\n\t\t\t\t\tname  = %q\n\t\t\t\t\tvalue = %q\n\t\t\t\t},", k, v)
		}
		setBlocks += "\n\t\t\t]"
	}

	return fmt.Sprintf(`
		data "helm_diff" "test" {
			name             = %q
			namespace        = %q
			repository       = %q
			chart            = "test-chart"
			version          = %q
			disable_webhooks = true%s
		}
	`, name, namespace, testRepositoryURL, version, setBlocks)
}

func testCheckHelmReleaseExists(t *testing.T, namespace, name string) error {
	t.Helper()

	secrets, err := client.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("owner=helm,name=%s", name),
	})
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(secrets.Items) == 0 {
		return fmt.Errorf("release %q not found in namespace %q (no Helm secrets found)", name, namespace)
	}

	return nil
}
