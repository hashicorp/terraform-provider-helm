package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/postrender"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/helm/pkg/strvals"
	"sigs.k8s.io/yaml"
)

var (
	_ resource.Resource                = &HelmRelease{}
	_ resource.ResourceWithModifyPlan  = &HelmRelease{}
	_ resource.ResourceWithImportState = &HelmRelease{}
)

type HelmRelease struct {
	meta *Meta
}

func NewHelmRelease() resource.Resource {
	return &HelmRelease{}
}

type HelmReleaseModel struct {
	Atomic                   types.Bool   `tfsdk:"atomic"`
	Chart                    types.String `tfsdk:"chart"`
	CleanupOnFail            types.Bool   `tfsdk:"cleanup_on_fail"`
	CreateNamespace          types.Bool   `tfsdk:"create_namespace"`
	DependencyUpdate         types.Bool   `tfsdk:"dependency_update"`
	Description              types.String `tfsdk:"description"`
	Devel                    types.Bool   `tfsdk:"devel"`
	DisableCrdHooks          types.Bool   `tfsdk:"disable_crd_hooks"`
	DisableOpenapiValidation types.Bool   `tfsdk:"disable_openapi_validation"`
	DisableWebhooks          types.Bool   `tfsdk:"disable_webhooks"`
	ForceUpdate              types.Bool   `tfsdk:"force_update"`
	ID                       types.String `tfsdk:"id"`
	Keyring                  types.String `tfsdk:"keyring"`
	Lint                     types.Bool   `tfsdk:"lint"`
	Manifest                 types.String `tfsdk:"manifest"`
	MaxHistory               types.Int64  `tfsdk:"max_history"`
	Metadata                 types.Object `tfsdk:"metadata"`
	Name                     types.String `tfsdk:"name"`
	Namespace                types.String `tfsdk:"namespace"`
	PassCredentials          types.Bool   `tfsdk:"pass_credentials"`
	Postrender               types.Object `tfsdk:"postrender"`
	RecreatePods             types.Bool   `tfsdk:"recreate_pods"`
	Replace                  types.Bool   `tfsdk:"replace"`
	RenderSubchartNotes      types.Bool   `tfsdk:"render_subchart_notes"`
	Repository               types.String `tfsdk:"repository"`
	RepositoryCaFile         types.String `tfsdk:"repository_ca_file"`
	RepositoryCertFile       types.String `tfsdk:"repository_cert_file"`
	RepositoryKeyFile        types.String `tfsdk:"repository_key_file"`
	RepositoryPassword       types.String `tfsdk:"repository_password"`
	RepositoryUsername       types.String `tfsdk:"repository_username"`
	ResetValues              types.Bool   `tfsdk:"reset_values"`
	ReuseValues              types.Bool   `tfsdk:"reuse_values"`
	Set                      types.Set    `tfsdk:"set"`
	SetList                  types.List   `tfsdk:"set_list"`
	SetSensitive             types.Set    `tfsdk:"set_sensitive"`
	SkipCrds                 types.Bool   `tfsdk:"skip_crds"`
	Status                   types.String `tfsdk:"status"`
	Timeout                  types.Int64  `tfsdk:"timeout"`
	Values                   types.List   `tfsdk:"values"`
	Verify                   types.Bool   `tfsdk:"verify"`
	Version                  types.String `tfsdk:"version"`
	Wait                     types.Bool   `tfsdk:"wait"`
	WaitForJobs              types.Bool   `tfsdk:"wait_for_jobs"`
}

var defaultAttributes = map[string]interface{}{
	"atomic":                     false,
	"cleanup_on_fail":            false,
	"create_namespace":           false,
	"dependency_update":          false,
	"disable_crd_hooks":          false,
	"disable_openapi_validation": false,
	"disable_webhooks":           false,
	"force_update":               false,
	"lint":                       false,
	"max_history":                int64(0),
	"pass_credentials":           false,
	"recreate_pods":              false,
	"render_subchart_notes":      true,
	"replace":                    false,
	"reset_values":               false,
	"reuse_values":               false,
	"skip_crds":                  false,
	"timeout":                    int64(300),
	"verify":                     false,
	"wait":                       true,
	"wait_for_jobs":              false,
}

type releaseMetaData struct {
	AppVersion    types.String `tfsdk:"app_version"`
	Chart         types.String `tfsdk:"chart"`
	Name          types.String `tfsdk:"name"`
	Namespace     types.String `tfsdk:"namespace"`
	Revision      types.Int64  `tfsdk:"revision"`
	Version       types.String `tfsdk:"version"`
	Values        types.String `tfsdk:"values"`
	FirstDeployed types.Int64  `tfsdk:"first_deployed"`
	LastDeployed  types.Int64  `tfsdk:"last_deployed"`
}
type setResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type set_listResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.List   `tfsdk:"value"`
}

type postrenderModel struct {
	Args       types.List   `tfsdk:"args"`
	BinaryPath types.String `tfsdk:"binary_path"`
}

type suppressDescriptionPlanModifier struct{}

func (m suppressDescriptionPlanModifier) Description(ctx context.Context) string {
	return "Suppress changes if the new description is an empty string"
}

func (m suppressDescriptionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressDescriptionPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.ValueString() == "" {
		resp.PlanValue = req.StateValue
	}
}

func suppressDescription() planmodifier.String {
	return suppressDescriptionPlanModifier{}
}

type suppressDevelPlanModifier struct{}

func (m suppressDevelPlanModifier) Description(ctx context.Context) string {
	return "Suppress changes if the version is set"
}

func (m suppressDevelPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressDevelPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	var version types.String
	req.Plan.GetAttribute(ctx, path.Root("version"), &version)
	if !version.IsNull() && version.ValueString() != "" {
		resp.PlanValue = req.StateValue
	}
}

func suppressDevel() planmodifier.Bool {
	return suppressDevelPlanModifier{}
}

// Supress Keyring
type suppressKeyringPlanModifier struct{}

func (m suppressKeyringPlanModifier) Description(ctx context.Context) string {
	return "Suppress changes if verify is false"
}

func (m suppressKeyringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressKeyringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var verify types.Bool
	req.Plan.GetAttribute(ctx, path.Root("verify"), &verify)
	if !verify.IsNull() && !verify.ValueBool() {
		resp.PlanValue = req.StateValue
	}
}

func suppressKeyring() planmodifier.String {
	return suppressKeyringPlanModifier{}
}

type NamespacePlanModifier struct{}

func (m NamespacePlanModifier) Description(context.Context) string {
	return "Sets the namespace value from the HELM_NAMESPACE environment variable or defaults to 'default'."
}

func (m NamespacePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m NamespacePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var namespace types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("namespace"), &namespace)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if namespace.IsNull() || namespace.ValueString() == "" {
		envNamespace := os.Getenv("HELM_NAMESPACE")
		if envNamespace == "" {
			envNamespace = "default"
		}
		resp.PlanValue = types.StringValue(envNamespace)
	}
}

