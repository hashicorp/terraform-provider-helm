package helm

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"helm.sh/helm/v3/pkg/action"
)

var (
	_ list.ListResource              = &HelmReleaseList{}
	_ list.ListResourceWithConfigure = &HelmReleaseList{}
)

type HelmReleaseList struct {
	meta *Meta
}

type HelmReleaseListConfig struct {
	AllNamespaces types.Bool   `tfsdk:"all_namespaces"`
	Namespace     types.String `tfsdk:"namespace"`
	Filter        types.String `tfsdk:"filter"`
}

func NewHelmReleaseList() list.ListResource {
	return &HelmReleaseList{}
}

func (l *HelmReleaseList) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		tflog.Debug(ctx, "Meta is nil for List Resource")

		// HACK FIXME
		l.meta = m

		return
	}

	meta, ok := req.ProviderData.(*Meta)
	if !ok {
		resp.Diagnostics.AddError(
			"Provider Configuration Error",
			fmt.Sprintf("Unexpected ProviderData type: %T", req.ProviderData),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Configured meta: %+v", meta))

	l.meta = meta
}

func (l *HelmReleaseList) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

func (l *HelmReleaseList) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
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

func (l *HelmReleaseList) List(ctx context.Context, req list.ListRequest, results *list.ListResultsStream) {
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

/*

func HelmReleaseResourceSchema() *tfprotov6.Schema {
	return &tfprotov6.Schema{
		Block: &tfprotov6.SchemaBlock{
			Attributes: []*tfprotov6.SchemaAttribute{
				{
					Name:        "filter",
					Type:        tftypes.String,
					Optional:    true,
					Description: "A regular expression (Perl compatible). Any releases that match the expression will be included.",
				},
				{
					Name:        "namespace",
					Type:        tftypes.String,
					Optional:    true,
					Description: "The namespace to list helm releases.",
				},
				{
					Name:        "all_namespaces",
					Type:        tftypes.Bool,
					Optional:    true,
					Description: "List releases across all namespaces.",
				},
			},
		},
	}
}

func (p *HelmProvider) ListResource(ctx context.Context, request *tfprotov6.ListResourceRequest) (*tfprotov6.ListResourceServerStream, error) {
	results := func(push func(tfprotov6.ListResourceResult) bool) {
		if request.TypeName == "helm_release" {
			// do the thing
			namespace := "default"
			allNamespaces := false
			filter := ""

			cfg, err := p.meta.GetHelmConfiguration(ctx, namespace)
			if err != nil {
				// FIXME push diag here
				return
			}

			list := action.NewList(cfg)
			list.Filter = filter
			list.AllNamespaces = allNamespaces

			releases, err := list.Run()
			if err != nil {
				// FIXME push diag here
				return
			}

			for _, rls := range releases {
				push(&tfprotov6.ListResourceResult{
					DisplayName: fmt.Sprintf("%s/%s", rls.Namespace, rls.Name),
					Identity: &tfprotov6.ResourceIdentityData{

					},
				})
			}
		}
	}

	return &tfprotov6.ListResourceServerStream{
		Results: results,
	}, nil
}

*/
