// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataTemplate_basic(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateConfigBasic(testResourceName, namespace, name, "1.2.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "5"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/deployment.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/service.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/serviceaccount.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/configmaps.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/tests/test-connection.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
			),
		}},
	})
}

func TestAccDataTemplate_crds(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateCRDs(testResourceName, namespace, name, "1.2.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "8"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/deployment.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/service.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/serviceaccount.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/configmaps.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/tests/test-connection.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
				resource.TestCheckResourceAttr(datasourceAddress, "crds.0", `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: apples.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    plural: apples
    singular: apple
    kind: Apple
    shortNames:
    - ap
`),
				resource.TestCheckResourceAttr(datasourceAddress, "crds.1", `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: oranges.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    plural: oranges
    singular: orange
    kind: Orange
    shortNames:
    - or
`),
			),
		}},
	})
}

func TestAccDataTemplate_templates(t *testing.T) {
	name := randName("basic")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)
	expectedTemplate := fmt.Sprintf(`---
# Source: test-chart/templates/configmaps.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: %[1]s-test-chart-one
  labels:
    helm.sh/chart: test-chart-1.2.3
    app.kubernetes.io/name: test-chart
    app.kubernetes.io/instance: %[1]s
    app.kubernetes.io/version: "1.19.5"
    app.kubernetes.io/managed-by: Helm
data:
  test: one
---
# Source: test-chart/templates/configmaps.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: %[1]s-test-chart-two
  labels:
    helm.sh/chart: test-chart-1.2.3
    app.kubernetes.io/name: test-chart
    app.kubernetes.io/instance: %[1]s
    app.kubernetes.io/version: "1.19.5"
    app.kubernetes.io/managed-by: Helm
data:
  test: two
`, name)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateConfigTemplates(testResourceName, namespace, name, "1.2.3"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "1"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "notes"),
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.templates/configmaps.yaml", expectedTemplate),
			),
		}},
	})
}

func TestAccDataTemplate_kubeVersion(t *testing.T) {
	name := randName("kube-version")
	namespace := randName(testNamespacePrefix)

	datasourceAddress := fmt.Sprintf("data.helm_template.%s", testResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config:      testAccDataHelmTemplateKubeVersionNoVersionSet(testResourceName, namespace, name, "1.2.3"),
			ExpectError: regexp.MustCompile("chart requires kubeVersion.*"),
		}},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config:      testAccDataHelmTemplateKubeVersion(testResourceName, namespace, name, "1.2.3", "1.18.0"),
			ExpectError: regexp.MustCompile("chart requires kubeVersion.*"),
		}},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config:      testAccDataHelmTemplateKubeVersion(testResourceName, namespace, name, "1.2.3", "abcdef"),
			ExpectError: regexp.MustCompile(`couldn't parse string "abcdef" into kube-version`),
		}},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testAccDataHelmTemplateKubeVersion(testResourceName, namespace, name, "1.2.3", "1.22.0"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(datasourceAddress, "manifests.%", "1"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifests.templates/deployment.yaml"),
				resource.TestCheckResourceAttrSet(datasourceAddress, "manifest"),
			),
		}},
	})
}

func testAccDataHelmTemplateConfigBasic(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
			show_only	= [""]
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

func testAccDataHelmTemplateConfigTemplates(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
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

			show_only = [
				"templates/configmaps.yaml",
				""
			]
		}
	`, resource, name, ns, testRepositoryURL, version)
}

func testAccDataHelmTemplateKubeVersion(resource, ns, name, version, kubeVersion string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name         = %q
			namespace    = %q
			description  = "Test"
			repository   = %q
  			chart        = "kube-version"
			version      = %q
			kube_version = %q
		}
	`, resource, name, ns, testRepositoryURL, version, kubeVersion)
}

func testAccDataHelmTemplateKubeVersionNoVersionSet(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name         = %q
			namespace    = %q
			description  = "Test"
			repository   = %q
  			chart        = "kube-version"
			version      = %q
		}
	`, resource, name, ns, testRepositoryURL, version)
}

func testAccDataHelmTemplateCRDs(resource, ns, name, version string) string {
	return fmt.Sprintf(`
		data "helm_template" "%s" {
 			name         = %q
			namespace    = %q
			description  = "Test"
			repository   = %q
  			chart        = "crds-chart"
			include_crds = true
			version      = %q
		}
	`, resource, name, ns, testRepositoryURL, version)
}