func NewNamespacePlanModifier() planmodifier.String {
	return &NamespacePlanModifier{}
}

func (r *HelmRelease) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

func (r *HelmRelease) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are available in the resource",
		Attributes: map[string]schema.Attribute{
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["atomic"].(bool)),
				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used",
			},
			"cleanup_on_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["cleanup_on_fail"].(bool)),
				Description: "Allow deletion of new resources created in this upgrade when upgrade fails",
			},
			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["create_namespace"].(bool)),
				Description: "Create the namespace if it does not exist",
			},
			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["dependency_update"].(bool)),
				Description: "Run helm dependency update before installing the chart",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Add a custom description",
				PlanModifiers: []planmodifier.String{
					suppressDescription(),
				},
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If 'version' is set, this is ignored",
				PlanModifiers: []planmodifier.Bool{
					suppressDevel(),
				},
			},
			"disable_crd_hooks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["disable_crd_hooks"].(bool)),
				Description: "Prevent CRD hooks from running, but run other hooks. See helm install --no-crd-hook",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["disable_openapi_validation"].(bool)),
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["disable_webhooks"].(bool)),
				Description: "Prevent hooks from running",
			},
			"force_update": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["force_update"].(bool)),
				Description: "Force resource update through delete/recreate if needed.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Description: "Location of public keys used for verification, Used only if 'verify is true'",
				PlanModifiers: []planmodifier.String{
					suppressKeyring(),
				},
			},
			"lint": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["lint"].(bool)),
				Description: "Run helm lint when planning",
			},
			"manifest": schema.StringAttribute{
				Description: "The rendered manifest as JSON.",
				Computed:    true,
			},
			"max_history": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultAttributes["max_history"].(int64)),
				Description: "Limit the maximum number of revisions saved per release. Use 0 for no limit",
			},
			"metadata": schema.SingleNestedAttribute{
				Description: "Status of the deployed release.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"app_version": schema.StringAttribute{
						Computed:    true,
						Description: "The version number of the application being deployed",
					},
					"chart": schema.StringAttribute{
						Computed:    true,
						Description: "The name of the chart",
					},
					"first_deployed": schema.Int64Attribute{
						Computed:    true,
						Description: "FirstDeployed is an int64 which represents timestamp when the release was first deployed.",
					},
					"last_deployed": schema.Int64Attribute{
						Computed:    true,
						Description: "LastDeployed is an int64 which represents timestamp when the release was last deployed.",
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Name is the name of the release",
					},
					"namespace": schema.StringAttribute{
						Computed:    true,
						Description: "Namespace is the kubernetes namespace of the release",
					},
					"revision": schema.Int64Attribute{
						Computed:    true,
						Description: "Version is an int32 which represents the version of the release",
					},
					"values": schema.StringAttribute{
						Computed:    true,
						Description: "Set of extra values. added to the chart. The sensitive data is cloaked. JSON encoded.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "A SemVer 2 conformant version string of the chart",
					},
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 53),
				},
				Description: "Release name. The length must not be longer than 53 characters",
			},
			"namespace": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					NewNamespacePlanModifier(),
				},
				Description: "Namespace to install the release into",
			},
			"pass_credentials": schema.BoolAttribute{
				Optional:    true,
				Description: "Pass credentials to all domains",
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["pass_credentials"].(bool)),
			},
			"recreate_pods": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["recreate_pods"].(bool)),
				Description: "Perform pods restart during upgrade/rollback",
			},
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["render_subchart_notes"].(bool)),
				Description: "If set, render subchart notes along with the parent",
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["replace"].(bool)),
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If it is a URL, the chart is installed without installing the repository",
			},
			"repository_ca_file": schema.StringAttribute{
				Optional:    true,
				Description: "The Repositories CA file",
			},
			"repository_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert file",
			},
			"repository_key_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert key file",
			},
			"repository_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for HTTP basic authentication",
			},
			"repository_username": schema.StringAttribute{
				Optional:    true,
				Description: "Username for HTTP basic authentication",
			},
			"reset_values": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When upgrading, reset the values to the ones built into the chart",
				Default:     booldefault.StaticBool(defaultAttributes["reset_values"].(bool)),
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
				Default:     booldefault.StaticBool(defaultAttributes["reuse_values"].(bool)),
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["skip_crds"].(bool)),
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the release",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultAttributes["timeout"].(int64)),
				Description: "Time in seconds to wait for any individual kubernetes operation",
			},
			"values": schema.ListAttribute{
				Optional:    true,
				Description: "List of values in raw YAML format to pass to helm",
				ElementType: types.StringType,
			},
			"verify": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["verify"].(bool)),
				Description: "Verify the package before installing it.",
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["wait"].(bool)),
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"wait_for_jobs": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultAttributes["wait_for_jobs"].(bool)),
				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
			},
			"set": schema.SetNestedAttribute{
				Description: "Custom values to be merged with the values",
				Optional:    true,
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
							Computed: true,
							Default:  stringdefault.StaticString(""),
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string"),
							},
						},
					},
				},
			},
			"set_list": schema.ListNestedAttribute{
				Description: "Custom sensitive values to be merged with the values",
				Optional:    true,
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
			"set_sensitive": schema.SetNestedAttribute{
				Description: "Custom sensitive values to be merged with the values",
				Optional:    true,
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
			"postrender": schema.SingleNestedAttribute{
				Description: "Postrender command config",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"args": schema.ListAttribute{
						Optional:    true,
						Description: "An argument to the post-renderer (can specify multiple)",
						ElementType: types.StringType,
					},
					"binary_path": schema.StringAttribute{
						Required:    true,
						Description: "The common binary path",
					},
				},
			},
		},
		Version: 1,
	}
}

func (r *HelmRelease) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Ensure that the ProviderData is not nil
	if req.ProviderData == nil {
		return
	}

	// Assert that the ProviderData is of type *Meta
	meta, ok := req.ProviderData.(*Meta)
	if !ok {
		resp.Diagnostics.AddError(
			"Provider Configuration Error",
			fmt.Sprintf("Unexpected ProviderData type: %T", req.ProviderData),
		)
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Configured meta: %+v", meta))
	r.meta = meta
}

const sensitiveContentValue = "(sensitive value)"

