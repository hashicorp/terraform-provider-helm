package helm

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

type HelmReleaseResource struct {
	meta *MetaRelease
}

func NewHelmReleaseResource(meta *MetaRelease) resource.Resource {
	return &HelmReleaseResource{
		meta: meta,
	}
}

type helmReleaseModel struct {
	Name                       types.String                 `tfsdk:"name"`
	Repository                 types.String                 `tfsdk:"repository"`
	Repository_Key_File        types.String                 `tfsdk:"repository_key_file"`
	Repository_Cert_File       types.String                 `tfsdk:"repository_cert_file"`
	Repository_Ca_File         types.String                 `tfsdk:"repository_ca_file"`
	Repository_Username        types.String                 `tfsdk:"repository_username"`
	Repository_Password        types.String                 `tfsdk:"repository_password"`
	Pass_Credentials           types.Bool                   `tfsdk:"pass_credentials"`
	Chart                      types.String                 `tfsdk:"chart"`
	Version                    types.String                 `tfsdk:"version"`
	Devel                      types.Bool                   `tfsdk:"devel"`
	Values                     types.List                   `tfsdk:"devel"`
	Set                        []setResourceModel           `tfsdk:"set"`
	Set_list                   []set_listResourceModel      `tfsdk:"set_list"`
	Set_Sensitive              []set_sensitiveResourceModel `tfsdk:"set_sensitive"`
	Namespace                  types.String                 `tfsdk:"namespace"`
	Verify                     types.Bool                   `tfsdk:"verify"`
	Keyring                    types.String                 `tfsdk:"keyring"`
	Timeout                    types.Int64                  `tfsdk:"timeout"`
	Disable_Webhooks           types.Bool                   `tfsdk:"disable_webhooks"`
	Disable_Crd_Hooks          types.Bool                   `tfsdk:"disable_crd_hooks"`
	Reset_Values               types.Bool                   `tfsdk:"reset_values"`
	Force_Update               types.Bool                   `tfsdk:"force_update"`
	Recreate_Pods              types.Bool                   `tfsdk:"recreate_pods"`
	Cleanup_On_Fail            types.Bool                   `tfsdk:"cleanup_on_fail"`
	Max_History                types.Int64                  `tfsdk:"max_history"`
	Atomic                     types.Bool                   `tfsdk:"atomic"`
	Skip_Crds                  types.Bool                   `tfsdk:"skip_crds"`
	Render_Subchart_Notes      types.Bool                   `tfsdk:"render_subchart_notes"`
	Disable_Openapi_Validation types.Bool                   `tfsdk:"disable_openapi_validation"`
	Wait                       types.Bool                   `tfsdk:"wait"`
	Wait_For_Jobs              types.Bool                   `tfsdk:"wait_for_jobs"`
	Status                     types.String                 `tfsdk:"STATUS"`
	Dependency_Update          types.Bool                   `tfsdk:"dependency_update"`
	Replace                    types.Bool                   `tfsdk:"replace"`
	Description                types.String                 `tfsdk:"description"`
	Create_Namespace           types.Bool                   `tfsdk:"create_namespace"`
	Postrender                 types.List                   `tfsdk:"postrender"`
	Lint                       types.Bool                   `tfsdk:"lint"`
	Manifest                   types.String                 `tfsdk:"manifest"`
	Metadata                   types.List                   `tfsdk:"metadata"`
}

type setResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type set_listResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type set_sensitiveResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type postrender struct {
	Binary_Path types.String `tfsdk:"binary_path"`
	Args        types.List   `tfsdk:"args"`
}

type MetaRelease struct {
	data           *helmReleaseModel
	Settings       *cli.EnvSettings
	RegistryClient *registry.Client
	HelmDriver     string
	sync.Mutex
}

func (r *HelmReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	/* ... */
}

