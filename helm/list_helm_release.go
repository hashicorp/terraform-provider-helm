// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"helm.sh/helm/v3/pkg/action"
)

var _ list.ListResource = &HelmRelease{}

type HelmReleaseListConfig struct {
	AllNamespaces types.Bool   `tfsdk:"all_namespaces"`
	Namespace     types.String `tfsdk:"namespace"`
	Filter        types.String `tfsdk:"filter"`
}

func NewHelmReleaseList() list.ListResource {
	return &HelmRelease{}
}

func (l *HelmRelease) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are available in the resource",
		Attributes: map[string]schema.Attribute{
			"all_namespaces": schema.BoolAttribute{
				Optional:    true,
				Description: "List releases across all namespaces.",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "The namespace to list helm releases.",
			},
			"filter": schema.StringAttribute{
				Optional:    true,
				Description: "A regular expression (Perl compatible). Any releases that match the expression will be included.",
			},
		},
	}
}

func (l *HelmRelease) List(ctx context.Context, req list.ListRequest, results *list.ListResultsStream) {
	results.Results = func(yield func(list.ListResult) bool) {
		listConfig := HelmReleaseListConfig{}
		req.Config.Get(ctx, &listConfig)

		namespace := listConfig.Namespace.ValueString()
		cfg, err := l.meta.GetHelmConfiguration(ctx, namespace)
		if err != nil {
			diags := diag.Diagnostics{}
			diags.AddError("Error creating Helm client", err.Error())
			yield(list.ListResult{
				Diagnostics: diags,
			})
			return
		}

		listAction := action.NewList(cfg)
		listAction.Filter = listConfig.Filter.ValueString()
		listAction.AllNamespaces = listConfig.AllNamespaces.ValueBool()

		releases, err := listAction.Run()
		if err != nil {
			diags := diag.Diagnostics{}
			diags.AddError("Error running list action", err.Error())
			yield(list.ListResult{
				Diagnostics: diags,
			})
			return
		}

		for _, rls := range releases {
			idData := HelmReleaseIdentityModel{
				Namespace:   types.StringValue(rls.Namespace),
				ReleaseName: types.StringValue(rls.Name),
			}
			id := tfsdk.ResourceIdentity{
				Schema: helmReleaseIdentitySchema(),
			}
			id.Set(ctx, idData)

			resourceData := HelmReleaseModel{}
			diags := setReleaseAttributes(ctx, &resourceData, &id, rls, l.meta)
			if diags.HasError() {
				yield(list.ListResult{
					Diagnostics: diags,
				})
				return
			}
			resourceData.Set = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":  types.StringType,
					"type":  types.StringType,
					"value": types.StringType,
				},
			})
			resourceData.SetWO = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":  types.StringType,
					"type":  types.StringType,
					"value": types.StringType,
				},
			})
			resourceData.SetSensitive = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":  types.StringType,
					"type":  types.StringType,
					"value": types.StringType,
				},
			})
			resourceData.SetList = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": types.StringType,
					"value": types.ListType{
						ElemType: types.StringType,
					},
				},
			})
			resourceData.Values = types.ListNull(types.StringType)

			r := tfsdk.Resource{
				Schema: helmReleaseSchema(),
			}
			diags = r.Set(ctx, &resourceData)

			yield(list.ListResult{
				DisplayName: fmt.Sprintf("%s/%s", rls.Namespace, rls.Name),
				Identity:    &id,
				Resource:    &r,
				Diagnostics: diags,
			})
		}
	}
}