func cloakSetValue(values map[string]interface{}, valuePath string) {
	pathKeys := strings.Split(valuePath, ".")
	sensitiveKey := pathKeys[len(pathKeys)-1]
	parentPathKeys := pathKeys[:len(pathKeys)-1]
	m := values
	for _, key := range parentPathKeys {
		v, ok := m[key].(map[string]interface{})
		if !ok {
			return
		}
		m = v
	}
	m[sensitiveKey] = sensitiveContentValue
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if vMap, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bvMap, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bvMap, vMap)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func (r *HelmRelease) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var state HelmReleaseModel
	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Plan state on Create: %+v", state))

	meta := r.meta
	if meta == nil {
		resp.Diagnostics.AddError("Initialization Error", "Meta instance is not initialized")
		return
	}
	namespace := state.Namespace.ValueString()
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError("Error getting helm configuration", fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err))
		return
	}
	ociDiags := OCIRegistryLogin(ctx, meta, actionConfig, meta.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.RepositoryUsername.ValueString(), state.RepositoryPassword.ValueString())
	resp.Diagnostics.Append(ociDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := action.NewInstall(actionConfig)
	cpo, chartName, cpoDiags := chartPathOptions(&state, meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(cpoDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, path, chartDiags := getChart(ctx, &state, meta, chartName, cpo)
	resp.Diagnostics.Append(chartDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, depDiags := checkChartDependencies(ctx, &state, c, path, meta)
	resp.Diagnostics.Append(depDiags...)
	if resp.Diagnostics.HasError() {
		return
	} else if updated {
		c, err = loader.Load(path)
		if err != nil {
			resp.Diagnostics.AddError("Error loading chart", fmt.Sprintf("Could not load chart: %s", err))
			return
		}
	}

	values, valuesDiags := getValues(ctx, &state)
	resp.Diagnostics.Append(valuesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err = isChartInstallable(c)
	if err != nil {
		resp.Diagnostics.AddError("Error checking if chart is installable", fmt.Sprintf("Chart is not installable: %s", err))
		return
	}

	client.ClientOnly = false
	client.DryRun = false
	client.DisableHooks = state.DisableWebhooks.ValueBool()
	client.Wait = state.Wait.ValueBool()
	client.WaitForJobs = state.WaitForJobs.ValueBool()
	client.Devel = state.Devel.ValueBool()
	client.DependencyUpdate = state.DependencyUpdate.ValueBool()
	client.Timeout = time.Duration(state.Timeout.ValueInt64()) * time.Second
	client.Namespace = state.Namespace.ValueString()
	client.ReleaseName = state.Name.ValueString()
	client.Atomic = state.Atomic.ValueBool()
	client.SkipCRDs = state.SkipCrds.ValueBool()
	client.SubNotes = state.RenderSubchartNotes.ValueBool()
	client.DisableOpenAPIValidation = state.DisableOpenapiValidation.ValueBool()
	client.Replace = state.Replace.ValueBool()
	client.Description = state.Description.ValueString()
	client.CreateNamespace = state.CreateNamespace.ValueBool()

	if !state.Postrender.IsNull() {
		tflog.Debug(ctx, "Postrender is not null")
		var postrenderConfig postrenderModel
		postrenderDiags := state.Postrender.As(ctx, &postrenderConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(postrenderDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("Postrender config: %+v", postrenderConfig))

		binaryPath := postrenderConfig.BinaryPath.ValueString()
		argsList := postrenderConfig.Args.Elements()

		var args []string
		for _, arg := range argsList {
			args = append(args, arg.(basetypes.StringValue).ValueString())
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating post-renderer with binary path: %s and args: %v", binaryPath, args))
		pr, err := postrender.NewExec(binaryPath, args...)
		if err != nil {
			resp.Diagnostics.AddError("Error creating post-renderer", fmt.Sprintf("Could not create post-renderer: %s", err))
			return
		}

		client.PostRenderer = pr
	}

	rel, err := client.Run(c, values)
	if err != nil && rel == nil {
		resp.Diagnostics.AddError("installation failed", err.Error())
		return
	}

	if err != nil && rel != nil {
		exists, existsDiags := resourceReleaseExists(ctx, state.Name.ValueString(), state.Namespace.ValueString(), meta)
		resp.Diagnostics.Append(existsDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !exists {
			resp.Diagnostics.AddError("installation failed", err.Error())
			return
		}

		diags := setReleaseAttributes(ctx, &state, rel, meta)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(diag.NewWarningDiagnostic("Helm release created with warnings", fmt.Sprintf("Helm release %q was created but has a failed status. Use the `helm` command to investigate the error, correct it, then run Terraform again.", client.ReleaseName)))
		resp.Diagnostics.Append(diag.NewErrorDiagnostic("Helm release error", err.Error()))

		return
	}

	diags = setReleaseAttributes(ctx, &state, rel, meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *HelmRelease) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HelmReleaseModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Current state before changes: %+v", state))

	meta := r.meta
	if meta == nil {
		resp.Diagnostics.AddError(
			"Meta not set",
			"The meta information is not set for the resource",
		)
		return
	}

	exists, diags := resourceReleaseExists(ctx, state.Name.ValueString(), state.Namespace.ValueString(), meta)
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	logID := fmt.Sprintf("[resourceReleaseRead: %s]", state.Name.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("%s Started", logID))

	c, err := meta.GetHelmConfiguration(ctx, state.Namespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting helm configuration",
			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", state.Namespace.ValueString(), err),
		)
		return
	}

	release, err := getRelease(ctx, meta, c, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting release",
			fmt.Sprintf("Unable to get Helm release %s: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}

	diags = setReleaseAttributes(ctx, &state, release, meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError(
			"Error setting release attributes",
			fmt.Sprintf("Unable to set attributes for helm release %s", state.Name.ValueString()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HelmRelease) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HelmReleaseModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Current state of the resource before update operation is applied
	var state HelmReleaseModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	logID := fmt.Sprintf("[resourceReleaseUpdate: %s]", state.Name.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("%s Started", logID))

	meta := r.meta
	namespace := state.Namespace.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("%s Getting helm configuration for namespace: %s", logID, namespace))
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("%s Failed to get helm configuration: %v", logID, err))
		resp.Diagnostics.AddError("Error getting helm configuration", fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err))
		return
	}
	ociDiags := OCIRegistryLogin(ctx, meta, actionConfig, meta.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.RepositoryUsername.ValueString(), state.RepositoryPassword.ValueString())
	resp.Diagnostics.Append(ociDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	client := action.NewUpgrade(actionConfig)

	cpo, chartName, cpoDiags := chartPathOptions(&plan, meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(cpoDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, path, chartDiags := getChart(ctx, &plan, meta, chartName, cpo)
	resp.Diagnostics.Append(chartDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check and update the chart's depenedcies if it's needed
	updated, depDiags := checkChartDependencies(ctx, &plan, c, path, meta)
	resp.Diagnostics.Append(depDiags...)
	if resp.Diagnostics.HasError() {
		return
	} else if updated {
		c, err = loader.Load(path)
		if err != nil {
			resp.Diagnostics.AddError("Error loading chart", fmt.Sprintf("Could not load chart: %s", err))
			return
		}
	}

	client.Devel = plan.Devel.ValueBool()
	client.Namespace = plan.Namespace.ValueString()
	client.Timeout = time.Duration(plan.Timeout.ValueInt64()) * time.Second
	client.Wait = plan.Wait.ValueBool()
	client.WaitForJobs = plan.WaitForJobs.ValueBool()
	client.DryRun = false
	client.DisableHooks = plan.DisableWebhooks.ValueBool()
	client.Atomic = plan.Atomic.ValueBool()
	client.SkipCRDs = plan.SkipCrds.ValueBool()
	client.SubNotes = plan.RenderSubchartNotes.ValueBool()
	client.DisableOpenAPIValidation = plan.DisableOpenapiValidation.ValueBool()
	client.Force = plan.ForceUpdate.ValueBool()
	client.ResetValues = plan.ResetValues.ValueBool()
	client.ReuseValues = plan.ReuseValues.ValueBool()
	client.Recreate = plan.RecreatePods.ValueBool()
	client.MaxHistory = int(plan.MaxHistory.ValueInt64())
	client.CleanupOnFail = plan.CleanupOnFail.ValueBool()
	client.Description = plan.Description.ValueString()

	if !plan.Postrender.IsNull() {
		var postrenderConfig postrenderModel
		postrenderDiags := plan.Postrender.As(ctx, &postrenderConfig, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(postrenderDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("Initial postrender values update method: %+v", postrenderConfig))

		binaryPath := postrenderConfig.BinaryPath.ValueString()
		argsList := postrenderConfig.Args.Elements()

		var args []string
		for _, arg := range argsList {
			args = append(args, arg.(basetypes.StringValue).ValueString())
		}
		tflog.Debug(ctx, fmt.Sprintf("Binary path update method: %s, Args: %v", binaryPath, args))
		pr, err := postrender.NewExec(binaryPath, args...)
		if err != nil {
			resp.Diagnostics.AddError("Error creating post-renderer", fmt.Sprintf("Could not create post-renderer: %s", err))
			return
		}
		client.PostRenderer = pr
	}
	values, valuesDiags := getValues(ctx, &plan)
	resp.Diagnostics.Append(valuesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	release, err := client.Run(name, c, values)
	if err != nil {
		resp.Diagnostics.AddError("Error upgrading chart", fmt.Sprintf("Upgrade failed: %s", err))
		return
	}

	diags = setReleaseAttributes(ctx, &plan, release, meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *HelmRelease) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HelmReleaseModel
	diags := req.State.Get(ctx, &state)

	for _, diag := range diags {
		tflog.Debug(ctx, fmt.Sprintf("Diagnostics after state get: %s", diag.Detail()))
	}

	// Append diagnostics to response
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("Error retrieving state: %v", resp.Diagnostics))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Retrieved state: %+v", state))

	// Check if meta is set
	meta := r.meta
	if meta == nil {
		resp.Diagnostics.AddError(
			"Meta not set",
			"The meta information is not set for the resource",
		)
		tflog.Error(ctx, "Meta information is not set for the resource")
		return
	}

	name := state.Name.ValueString()
	namespace := state.Namespace.ValueString()

	exists, diags := resourceReleaseExists(ctx, name, namespace, meta)
	if !exists {
		return
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get Helm configuration
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting helm configuration",
			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
		)
		tflog.Error(ctx, fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Retrieved Helm configuration for namespace: %s", namespace))

	// Initialize uninstall action
	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = state.Wait.ValueBool()
	uninstall.DisableHooks = state.DisableWebhooks.ValueBool()
	uninstall.Timeout = time.Duration(state.Timeout.ValueInt64()) * time.Second

	// Uninstall the release
	tflog.Info(ctx, fmt.Sprintf("Uninstalling Helm release: %s", name))
	res, err := uninstall.Run(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error uninstalling release",
			fmt.Sprintf("Unable to uninstall Helm release %s: %s", name, err),
		)
		tflog.Error(ctx, fmt.Sprintf("Unable to uninstall Helm release %s: %s", name, err))
		return
	}

	if res.Info != "" {
		resp.Diagnostics.Append(diag.NewWarningDiagnostic(
			"Helm uninstall returned an information message",
			res.Info,
		))
	}
}

func chartPathOptions(model *HelmReleaseModel, meta *Meta, cpo *action.ChartPathOptions) (*action.ChartPathOptions, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	chartName := model.Chart.ValueString()
	repository := model.Repository.ValueString()

	var repositoryURL string
	if registry.IsOCI(repository) {
		// LocateChart expects the chart name to contain the full OCI path
		u, err := url.Parse(repository)
		if err != nil {
			diags.AddError("Invalid Repository URL", fmt.Sprintf("Failed to parse repository URL %s: %s", repository, err))
			return nil, "", diags
		}
		u.Path = pathpkg.Join(u.Path, chartName)
		chartName = u.String()
	} else {
		var err error
		repositoryURL, chartName, err = buildChartNameWithRepository(repository, strings.TrimSpace(chartName))
		if err != nil {
			diags.AddError("Error building Chart Name With Repository", fmt.Sprintf("Could not build Chart Name With Repository %s and chart %s: %s", repository, chartName, err))
			return nil, "", diags
		}
	}

	version := getVersion(model)

	cpo.CaFile = model.RepositoryCaFile.ValueString()
	cpo.CertFile = model.RepositoryCertFile.ValueString()
	cpo.KeyFile = model.RepositoryKeyFile.ValueString()
	cpo.Keyring = model.Keyring.ValueString()
	cpo.RepoURL = repositoryURL
	cpo.Verify = model.Verify.ValueBool()
	if !useChartVersion(chartName, cpo.RepoURL) {
		cpo.Version = version
	}
	cpo.Username = model.RepositoryUsername.ValueString()
	cpo.Password = model.RepositoryPassword.ValueString()
	cpo.PassCredentialsAll = model.PassCredentials.ValueBool()

	return cpo, chartName, diags
}

func useChartVersion(chart string, repo string) bool {
	// checks if chart is a URL or OCI registry

	if _, err := url.ParseRequestURI(chart); err == nil && !registry.IsOCI(chart) {
		return true
	}
	// checks if chart is a local chart
	if _, err := os.Stat(chart); err == nil {
		return true
	}
	// checks if repo is a local chart
	if _, err := os.Stat(repo); err == nil {
		return true
	}

	return false
}

func buildChartNameWithRepository(repository, name string) (string, string, error) {
	_, err := url.ParseRequestURI(repository)
	if err == nil {
		return repository, name, nil
	}

	if strings.Index(name, "/") == -1 && repository != "" {
		name = fmt.Sprintf("%s/%s", repository, name)
	}

	return "", name, nil
}

func getVersion(model *HelmReleaseModel) string {
	version := model.Version.ValueString()
	if version == "" && model.Devel.ValueBool() {
		return ">0.0.0-0"
	}
	return strings.TrimSpace(version)
}

func isChartInstallable(ch *chart.Chart) error {
	chartType := ch.Metadata.Type
	if strings.EqualFold(chartType, "library") {
		return errors.Errorf("library charts are not installable")
	}
	return nil
}

func getChart(ctx context.Context, model *HelmReleaseModel, m *Meta, name string, cpo *action.ChartPathOptions) (*chart.Chart, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	tflog.Debug(ctx, fmt.Sprintf("Helm settings: %+v", m.Settings))

	path, err := cpo.LocateChart(name, m.Settings)
	if err != nil {
		diags.AddError("Error locating chart", fmt.Sprintf("Unable to locate chart %s: %s", name, err))
		return nil, "", diags
	}

	c, err := loader.Load(path)
	if err != nil {
		diags.AddError("Error loading chart", fmt.Sprintf("Unable to load chart %s: %s", path, err))
		return nil, "", diags
	}

	return c, path, diags
}

func getValues(ctx context.Context, model *HelmReleaseModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

	// Processing "values" attribute
	for _, raw := range model.Values.Elements() {
		if raw.IsNull() {
			continue
		}

		value, ok := raw.(types.String)
		if !ok {
			diags.AddError("Type Error", fmt.Sprintf("Expected types.String, got %T", raw))
			return nil, diags
		}

		values := value.ValueString()
		if values == "" {
			continue
		}

		currentMap := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(values), &currentMap); err != nil {
			diags.AddError("Error unmarshaling values", fmt.Sprintf("---> %v %s", err, values))
			return nil, diags
		}

		base = mergeMaps(base, currentMap)
	}

	// Processing "set" attribute
	if !model.Set.IsNull() {
		tflog.Debug(ctx, "Processing Set attribute")
		var setList []setResourceModel
		setDiags := model.Set.ElementsAs(ctx, &setList, false)
		diags.Append(setDiags...)
		if diags.HasError() {
			return nil, diags
		}

		for i, set := range setList {
			tflog.Debug(ctx, fmt.Sprintf("Processing Set element at index %d: %v", i, set))
			setDiags := getValue(base, set)
			diags.Append(setDiags...)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("Error occurred while processing Set element at index %d", i))
				return nil, diags
			}
		}
	}

	// Processing "set_list" attribute
	if !model.SetList.IsUnknown() {
		tflog.Debug(ctx, "Processing Set_list attribute")
		var setListSlice []set_listResourceModel
		setListDiags := model.SetList.ElementsAs(ctx, &setListSlice, false)
		diags.Append(setListDiags...)
		if diags.HasError() {
			tflog.Debug(ctx, "Error occurred while processing Set_list attribute")
			return nil, diags
		}

		for i, setList := range setListSlice {
			tflog.Debug(ctx, fmt.Sprintf("Processing Set_list element at index %d: %v", i, setList))
			setListDiags := getListValue(ctx, base, setList)
			diags.Append(setListDiags...)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("Error occurred while processing Set_list element at index %d", i))
				return nil, diags
			}
		}
	}

	// Processing "set_sensitive" attribute
	if !model.SetSensitive.IsNull() {
		tflog.Debug(ctx, "Processing Set_Sensitive attribute")
		var setSensitiveList []setResourceModel
		setSensitiveDiags := model.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
		diags.Append(setSensitiveDiags...)
		if diags.HasError() {
			tflog.Debug(ctx, "Error occurred while processing Set_Sensitive attribute")
			return nil, diags
		}

		for i, setSensitive := range setSensitiveList {
			tflog.Debug(ctx, fmt.Sprintf("Processing Set_Sensitive element at index %d: %v", i, setSensitive))
			setSensitiveDiags := getValue(base, setSensitive)
			diags.Append(setSensitiveDiags...)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("Error occurred while processing Set_Sensitive element at index %d", i))
				return nil, diags
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Final merged values: %v", base))
	logDiags := logValues(ctx, base, model)
	diags.Append(logDiags...)
	if diags.HasError() {
		tflog.Debug(ctx, "Error occurred while logging values")
		return nil, diags
	}

	return base, diags
}

func getValue(base map[string]interface{}, set setResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()
	value := set.Value.ValueString()
	valueType := set.Type.ValueString()

	switch valueType {
	case "auto", "":
		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, value, err))
			return diags
		}
	case "string":
		if err := strvals.ParseIntoString(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing string value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, value, err))
			return diags
		}
	default:
		diags.AddError("Unexpected type", fmt.Sprintf("Unexpected type: %s", valueType))
		return diags
	}
	return diags
}

func logValues(ctx context.Context, values map[string]interface{}, state *HelmReleaseModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Copy array to avoid changing values by the cloak function.
	asJSON, err := json.Marshal(values)
	if err != nil {
		diags.AddError("Error marshaling values to JSON", fmt.Sprintf("Failed to marshal values to JSON: %s", err))
		return diags
	}

	var c map[string]interface{}
	err = json.Unmarshal(asJSON, &c)
	if err != nil {
		diags.AddError("Error unmarshaling JSON to map", fmt.Sprintf("Failed to unmarshal JSON to map: %s", err))
		return diags
	}

	cloakSetValues(c, state)

	y, err := yaml.Marshal(c)
	if err != nil {
		diags.AddError("Error marshaling map to YAML", fmt.Sprintf("Failed to marshal map to YAML: %s", err))
		return diags
	}

	tflog.Debug(ctx, fmt.Sprintf("---[ values.yaml ]-----------------------------------\n%s\n", string(y)))

	return diags
}

func cloakSetValues(config map[string]interface{}, state *HelmReleaseModel) {
	if !state.SetSensitive.IsNull() {
		var setSensitiveList []setResourceModel
		diags := state.SetSensitive.ElementsAs(context.Background(), &setSensitiveList, false)
		if diags.HasError() {
			// Handle diagnostics error
			return
		}

		for _, set := range setSensitiveList {
			cloakSetValue(config, set.Name.ValueString())
		}
	}
}

func getListValue(ctx context.Context, base map[string]interface{}, set set_listResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()

	if set.Value.IsNull() {
		diags.AddError("Null List Value", "The list value is null.")
		return diags
	}

	// Get the elements of the ListValue
	elements := set.Value.Elements()

	// Convert elements to a list of strings
	listStringArray := make([]string, 0, len(elements))
	for _, element := range elements {
		if !element.IsNull() {
			strValue := element.(types.String).ValueString()
			listStringArray = append(listStringArray, strValue)
		}
	}

	// Join the list into a single string
	listString := strings.Join(listStringArray, ",")

	if err := strvals.ParseInto(fmt.Sprintf("%s={%s}", name, listString), base); err != nil {
		diags.AddError("Error parsing list value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, listString, err))
		return diags
	}

	return diags
}

func setReleaseAttributes(ctx context.Context, state *HelmReleaseModel, r *release.Release, meta *Meta) diag.Diagnostics {
	var diags diag.Diagnostics

	// Update state with attributes from the helm release
	state.Name = types.StringValue(r.Name)
	state.Version = types.StringValue(r.Chart.Metadata.Version)
	state.Namespace = types.StringValue(r.Namespace)
	state.Status = types.StringValue(r.Info.Status.String())

	state.ID = types.StringValue(r.Name)

	// Cloak sensitive values in the release config
	cloakSetValues(r.Config, state)
	values := "{}"
	if r.Config != nil {
		v, err := json.Marshal(r.Config)
		if err != nil {
			diags.AddError(
				"Error marshaling values",
				fmt.Sprintf("unable to marshal values: %s", err),
			)
			return diags
		}
		values = string(v)
	}

	// Handling the helm release if manifest experiment is enabled
	if meta.ExperimentEnabled("manifest") {
		jsonManifest, err := convertYAMLManifestToJSON(r.Manifest)
		if err != nil {
			diags.AddError(
				"Error converting manifest to JSON",
				fmt.Sprintf("Unable to convert manifest to JSON: %s", err),
			)
			return diags
		}
		sensitiveValues := extractSensitiveValues(state)
		manifest := redactSensitiveValues(string(jsonManifest), sensitiveValues)
		state.Manifest = types.StringValue(manifest)
	}

	// Create metadata as a slice of maps
	metadata := map[string]attr.Value{
		"name":           types.StringValue(r.Name),
		"revision":       types.Int64Value(int64(r.Version)),
		"namespace":      types.StringValue(r.Namespace),
		"chart":          types.StringValue(r.Chart.Metadata.Name),
		"version":        types.StringValue(r.Chart.Metadata.Version),
		"app_version":    types.StringValue(r.Chart.Metadata.AppVersion),
		"values":         types.StringValue(values),
		"first_deployed": types.Int64Value(r.Info.FirstDeployed.Unix()),
		"last_deployed":  types.Int64Value(r.Info.LastDeployed.Unix()),
	}

	// Convert the list of ObjectValues to a ListValue
	metadataObject, diag := types.ObjectValue(metadataAttrTypes(), metadata)
	diags.Append(diag...)
	if diags.HasError() {
		tflog.Error(ctx, "Error converting metadata to ListValue", map[string]interface{}{
			"metadata": metadata,
			"error":    diags,
		})

		return diags
	}

	// Log metadata after conversion
	tflog.Debug(ctx, fmt.Sprintf("Metadata after conversion: %+v", metadataObject))
	state.Metadata = metadataObject
	return diags
}

func metadataAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":           types.StringType,
		"revision":       types.Int64Type,
		"namespace":      types.StringType,
		"chart":          types.StringType,
		"version":        types.StringType,
		"app_version":    types.StringType,
		"values":         types.StringType,
		"first_deployed": types.Int64Type,
		"last_deployed":  types.Int64Type,
	}
}

func extractSensitiveValues(state *HelmReleaseModel) map[string]string {
	sensitiveValues := make(map[string]string)

	if !state.SetSensitive.IsNull() {
		var setSensitiveList []setResourceModel
		diags := state.SetSensitive.ElementsAs(context.Background(), &setSensitiveList, false)
		if diags.HasError() {
			return sensitiveValues
		}

		for _, set := range setSensitiveList {
			sensitiveValues[set.Name.ValueString()] = "(sensitive value)"
		}
	}

	return sensitiveValues
}

func (m *Meta) ExperimentEnabled(name string) bool {
	if enabled, exists := m.Experiments[name]; exists {
		return enabled
	}
	return false
}

// c
func resourceReleaseExists(ctx context.Context, name, namespace string, meta *Meta) (bool, diag.Diagnostics) {
	logID := fmt.Sprintf("[resourceReleaseExists: %s]", name)
	tflog.Debug(ctx, fmt.Sprintf("%s Start", logID))

	var diags diag.Diagnostics

	c, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		diags.AddError(
			"Error getting helm configuration",
			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
		)
		return false, diags
	}

	_, err = getRelease(ctx, meta, c, name)

	tflog.Debug(ctx, fmt.Sprintf("%s Done", logID))

	if err == nil {
		return true, diags
	}

	if err == errReleaseNotFound {
		return false, diags
	}

	diags.AddError(
		"Error checking release existence",
		fmt.Sprintf("Error checking release %s in namespace %s: %s", name, namespace, err),
	)
	return false, diags
}

