// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHelmRelease_Upgrade_MetadataStructure(t *testing.T) {
	name := randName("upgrade")
	namespace := "default"
	resourceName := "helm_release.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Apply using SDKv2 (v2.17.0)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.17.0",
					},
				},
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"
  wait       = false
}
`, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.chart", "nginx"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.version", "15.0.0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.app_version", "1.25.0"),
					resource.TestCheckResourceAttr(resourceName, "status", "deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.notes"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.first_deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.last_deployed"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.revision", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.values", "{}"),
				),
			},

			// Step 2: Reapply using Plugin Framework
			{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"
  wait       = false
}
`, name, namespace),
				// Checking if strucuture of metadata has been migrated to plugin framework strucuture
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.chart", "nginx"),
					resource.TestCheckResourceAttr(resourceName, "metadata.version", "15.0.0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.app_version", "1.25.0"),
					resource.TestCheckResourceAttr(resourceName, "status", "deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.notes"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.first_deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.last_deployed"),
					resource.TestCheckResourceAttr(resourceName, "metadata.revision", "2"),
				),
			},
		},
	})
}

func TestAccHelmRelease_Upgrade_values(t *testing.T) {
	// regression test, see: https://github.com/hashicorp/terraform-provider-helm/issues/1637

	name := randName("upgrade")
	namespace := "default"
	resourceName := "helm_release.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.17.0",
					},
				},
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"
  wait       = false
  values     = [
<<EOF
foo: bar
EOF
  ]
}
`, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "values.0", "foo: bar\n"),
				),
			},
			{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"
  wait       = false
  values     = [
<<EOF
foo: bar
EOF
	]
}
`, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "values.0", "foo: bar\n"),
				),
			},
		},
	})
}

func TestAccHelmRelease_Upgrade_PostrenderStructure(t *testing.T) {
	name := randName("upgrade-pr")
	namespace := "default"
	resourceName := "helm_release.test"

	binaryPath := "true"
	args := []string{"hello", "world"}

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Apply using SDKv2 (v2.17.0)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.17.0",
					},
				},
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"

  postrender {
    binary_path = "%s"
    args        = ["%s", "%s"]
  }

  wait = false
}
`, name, namespace, binaryPath, args[0], args[1]),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "postrender.0.binary_path", binaryPath),
					resource.TestCheckResourceAttr(resourceName, "postrender.0.args.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "postrender.0.args.0", args[0]),
					resource.TestCheckResourceAttr(resourceName, "postrender.0.args.1", args[1]),
				),
			},

			// Step 2: Reapply using Plugin Framework
			{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				Config: fmt.Sprintf(`
resource "helm_release" "test" {
  name       = "%s"
  namespace  = "%s"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.0.0"

  postrender = {
    binary_path = "%s"
    args        = ["%s", "%s"]
  }

  wait = false
}
`, name, namespace, binaryPath, args[0], args[1]),
				// Checking if strucuture of postrender has been migrated to plugin framework strucuture
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "postrender.binary_path", binaryPath),
					resource.TestCheckResourceAttr(resourceName, "postrender.args.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "postrender.args.0", args[0]),
					resource.TestCheckResourceAttr(resourceName, "postrender.args.1", args[1]),
				),
			},
		},
	})
}
