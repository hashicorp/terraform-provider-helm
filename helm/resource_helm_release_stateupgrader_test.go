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

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttrSet(resourceName, "metadata.revision"),
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

	resource.Test(t, resource.TestCase{
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

	resource.Test(t, resource.TestCase{
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

// Tests for upgrading from schema version 0 (v2.4.0 and earlier)
// These versions did not have first_deployed, last_deployed, notes in metadata
// and did not have pass_credentials attribute

func TestAccHelmRelease_UpgradeV0_MetadataStructure(t *testing.T) {
	name := randName("upgrade-v0")
	namespace := "default"
	resourceName := "helm_release.test"

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Apply using old SDKv2 (v2.4.0) - schema version 0
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.4.0",
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
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.app_version"),
					resource.TestCheckResourceAttr(resourceName, "status", "deployed"),
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
				// Checking if structure of metadata has been migrated from v0 to plugin framework structure
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.namespace", namespace),
					resource.TestCheckResourceAttr(resourceName, "metadata.chart", "nginx"),
					resource.TestCheckResourceAttr(resourceName, "metadata.version", "15.0.0"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.app_version"),
					resource.TestCheckResourceAttr(resourceName, "status", "deployed"),
					// first_deployed and last_deployed should be populated on next apply
					resource.TestCheckResourceAttrSet(resourceName, "metadata.first_deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.last_deployed"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.revision"),
				),
			},
		},
	})
}

func TestAccHelmRelease_UpgradeV0_values(t *testing.T) {
	// Test that values are preserved when upgrading from schema v0
	name := randName("upgrade-v0")
	namespace := "default"
	resourceName := "helm_release.test"

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.4.0",
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

func TestAccHelmRelease_UpgradeV0_PostrenderStructure(t *testing.T) {
	// Note: v2.4.0 does not support 'args' in postrender block, so we only test binary_path
	name := randName("upgrade-v0-pr")
	namespace := "default"
	resourceName := "helm_release.test"

	binaryPath := "true"

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Apply using old SDKv2 (v2.4.0) - schema version 0
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.4.0",
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
  }

  wait = false
}
`, name, namespace, binaryPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "postrender.0.binary_path", binaryPath),
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
  }

  wait = false
}
`, name, namespace, binaryPath),
				// Checking if structure of postrender has been migrated from v0 to plugin framework structure
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "postrender.binary_path", binaryPath),
				),
			},
		},
	})
}

func TestAccHelmRelease_UpgradeV0_SetBlocks(t *testing.T) {
	// Test that set and set_sensitive blocks are preserved when upgrading from schema v0
	name := randName("upgrade-v0-set")
	namespace := "default"
	resourceName := "helm_release.test"

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: Apply using old SDKv2 (v2.4.0) - schema version 0
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"helm": {
						Source:            "hashicorp/helm",
						VersionConstraint: "=2.4.0",
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

  set {
    name  = "replicaCount"
    value = "2"
  }

  set_sensitive {
    name  = "service.type"
    value = "ClusterIP"
  }
}
`, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "set.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "set.0.name", "replicaCount"),
					resource.TestCheckResourceAttr(resourceName, "set.0.value", "2"),
					resource.TestCheckResourceAttr(resourceName, "set_sensitive.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "set_sensitive.0.name", "service.type"),
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

  set = [{
    name  = "replicaCount"
    value = "2"
  }]

  set_sensitive = [{
    name  = "service.type"
    value = "ClusterIP"
  }]
}
`, name, namespace),
				// Checking that set and set_sensitive are preserved after v0 migration
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "set.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "set.0.name", "replicaCount"),
					resource.TestCheckResourceAttr(resourceName, "set.0.value", "2"),
					resource.TestCheckResourceAttr(resourceName, "set_sensitive.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "set_sensitive.0.name", "service.type"),
				),
			},
		},
	})
}
