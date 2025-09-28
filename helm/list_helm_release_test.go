// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccListRelease_basic(t *testing.T) {
	namespace := createRandomNamespace(t)
	defer deleteNamespace(t, namespace)

	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				ConfigDirectory:          config.StaticDirectory("testdata/list/"),
				ConfigVariables: config.Variables{
					"test_repo": config.StringVariable(testRepositoryURL),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("helm_release.test[0]", tfjsonpath.New("name"), knownvalue.StringExact("test-0")),
					statecheck.ExpectKnownValue("helm_release.test[1]", tfjsonpath.New("name"), knownvalue.StringExact("test-1")),
					statecheck.ExpectKnownValue("helm_release.test[2]", tfjsonpath.New("name"), knownvalue.StringExact("test-2")),
				},
			},
			{
				Query:                    true,
				ProtoV6ProviderFactories: protoV6ProviderFactories(),
				ConfigDirectory:          config.StaticDirectory("testdata/list/"),
				ConfigVariables: config.Variables{
					"test_repo": config.StringVariable(testRepositoryURL),
				},
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectIdentity("helm_release.test", map[string]knownvalue.Check{
						"namespace":    knownvalue.StringExact("default"),
						"release_name": knownvalue.StringExact("test-0"),
					}),
					querycheck.ExpectIdentity("helm_release.test", map[string]knownvalue.Check{
						"namespace":    knownvalue.StringExact("default"),
						"release_name": knownvalue.StringExact("test-1"),
					}),
					querycheck.ExpectIdentity("helm_release.test", map[string]knownvalue.Check{
						"namespace":    knownvalue.StringExact("default"),
						"release_name": knownvalue.StringExact("test-2"),
					}),
				},
			},
		},
	})
}