func (r *HelmReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are avaiavle in the resource",
		Attributes: map[string]schema.Attribute{

			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 53),
				},
				Description: "Release name. The length must not be longer than 53 characters",
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If is a URL the chart is installed without installing the repository",
			},
			"repository_key_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert key file",
			},
			"repository_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert file",
			},
			"respository_ca_file": schema.StringAttribute{
				Optional:    true,
				Description: "The Repositories CA file",
			},
			"repository_username": schema.StringAttribute{
				Optional:    true,
				Description: "Username for HTTP basic authentication",
			},
			"repository_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for HTTP basic authentication",
			},
			"pass_credentials": schema.BoolAttribute{
				Optional:    true,
				Description: "Pass credentials to all domains",
				Default:     booldefault.StaticBool(false),
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used,",
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed ",
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If 'version' is set, this is ignored",
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				//return d.Get("version").(string) != ""
			},
			"values": schema.ListAttribute{
				Optional:    true,
				Description: "List of values in raw yamls format to pass to helm",
				ElementType: types.StringType,
			},
			"set": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Custom values to be merged with the values",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Optional: true,
							Default:  stringdefault.StaticString(""),
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string"),
							},
						},
					},
				},
			},
			"set_list": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Custom sensitive values to be merged with the values",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"set_sensitive": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Custom sensitive values to be merged with the values",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							Required:  true,
							Sensitive: true,
						},
						"type": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string"),
							},
						},
					},
				},
			},
			"namespace": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Namespace to install the release into",
			},
			"verify": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Verify the package before installing it.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Description: "Location of public keys used for verification, Used on if 'verify is true'",
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				//return !d.Get("verify").(bool)
				//},
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(300),
				Description: "Time in seconds to wait for any indvidiual kubernetes operation",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Prevent hooks from running",
			},
			"disable_crd_hooks": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Prevent CRD hooks from, running, but run other hooks.  See helm install --no-crd-hook",
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
				Default:     booldefault.StaticBool(false),
			},
			"reset_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reset the values to the ones built into the chart",
				Default:     booldefault.StaticBool(false),
			},
			"force_update": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Force resource update through delete/recreate if needed.",
			},
			"recreate_pods": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Perform pods restart during upgrade/rollback",
			},
			"cleanup_on_fail": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow deletion of new resources created in this upgrade when upgrade fails",
			},
			"max_history": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Limit the maximum number of revisions saved per release. Use 0 for no limit",
			},
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
			},
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(true),
				Description: "If set, render subchart notes along with the parent",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"wait_for_jobs": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the release",
			},

			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Run helm dependency update before installing the chart",
			},

			"replace": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Add a custom description",
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				//return new == ""
				//},
			},

			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Create the namespace if it does not exist",
			},
			"postrender": schema.ListNestedAttribute{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				Optional:    true,
				Description: "Postrender command config",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"binary_path": schema.StringAttribute{
							Required:    true,
							Description: "The common binary path",
						},
						"args": schema.ListAttribute{
							Optional:    true,
							Description: "An argument to the post-renderer (can specify multiple)",
							ElementType: types.StringType,
						},
					},
				},
			},

			"lint": schema.BoolAttribute{
				Optional:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Run helm lint when planning",
			},

			"manifest": schema.StringAttribute{
				Description: "The rendered manifest as 	JSON.",
				Computed:    true,
			},
			"metadata": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Stats of the deployed release.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name is the name of the release",
						},
						"revision": schema.Int64Attribute{
							Computed:    true,
							Description: "Version is an int32 which represents the version of the release",
						},
						"namespace": schema.StringAttribute{
							Computed:    true,
							Description: "Namespace is the kubernetes namespace of the release",
						},
						"chart": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the chart",
						},
						"version": schema.StringAttribute{
							Computed:    true,
							Description: "A SemVer 2 conformant version string of the chart",
						},
						"app_version": schema.StringAttribute{
							Computed:    true,
							Description: "The version number of the application being deployed",
						},
						"values": schema.StringAttribute{
							Computed:    true,
							Description: "Set of extra values. added to the chart. The sensitive data is cloaked. JSON encdoed.",
						},
					},
				},
			},
		},
	}
}

func (r *HelmReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
}
func (r *HelmReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *HelmReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *HelmReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