var errReleaseNotFound = fmt.Errorf("release: not found")

// c
func getRelease(ctx context.Context, m *Meta, cfg *action.Configuration, name string) (*release.Release, error) {
	get := action.NewGet(cfg)
	tflog.Debug(ctx, fmt.Sprintf("%s getRelease post action created", name))

	res, err := get.Run(name)
	tflog.Debug(ctx, fmt.Sprintf("%s getRelease post run", name))

	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("getRelease for %s occurred", name))
		tflog.Debug(ctx, fmt.Sprintf("%v", err))
		if strings.Contains(err.Error(), "release: not found") {
			tflog.Error(ctx, errReleaseNotFound.Error())
			return nil, errReleaseNotFound
		}
		tflog.Debug(ctx, fmt.Sprintf("Could not get release %s", err))
		tflog.Error(ctx, err.Error())
		return nil, err
	}

	tflog.Debug(ctx, fmt.Sprintf("%s getRelease completed", name))
	return res, nil
}

// c
func checkChartDependencies(ctx context.Context, model *HelmReleaseModel, c *chart.Chart, path string, m *Meta) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	p := getter.All(m.Settings)

	if req := c.Metadata.Dependencies; req != nil {
		err := action.CheckDependencies(c, req)
		if err != nil {
			if model.DependencyUpdate.ValueBool() {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          model.Keyring.ValueString(),
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: m.Settings.RepositoryConfig,
					RepositoryCache:  m.Settings.RepositoryCache,
					Debug:            m.Settings.Debug,
				}
				tflog.Debug(ctx, "Downloading chart dependencies...")
				if err := man.Update(); err != nil {
					diags.AddError("", fmt.Sprintf("Failed to update chart dependencies: %s", err))
					return true, diags
				}
				return true, diags
			}
			diags.AddError("", "Found in Chart.yaml, but missing in charts/ directory")
			return false, diags
		}
	}
	tflog.Debug(ctx, "Chart dependencies are up to date.")
	return false, diags
}

