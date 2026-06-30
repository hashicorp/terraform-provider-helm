// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package testing

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-helm/helm"
)

var providerFactory = map[string]func() (tfprotov6.ProviderServer, error){
	"helm": providerserver.NewProtocol6WithError(helm.New("version")()),
}

func TestAccDeferredActions_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_9_0),
		},
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{
			Plan:  resource.PlanOptions{AllowDeferral: true},
			Apply: resource.ApplyOptions{AllowDeferral: true},
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: providerFactory,
				ConfigDirectory: func(tscr config.TestStepConfigRequest) string {
					return "config-da-basic"
				},
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kind_cluster.demo", plancheck.ResourceActionCreate),
						plancheck.ExpectDeferredChange("helm_release.test-release", plancheck.DeferredReasonProviderConfigUnknown),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("helm_release.test-release", plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("endpoint"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("cluster_ca_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_key"), knownvalue.NotNull()),
				},
			},
			{
				ProtoV6ProviderFactories: providerFactory,
				ConfigDirectory: func(tscr config.TestStepConfigRequest) string {
					return "config-da-basic"
				},
				ExpectNonEmptyPlan: false,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("helm_release.test-release", plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("endpoint"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("cluster_ca_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_certificate"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("kind_cluster.demo", tfjsonpath.New("client_key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("helm_release.test-release", tfjsonpath.New("name"), knownvalue.StringExact("test-hello")),
					statecheck.ExpectKnownValue("helm_release.test-release", tfjsonpath.New("chart"), knownvalue.StringExact("hello")),
					statecheck.ExpectKnownValue("helm_release.test-release", tfjsonpath.New("repository"), knownvalue.StringExact("https://cloudecho.github.io/charts/")),
					statecheck.ExpectKnownValue("helm_release.test-release", tfjsonpath.New("status"), knownvalue.StringExact("deployed")),
				},
			},
		},
	})
}
