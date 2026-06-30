// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0
package helm

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (r *HelmRelease) buildUpgradeStateMap(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		1: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				oldType := tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"metadata": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":           tftypes.String,
									"namespace":      tftypes.String,
									"revision":       tftypes.Number,
									"version":        tftypes.String,
									"chart":          tftypes.String,
									"app_version":    tftypes.String,
									"values":         tftypes.String,
									"first_deployed": tftypes.Number,
									"last_deployed":  tftypes.Number,
									"notes":          tftypes.String,
								},
							},
						},
						"postrender": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"binary_path": tftypes.String,
									"args":        tftypes.List{ElementType: tftypes.String},
								},
							},
						},

						"set": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":  tftypes.String,
									"value": tftypes.String,
									"type":  tftypes.String,
								},
							},
						},
						"set_list": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name": tftypes.String,
									"value": tftypes.List{
										ElementType: tftypes.String,
									},
								},
							},
						},
						"set_sensitive": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":  tftypes.String,
									"value": tftypes.String,
									"type":  tftypes.String,
								},
							},
						},
						"atomic":                     tftypes.Bool,
						"chart":                      tftypes.String,
						"cleanup_on_fail":            tftypes.Bool,
						"create_namespace":           tftypes.Bool,
						"dependency_update":          tftypes.Bool,
						"description":                tftypes.String,
						"devel":                      tftypes.Bool,
						"disable_crd_hooks":          tftypes.Bool,
						"disable_openapi_validation": tftypes.Bool,
						"disable_webhooks":           tftypes.Bool,
						"force_update":               tftypes.Bool,
						"id":                         tftypes.String,
						"keyring":                    tftypes.String,
						"lint":                       tftypes.Bool,
						"manifest":                   tftypes.String,
						"max_history":                tftypes.Number,
						"name":                       tftypes.String,
						"namespace":                  tftypes.String,
						"pass_credentials":           tftypes.Bool,
						"recreate_pods":              tftypes.Bool,
						"render_subchart_notes":      tftypes.Bool,
						"replace":                    tftypes.Bool,
						"repository":                 tftypes.String,
						"repository_ca_file":         tftypes.String,
						"repository_cert_file":       tftypes.String,
						"repository_key_file":        tftypes.String,
						"repository_password":        tftypes.String,
						"repository_username":        tftypes.String,
						"reset_values":               tftypes.Bool,
						"reuse_values":               tftypes.Bool,
						"skip_crds":                  tftypes.Bool,
						"status":                     tftypes.String,
						"timeout":                    tftypes.Number,
						"upgrade_install":            tftypes.Bool,
						"values":                     tftypes.List{ElementType: tftypes.String},
						"verify":                     tftypes.Bool,
						"version":                    tftypes.String,
						"wait":                       tftypes.Bool,
						"wait_for_jobs":              tftypes.Bool,
					},
				}

				// Unmarshalling the old raw state as old type
				oldRawValue, err := req.RawState.Unmarshal(oldType)
				if err != nil {
					resp.Diagnostics.AddError("Failed to unmarshal prior state", err.Error())
					return
				}

				var oldState map[string]tftypes.Value
				if err := oldRawValue.As(&oldState); err != nil {
					resp.Diagnostics.AddError("Failed to convert old state", err.Error())
					return
				}

				// Converting metadata into object
				var metadataList []tftypes.Value
				if err := oldState["metadata"].As(&metadataList); err != nil || len(metadataList) == 0 {
					resp.Diagnostics.AddError("Invalid old metadata format", "Expected a non-empty list")
					return
				}

				var metadata map[string]tftypes.Value
				if err := metadataList[0].As(&metadata); err != nil {
					resp.Diagnostics.AddError("Failed to read metadata[0]", err.Error())
					return
				}
				var postrenderList []tftypes.Value
				var prObj map[string]tftypes.Value

				if prVal, ok := oldState["postrender"]; ok && !prVal.IsNull() {
					if err := prVal.As(&postrenderList); err == nil && len(postrenderList) > 0 {
						if err := postrenderList[0].As(&prObj); err != nil {
							resp.Diagnostics.AddError("Failed to read postrender[0]", err.Error())
							return
						}
					}
				}

				if prObj == nil {
					prObj = map[string]tftypes.Value{
						"binary_path": tftypes.NewValue(tftypes.String, ""),
						"args":        tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
					}
				}

				// Creating new type in FW
				newType := tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"metadata": tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"name":           tftypes.String,
								"namespace":      tftypes.String,
								"revision":       tftypes.Number,
								"version":        tftypes.String,
								"chart":          tftypes.String,
								"app_version":    tftypes.String,
								"values":         tftypes.String,
								"first_deployed": tftypes.Number,
								"last_deployed":  tftypes.Number,
								"notes":          tftypes.String,
							},
						},
						"postrender": tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"binary_path": tftypes.String,
								"args":        tftypes.List{ElementType: tftypes.String},
							},
						},
						"set": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":  tftypes.String,
									"value": tftypes.String,
									"type":  tftypes.String,
								},
							},
						},
						"set_list": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name": tftypes.String,
									"value": tftypes.List{
										ElementType: tftypes.String,
									},
								},
							},
						},
						"set_sensitive": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":  tftypes.String,
									"value": tftypes.String,
									"type":  tftypes.String,
								},
							},
						},
						"set_wo": tftypes.List{
							ElementType: tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"name":  tftypes.String,
									"value": tftypes.String,
									"type":  tftypes.String,
								},
							},
						},
						"atomic":                     tftypes.Bool,
						"chart":                      tftypes.String,
						"cleanup_on_fail":            tftypes.Bool,
						"create_namespace":           tftypes.Bool,
						"dependency_update":          tftypes.Bool,
						"description":                tftypes.String,
						"devel":                      tftypes.Bool,
						"disable_crd_hooks":          tftypes.Bool,
						"disable_openapi_validation": tftypes.Bool,
						"disable_webhooks":           tftypes.Bool,
						"force_update":               tftypes.Bool,
						"id":                         tftypes.String,
						"keyring":                    tftypes.String,
						"lint":                       tftypes.Bool,
						"manifest":                   tftypes.String,
						"max_history":                tftypes.Number,
						"name":                       tftypes.String,
						"namespace":                  tftypes.String,
						"pass_credentials":           tftypes.Bool,
						"recreate_pods":              tftypes.Bool,
						"render_subchart_notes":      tftypes.Bool,
						"replace":                    tftypes.Bool,
						"repository":                 tftypes.String,
						"repository_ca_file":         tftypes.String,
						"repository_cert_file":       tftypes.String,
						"repository_key_file":        tftypes.String,
						"repository_password":        tftypes.String,
						"repository_username":        tftypes.String,
						"reset_values":               tftypes.Bool,
						"resources":                  tftypes.Map{ElementType: tftypes.String},
						"reuse_values":               tftypes.Bool,
						"skip_crds":                  tftypes.Bool,
						"set_wo_revision":            tftypes.Number,
						"status":                     tftypes.String,
						"timeout":                    tftypes.Number,
						"timeouts": tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"create": tftypes.String,
								"read":   tftypes.String,
								"update": tftypes.String,
								"delete": tftypes.String,
							},
						},
						"upgrade_install": tftypes.Bool,
						"take_ownership":  tftypes.Bool,
						"values":          tftypes.List{ElementType: tftypes.String},
						"verify":          tftypes.Bool,
						"version":         tftypes.String,
						"wait":            tftypes.Bool,
						"wait_for_jobs":   tftypes.Bool,
					},
				}
				newValue := tftypes.NewValue(newType, map[string]tftypes.Value{
					"metadata": tftypes.NewValue(newType.AttributeTypes["metadata"], metadata),
					"postrender": tftypes.NewValue(
						newType.AttributeTypes["postrender"],
						prObj,
					),
					"set_wo": tftypes.NewValue(
						newType.AttributeTypes["set_wo"],
						[]tftypes.Value{},
					),
					"take_ownership":             tftypes.NewValue(tftypes.Bool, false),
					"set_wo_revision":            tftypes.NewValue(tftypes.Number, float64(1)),
					"resources":                  tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
					"timeouts":                   tftypes.NewValue(newType.AttributeTypes["timeouts"], nil),
					"atomic":                     oldState["atomic"],
					"chart":                      oldState["chart"],
					"cleanup_on_fail":            oldState["cleanup_on_fail"],
					"create_namespace":           oldState["create_namespace"],
					"dependency_update":          oldState["dependency_update"],
					"description":                oldState["description"],
					"devel":                      oldState["devel"],
					"disable_crd_hooks":          oldState["disable_crd_hooks"],
					"disable_openapi_validation": oldState["disable_openapi_validation"],
					"disable_webhooks":           oldState["disable_webhooks"],
					"force_update":               oldState["force_update"],
					"id":                         oldState["id"],
					"keyring":                    oldState["keyring"],
					"lint":                       oldState["lint"],
					"manifest":                   oldState["manifest"],
					"max_history":                oldState["max_history"],
					"name":                       oldState["name"],
					"namespace":                  oldState["namespace"],
					"pass_credentials":           oldState["pass_credentials"],
					"recreate_pods":              oldState["recreate_pods"],
					"render_subchart_notes":      oldState["render_subchart_notes"],
					"replace":                    oldState["replace"],
					"repository":                 oldState["repository"],
					"repository_ca_file":         oldState["repository_ca_file"],
					"repository_cert_file":       oldState["repository_cert_file"],
					"repository_key_file":        oldState["repository_key_file"],
					"repository_password":        oldState["repository_password"],
					"repository_username":        oldState["repository_username"],
					"reset_values":               oldState["reset_values"],
					"reuse_values":               oldState["reuse_values"],
					"set":                        oldState["set"],
					"set_list":                   oldState["set_list"],
					"set_sensitive":              oldState["set_sensitive"],
					"skip_crds":                  oldState["skip_crds"],
					"status":                     oldState["status"],
					"timeout":                    oldState["timeout"],
					"upgrade_install":            oldState["upgrade_install"],
					"values":                     oldState["values"],
					"verify":                     oldState["verify"],
					"version":                    oldState["version"],
					"wait":                       oldState["wait"],
					"wait_for_jobs":              oldState["wait_for_jobs"],
				})

				dv, err := tfprotov6.NewDynamicValue(newType, newValue)
				if err != nil {
					resp.Diagnostics.AddError("Failed to construct upgraded state", err.Error())
					return
				}
				// Providing a message to the user, informing there state has been migrated to the current framework strucuture
				resp.Diagnostics.AddWarning("UpgradeState Triggered", "Successfully migrated state from SDKv2 to Plugin Framework")

				resp.DynamicValue = &dv
			},
		},
	}
}