func (r *HelmRelease) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// resource is being destroyed
		return
	}
	var plan HelmReleaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state *HelmReleaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Plan state on ModifyPlan: %+v", plan))
	tflog.Debug(ctx, fmt.Sprintf("Actual state on ModifyPlan: %+v", state))

	logID := fmt.Sprintf("[resourceDiff: %s]", plan.Name.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("%s Start", logID))

	meta := r.meta
	name := plan.Name.ValueString()
	namespace := plan.Namespace.ValueString()

	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Helm configuration", err.Error())
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Initial Values: Name=%s, Namespace=%s, Repository=%s, Repository_Username=%s, Repository_Password=%s, Chart=%s", logID,
		name, namespace, plan.Repository.ValueString(), plan.RepositoryUsername.ValueString(), plan.RepositoryPassword.ValueString(), plan.Chart.ValueString()))

	repositoryURL := plan.Repository.ValueString()
	repositoryUsername := plan.RepositoryUsername.ValueString()
	repositoryPassword := plan.RepositoryPassword.ValueString()
	chartName := plan.Chart.ValueString()
	ociDiags := OCIRegistryLogin(ctx, meta, actionConfig, meta.RegistryClient, repositoryURL, chartName, repositoryUsername, repositoryPassword)
	resp.Diagnostics.Append(ociDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always set desired state to DEPLOYED
	plan.Status = types.StringValue(release.StatusDeployed.String())

	if recomputeMetadata(plan, state) {
		tflog.Debug(ctx, fmt.Sprintf("%s Metadata has changes, setting to unknown", logID))
		plan.Metadata = types.ObjectUnknown(metadataAttrTypes())
	}

	if !useChartVersion(plan.Chart.ValueString(), plan.Repository.ValueString()) {
		// Check if version has changed
		if state != nil && !plan.Version.Equal(state.Version) {

			// Ensure trimming 'v' prefix correctly
			oldVersionStr := strings.TrimPrefix(state.Version.String(), "v")
			newVersionStr := strings.TrimPrefix(plan.Version.String(), "v")

			if oldVersionStr != newVersionStr && newVersionStr != "" {
				// Setting Metadata to a computed value
				plan.Metadata = types.ObjectUnknown(metadataAttrTypes())
			}
		}
	}

	client := action.NewInstall(actionConfig)
	cpo, chartName, diags := chartPathOptions(&plan, meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	chart, path, diags := getChart(ctx, &plan, meta, chartName, cpo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Got chart", logID))

	updated, diags := checkChartDependencies(ctx, &plan, chart, path, meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	} else if updated {
		chart, err = loader.Load(path)
		if err != nil {
			resp.Diagnostics.AddError("Error loading chart", err.Error())
			return
		}
	}

	if plan.Lint.ValueBool() {
		diags := resourceReleaseValidate(ctx, &plan, meta, cpo)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Release validated", logID))

	if meta.ExperimentEnabled("manifest") {
		// Check if all necessary values are known
		if valuesUnknown(plan) {
			tflog.Debug(ctx, "not all values are known, skipping dry run to render manifest")
			plan.Manifest = types.StringNull()
			plan.Version = types.StringNull()
			return
		}

		var postRenderer postrender.PostRenderer
		if !plan.Postrender.IsNull() {
			// Extract the list of postrender configurations
			var postrenderConfig postrenderModel
			postrenderDiags := plan.Postrender.As(ctx, &postrenderConfig, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(postrenderDiags...)
			if resp.Diagnostics.HasError() {
				return
			}

			binaryPath := postrenderConfig.BinaryPath.ValueString()
			argsList := postrenderConfig.Args.Elements()

			var args []string
			for _, arg := range argsList {
				args = append(args, arg.(basetypes.StringValue).ValueString())
			}

			pr, err := postrender.NewExec(binaryPath, args...)
			if err != nil {
				resp.Diagnostics.AddError("Error creating post-renderer", fmt.Sprintf("Could not create post-renderer: %s", err))
				return
			}

			client.PostRenderer = pr
		}
		if state == nil {
			install := action.NewInstall(actionConfig)
			install.ChartPathOptions = *cpo
			install.DryRun = true
			install.DisableHooks = plan.DisableWebhooks.ValueBool()
			install.Wait = plan.Wait.ValueBool()
			install.WaitForJobs = plan.WaitForJobs.ValueBool()
			install.Devel = plan.Devel.ValueBool()
			install.DependencyUpdate = plan.DependencyUpdate.ValueBool()
			install.Timeout = time.Duration(plan.Timeout.ValueInt64()) * time.Second
			install.Namespace = plan.Namespace.ValueString()
			install.ReleaseName = plan.Name.ValueString()
			install.Atomic = plan.Atomic.ValueBool()
			install.SkipCRDs = plan.SkipCrds.ValueBool()
			install.SubNotes = plan.RenderSubchartNotes.ValueBool()
			install.DisableOpenAPIValidation = plan.DisableOpenapiValidation.ValueBool()
			install.Replace = plan.Replace.ValueBool()
			install.Description = plan.Description.ValueString()
			install.CreateNamespace = plan.CreateNamespace.ValueBool()
			install.PostRenderer = postRenderer

			values, diags := getValues(ctx, &plan)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			tflog.Debug(ctx, fmt.Sprintf("%s performing dry run install", logID))
			dry, err := install.Run(chart, values)
			if err != nil {
				// NOTE if the cluster is not reachable then we can't run the install
				// this will happen if the user has their cluster creation in the
				// same apply. We are catching this case here and marking manifest
				// as computed to avoid breaking existing configs

				if strings.Contains(err.Error(), "Kubernetes cluster unreachable") {
					// FIXME add diagnostic here

					resp.Diagnostics.AddError("cluster was unreachable at create time, marking manifest as computed", err.Error())
					plan.Manifest = types.StringNull()
					return
				}
				resp.Diagnostics.AddError("Error performing dry run install", err.Error())
				return
			}

			jsonManifest, err := convertYAMLManifestToJSON(dry.Manifest)
			if err != nil {
				resp.Diagnostics.AddError("Error converting YAML manifest to JSON", err.Error())
				return
			}
			valuesMap := make(map[string]string)
			if !plan.SetSensitive.IsNull() {
				var setSensitiveList []setResourceModel
				setSensitiveDiags := plan.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
				resp.Diagnostics.Append(setSensitiveDiags...)
				if resp.Diagnostics.HasError() {
					return
				}

				for _, set := range setSensitiveList {
					valuesMap[set.Name.ValueString()] = set.Value.ValueString()
				}
			}
			manifest := redactSensitiveValues(string(jsonManifest), valuesMap)
			plan.Manifest = types.StringValue(manifest)
			return
		}

		_, err = getRelease(ctx, meta, actionConfig, name)
		if err == errReleaseNotFound {
			if len(chart.Metadata.Version) > 0 {
				plan.Version = types.StringValue(chart.Metadata.Version)
			}
			plan.Manifest = types.StringNull()
			return
		} else if err != nil {
			resp.Diagnostics.AddError("Error retrieving old release for a diff", err.Error())
			return
		}

		upgrade := action.NewUpgrade(actionConfig)
		upgrade.ChartPathOptions = *cpo
		upgrade.Devel = plan.Devel.ValueBool()
		upgrade.Namespace = plan.Namespace.ValueString()
		upgrade.Timeout = time.Duration(plan.Timeout.ValueInt64()) * time.Second
		upgrade.Wait = plan.Wait.ValueBool()
		upgrade.DryRun = true
		upgrade.DisableHooks = plan.DisableWebhooks.ValueBool()
		upgrade.Atomic = plan.Atomic.ValueBool()
		upgrade.SubNotes = plan.RenderSubchartNotes.ValueBool()
		upgrade.WaitForJobs = plan.WaitForJobs.ValueBool()
		upgrade.Force = plan.ForceUpdate.ValueBool()
		upgrade.ResetValues = plan.ResetValues.ValueBool()
		upgrade.ReuseValues = plan.ReuseValues.ValueBool()
		upgrade.Recreate = plan.RecreatePods.ValueBool()
		upgrade.MaxHistory = int(plan.MaxHistory.ValueInt64())
		upgrade.CleanupOnFail = plan.CleanupOnFail.ValueBool()
		upgrade.Description = plan.Description.ValueString()
		upgrade.PostRenderer = postRenderer

		values, diags := getValues(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		tflog.Debug(ctx, fmt.Sprintf("%s performing dry run upgrade", logID))
		dry, err := upgrade.Run(name, chart, values)
		if err != nil && strings.Contains(err.Error(), "has no deployed releases") {
			if len(chart.Metadata.Version) > 0 && cpo.Version != "" {
				plan.Version = types.StringValue(chart.Metadata.Version)
			}
			plan.Version = types.StringNull()
			plan.Manifest = types.StringNull()
			return
		} else if err != nil {
			resp.Diagnostics.AddError("Error running dry run for a diff", err.Error())
			return
		}

		jsonManifest, err := convertYAMLManifestToJSON(dry.Manifest)
		if err != nil {
			resp.Diagnostics.AddError("Error converting YAML manifest to JSON", err.Error())
			return
		}
		valuesMap := make(map[string]string)
		if !plan.SetSensitive.IsNull() {
			var setSensitiveList []setResourceModel
			setSensitiveDiags := plan.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
			resp.Diagnostics.Append(setSensitiveDiags...)
			if resp.Diagnostics.HasError() {
				return
			}

			for _, set := range setSensitiveList {
				valuesMap[set.Name.ValueString()] = set.Value.ValueString()
			}
		}
		manifest := redactSensitiveValues(string(jsonManifest), valuesMap)
		plan.Manifest = types.StringValue(manifest)
		tflog.Debug(ctx, fmt.Sprintf("%s set manifest: %s", logID, jsonManifest))
	} else {
		plan.Manifest = types.StringNull()
	}

	tflog.Debug(ctx, fmt.Sprintf("%s Done", logID))

	if len(chart.Metadata.Version) > 0 {
		plan.Version = types.StringValue(chart.Metadata.Version)
	} else {
		state.Version = types.StringNull()
	}
	resp.Plan.Set(ctx, &plan)
}

// TODO: write unit test, always returns true for recomputing the metadata
// returns true if any metadata fields have changed
func recomputeMetadata(plan HelmReleaseModel, state *HelmReleaseModel) bool {
	if state == nil {
		return true
	}

	if !plan.Chart.Equal(state.Chart) {
		return true
	}
	if !plan.Repository.Equal(state.Repository) {
		return true
	}
	if !plan.Values.Equal(state.Values) {
		return true
	}
	if !plan.Set.Equal(state.Set) {
		return true
	}
	if !plan.SetSensitive.Equal(state.SetSensitive) {
		return true
	}
	if !plan.SetList.Equal(state.SetList) {
		return true
	}
	return false
}

func resourceReleaseValidate(ctx context.Context, model *HelmReleaseModel, meta *Meta, cpo *action.ChartPathOptions) diag.Diagnostics {
	var diags diag.Diagnostics

	cpo, name, chartDiags := chartPathOptions(model, meta, cpo)
	diags.Append(chartDiags...)
	if diags.HasError() {
		diags.AddError("Malformed values", fmt.Sprintf("Chart path options error: %s", chartDiags))
		return diags
	}

	values, valuesDiags := getValues(ctx, model)
	diags.Append(valuesDiags...)
	if diags.HasError() {
		return diags
	}

	lintDiags := lintChart(meta, name, cpo, values)
	if lintDiags != nil {
		diagnostic := diag.NewErrorDiagnostic("Lint Error", lintDiags.Error())
		diags = append(diags, diagnostic)
	}
	return diags
}

func lintChart(m *Meta, name string, cpo *action.ChartPathOptions, values map[string]interface{}) error {
	path, err := cpo.LocateChart(name, m.Settings)
	if err != nil {
		return err
	}

	l := action.NewLint()
	result := l.Run([]string{path}, values)

	return resultToError(result)
}

func resultToError(r *action.LintResult) error {
	if len(r.Errors) == 0 {
		return nil
	}

	messages := []string{}
	for _, msg := range r.Messages {
		for _, err := range r.Errors {
			if err == msg.Err {
				messages = append(messages, fmt.Sprintf("%s: %s", msg.Path, msg.Err))
				break
			}
		}
	}

	return fmt.Errorf("malformed chart or values: \n\t%s", strings.Join(messages, "\n\t"))
}

func (r *HelmRelease) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	namespace, name, err := parseImportIdentifier(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse import identifier",
			fmt.Sprintf("Unable to parse identifier %s: %s", req.ID, err),
		)
		return
	}

	meta := r.meta
	if meta == nil {
		resp.Diagnostics.AddError(
			"Meta not set",
			"The meta information is not set for the resource",
		)
		return
	}

	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting helm configuration",
			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
		)
		return
	}

	release, err := getRelease(ctx, meta, actionConfig, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting release",
			fmt.Sprintf("Unable to get Helm release %s: %s", name, err.Error()),
		)
		return
	}

	var state HelmReleaseModel
	state.Name = types.StringValue(release.Name)
	state.Description = types.StringValue(release.Info.Description)
	state.Chart = types.StringValue(release.Chart.Metadata.Name)

	// Set default attributes
	for key, value := range defaultAttributes {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(key), value)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Set release-specific attributes using the helper function
	diags := setReleaseAttributes(ctx, &state, release, meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Setting final state: %+v", state))
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		tflog.Error(ctx, "Error setting final state", map[string]interface{}{
			"state":       state,
			"diagnostics": diags,
		})
		return
	}
}

func parseImportIdentifier(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := errors.Errorf("Unexpected ID format (%q), expected namespace/name", id)
		return "", "", err
	}

	return parts[0], parts[1], nil
}

// returns true if any values, set_list, set, set_sensitive are unknown
func valuesUnknown(plan HelmReleaseModel) bool {
	if plan.Values.IsUnknown() {
		return true
	}
	if plan.SetList.IsUnknown() {
		return true
	}
	if plan.Set.IsUnknown() {
		return true
	}
	if plan.SetSensitive.IsUnknown() {
		return true
	}
	return false
}
