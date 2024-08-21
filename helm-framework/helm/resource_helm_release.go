package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	_ resource.Resource                 = &HelmReleaseResource{}
	_ resource.ResourceWithUpgradeState = &HelmReleaseResource{}
	_ resource.ResourceWithModifyPlan   = &HelmReleaseResource{}
	_ resource.ResourceWithImportState  = &HelmReleaseResource{}
)

type HelmReleaseResource struct {
	meta *Meta
}

func NewHelmReleaseResource() resource.Resource {
	return &HelmReleaseResource{}
}

type HelmReleaseModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Repository                 types.String `tfsdk:"repository"`
	Repository_Key_File        types.String `tfsdk:"repository_key_file"`
	Repository_Cert_File       types.String `tfsdk:"repository_cert_file"`
	Repository_Ca_File         types.String `tfsdk:"repository_ca_file"`
	Repository_Username        types.String `tfsdk:"repository_username"`
	Repository_Password        types.String `tfsdk:"repository_password"`
	Pass_Credentials           types.Bool   `tfsdk:"pass_credentials"`
	Chart                      types.String `tfsdk:"chart"`
	Version                    types.String `tfsdk:"version"`
	Devel                      types.Bool   `tfsdk:"devel"`
	Values                     types.List   `tfsdk:"values"`
	Set                        types.Set    `tfsdk:"set"`
	Set_list                   types.List   `tfsdk:"set_list"`
	Set_Sensitive              types.Set    `tfsdk:"set_sensitive"`
	Namespace                  types.String `tfsdk:"namespace"`
	Verify                     types.Bool   `tfsdk:"verify"`
	Keyring                    types.String `tfsdk:"keyring"`
	Timeout                    types.Int64  `tfsdk:"timeout"`
	Disable_Webhooks           types.Bool   `tfsdk:"disable_webhooks"`
	Disable_Crd_Hooks          types.Bool   `tfsdk:"disable_crd_hooks"`
	Reset_Values               types.Bool   `tfsdk:"reset_values"`
	Reuse_Values               types.Bool   `tfsdk:"reuse_values"`
	Force_Update               types.Bool   `tfsdk:"force_update"`
	Recreate_Pods              types.Bool   `tfsdk:"recreate_pods"`
	Cleanup_On_Fail            types.Bool   `tfsdk:"cleanup_on_fail"`
	Max_History                types.Int64  `tfsdk:"max_history"`
	Atomic                     types.Bool   `tfsdk:"atomic"`
	Skip_Crds                  types.Bool   `tfsdk:"skip_crds"`
	Render_Subchart_Notes      types.Bool   `tfsdk:"render_subchart_notes"`
	Disable_Openapi_Validation types.Bool   `tfsdk:"disable_openapi_validation"`
	Wait                       types.Bool   `tfsdk:"wait"`
	Wait_For_Jobs              types.Bool   `tfsdk:"wait_for_jobs"`
	Status                     types.String `tfsdk:"status"`
	Dependency_Update          types.Bool   `tfsdk:"dependency_update"`
	Replace                    types.Bool   `tfsdk:"replace"`
	Description                types.String `tfsdk:"description"`
	Create_Namespace           types.Bool   `tfsdk:"create_namespace"`
	Postrender                 types.List   `tfsdk:"postrender"`
	Lint                       types.Bool   `tfsdk:"lint"`
	Manifest                   types.String `tfsdk:"manifest"`
	Metadata                   types.List   `tfsdk:"metadata"`
}

var defaultAttributes = map[string]interface{}{
	"verify":                     false,
	"timeout":                    300,
	"wait":                       true,
	"wait_for_jobs":              false,
	"disable_webhooks":           false,
	"atomic":                     false,
	"render_subchart_notes":      true,
	"disable_openapi_validation": false,
	"disable_crd_hooks":          false,
	"force_update":               false,
	"reset_values":               false,
	"reuse_values":               false,
	"recreate_pods":              false,
	"max_history":                0,
	"skip_crds":                  false,
	"cleanup_on_fail":            false,
	"dependency_update":          false,
	"replace":                    false,
	"create_namespace":           false,
	"lint":                       false,
	"pass_credentials":           false,
}

type releaseMetaData struct {
	Name        types.String `tfsdk:"name"`
	Revision    types.Int64  `tfsdk:"revision"`
	Namespace   types.String `tfsdk:"namespace"`
	Chart       types.String `tfsdk:"chart"`
	Version     types.String `tfsdk:"version"`
	App_Version types.String `tfsdk:"app_version"`
	Values      types.String `tfsdk:"values"`
}
type setResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type set_listResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.List   `tfsdk:"value"`
}

type set_sensitiveResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type postrenderModel struct {
	Binary_Path types.String `tfsdk:"binary_path"`
	Args        types.List   `tfsdk:"args"`
}

// Supress describption
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

func (r *HelmReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

func (r *HelmReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are available in the resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
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
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If it is a URL, the chart is installed without installing the repository",
			},
			"repository_key_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert key file",
			},
			"repository_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repositories cert file",
			},
			"repository_ca_file": schema.StringAttribute{
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
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used",
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed",
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If 'version' is set, this is ignored",
				PlanModifiers: []planmodifier.Bool{
					suppressDevel(),
				},
			},
			"values": schema.ListAttribute{
				Optional:    true,
				Description: "List of values in raw YAML format to pass to helm",
				ElementType: types.StringType,
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
			"verify": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Verify the package before installing it.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Description: "Location of public keys used for verification, Used only if 'verify is true'",
				PlanModifiers: []planmodifier.String{
					suppressKeyring(),
				},
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(300),
				Description: "Time in seconds to wait for any individual kubernetes operation",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Prevent hooks from running",
			},
			"disable_crd_hooks": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Prevent CRD hooks from running, but run other hooks. See helm install --no-crd-hook",
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
				Default:     booldefault.StaticBool(false),
			},
			"reset_values": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When upgrading, reset the values to the ones built into the chart",
				Default:     booldefault.StaticBool(false),
			},
			"force_update": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Force resource update through delete/recreate if needed.",
			},
			"recreate_pods": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Perform pods restart during upgrade/rollback",
			},
			"cleanup_on_fail": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow deletion of new resources created in this upgrade when upgrade fails",
			},
			"max_history": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Limit the maximum number of revisions saved per release. Use 0 for no limit",
			},
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
			},
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "If set, render subchart notes along with the parent",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"wait_for_jobs": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the release",
			},
			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Run helm dependency update before installing the chart",
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Add a custom description",
				PlanModifiers: []planmodifier.String{
					suppressDescription(),
				},
			},
			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Create the namespace if it does not exist",
			},
			"lint": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Run helm lint when planning",
			},
			"manifest": schema.StringAttribute{
				Description: "The rendered manifest as JSON.",
				Computed:    true,
			},
			"metadata": schema.ListNestedAttribute{
				Description: "Status of the deployed release.",
				Computed:    true,
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
							Description: "Set of extra values. added to the chart. The sensitive data is cloaked. JSON encoded.",
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"set": schema.SetNestedBlock{
				Description: "Custom values to be merged with the values",
				NestedObject: schema.NestedBlockObject{
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
			"set_list": schema.ListNestedBlock{
				Description: "Custom sensitive values to be merged with the values",
				NestedObject: schema.NestedBlockObject{
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
			"set_sensitive": schema.SetNestedBlock{
				Description: "Custom sensitive values to be merged with the values",
				NestedObject: schema.NestedBlockObject{
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
			// single nested
			"postrender": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				Description: "Postrender command config",
				NestedObject: schema.NestedBlockObject{
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
		},
		// Indicating schema has undergone changes
		Version: 1,
	}
}

func (r *HelmReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// maps version 0 state to the upgrade function.
// If terraform detects data with version 0, we call upgrade to upgrade the state to the current schema version "1"
func (r *HelmReleaseResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: stateUpgradeV0toV1,
		},
	}
}

func stateUpgradeV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var priorState map[string]interface{}
	diags := req.State.Get(ctx, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if priorState["pass_credentials"] == nil {
		priorState["pass_credentials"] = false
	}
	if priorState["wait_for_jobs"] == nil {
		priorState["wait_for_jobs"] = false
	}

	diags = resp.State.Set(ctx, priorState)
	resp.Diagnostics.Append(diags...)
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

func (r *HelmReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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
	} else {
	}
	namespace := state.Namespace.ValueString()
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError("Error getting helm configuration", fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err))
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("clientSearch%#v", meta.RegistryClient))
	ociDiags := OCIRegistryLogin(ctx, actionConfig, meta.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.Repository_Username.ValueString(), state.Repository_Password.ValueString())
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
		panic("Error might lie here")
		return
	}

	err = isChartInstallable(c)
	if err != nil {
		resp.Diagnostics.AddError("Error checking if chart is installable", fmt.Sprintf("Chart is not installable: %s", err))
		return
	}

	client.ClientOnly = false
	client.DryRun = false
	client.DisableHooks = state.Disable_Webhooks.ValueBool()
	client.Wait = state.Wait.ValueBool()
	client.WaitForJobs = state.Wait_For_Jobs.ValueBool()
	client.Devel = state.Devel.ValueBool()
	client.DependencyUpdate = state.Dependency_Update.ValueBool()
	client.Timeout = time.Duration(state.Timeout.ValueInt64()) * time.Second
	client.Namespace = state.Namespace.ValueString()
	client.ReleaseName = state.Name.ValueString()
	client.Atomic = state.Atomic.ValueBool()
	client.SkipCRDs = state.Skip_Crds.ValueBool()
	client.SubNotes = state.Render_Subchart_Notes.ValueBool()
	client.DisableOpenAPIValidation = state.Disable_Openapi_Validation.ValueBool()
	client.Replace = state.Replace.ValueBool()
	client.Description = state.Description.ValueString()
	client.CreateNamespace = state.Create_Namespace.ValueBool()

	if !state.Postrender.IsNull() {
		tflog.Debug(ctx, "Postrender is not null")
		// Extract the list of postrender configurations
		var postrenderList []postrenderModel
		postrenderDiags := state.Postrender.ElementsAs(ctx, &postrenderList, false)
		resp.Diagnostics.Append(postrenderDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("Postrender list extracted: %+v", postrenderList))
		// Since postrender is defined as a list but can only have one element, we fetch the first item
		if len(postrenderList) > 0 {

			prModel := postrenderList[0]

			binaryPath := prModel.Binary_Path.ValueString()
			argsList := prModel.Args.Elements()

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
	}

	rel, err := client.Run(c, values)
	if err != nil && rel == nil {
		resp.Diagnostics.AddError("installation failed", err.Error())
		return
	}

	if err != nil && rel != nil {
		fmt.Printf("Namespace value before calling resourceReleaseExists: %s\n", state.Namespace.ValueString())

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

	tflog.Debug(ctx, fmt.Sprintf("Actual state after Create: %+v", state))
}

func (r *HelmReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	tflog.Debug(ctx, fmt.Sprintf("%s Done", logID))
	// Save data into terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HelmReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Desired state of the resource after update operation is applied
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
	tflog.Debug(ctx, fmt.Sprintf("Plan state on Update: %+v", plan))
	tflog.Debug(ctx, fmt.Sprintf("Actual state before Update: %+v", state))

	logID := fmt.Sprintf("[resourceReleaseUpdate: %s]", state.Name.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("%s Started", logID))

	meta := r.meta
	namespace := state.Namespace.ValueString()
	tflog.Debug(ctx, fmt.Sprintf("%s Getting helm configuration for namespace: %s", logID, namespace))
	tflog.Debug(ctx, fmt.Sprintf("%s Getting helm configuration", logID))
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("%s Failed to get helm configuration: %v", logID, err))
		resp.Diagnostics.AddError("Error getting helm configuration", fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err))
		return
	}
	ociDiags := OCIRegistryLogin(ctx, actionConfig, meta.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.Repository_Username.ValueString(), state.Repository_Password.ValueString())
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
	client.WaitForJobs = plan.Wait_For_Jobs.ValueBool()
	client.DryRun = false
	client.DisableHooks = plan.Disable_Webhooks.ValueBool()
	client.Atomic = plan.Atomic.ValueBool()
	client.SkipCRDs = plan.Skip_Crds.ValueBool()
	client.SubNotes = plan.Render_Subchart_Notes.ValueBool()
	client.DisableOpenAPIValidation = plan.Disable_Openapi_Validation.ValueBool()
	client.Force = plan.Force_Update.ValueBool()
	client.ResetValues = plan.Reset_Values.ValueBool()
	client.ReuseValues = plan.Reuse_Values.ValueBool()
	client.Recreate = plan.Recreate_Pods.ValueBool()
	client.MaxHistory = int(plan.Max_History.ValueInt64())
	client.CleanupOnFail = plan.Cleanup_On_Fail.ValueBool()
	client.Description = plan.Description.ValueString()

	if !plan.Postrender.IsNull() {
		// Extract the list of postrender configurations
		var postrenderList []postrenderModel
		postrenderDiags := plan.Postrender.ElementsAs(ctx, &postrenderList, false)
		resp.Diagnostics.Append(postrenderDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("Initial postrender values update method: %+v", postrenderList))

		// Since postrender is defined as a list but can only have one element, we fetch the first item
		if len(postrenderList) > 0 {
			prModel := postrenderList[0]

			binaryPath := prModel.Binary_Path.ValueString()
			argsList := prModel.Args.Elements()

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

// c
func (r *HelmReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Initialize state
	var state HelmReleaseModel
	diags := req.State.Get(ctx, &state)

	for _, diag := range diags {
		log.Printf("[DEBUG] Diagnostics after state get: %s", diag.Detail())
	}

	// Append diagnostics to response
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		log.Printf("[ERROR] Error retrieving state: %v", resp.Diagnostics)
		return
	}
	log.Printf("[DEBUG] Retrieved state: %+v", state)

	// Check if meta is set
	meta := r.meta
	if meta == nil {
		resp.Diagnostics.AddError(
			"Meta not set",
			"The meta information is not set for the resource",
		)
		log.Printf("[ERROR] Meta information is not set for the resource")
		return
	}
	log.Printf("[DEBUG] Meta information is set")

	exists, diags := resourceReleaseExists(ctx, state.Name.ValueString(), state.Namespace.ValueString(), meta)
	if !exists {
		return
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Get namespace
	namespace := state.Namespace.ValueString()
	log.Printf("[DEBUG] Namespace: %s", namespace)

	// Get Helm configuration
	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting helm configuration",
			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
		)
		log.Printf("[ERROR] Unable to get Helm configuration for namespace %s: %s", namespace, err)
		return
	}
	log.Printf("[DEBUG] Retrieved Helm configuration for namespace: %s", namespace)

	// Get release name
	name := state.Name.ValueString()
	log.Printf("[DEBUG] Release name: %s", name)

	// Initialize uninstall action
	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = state.Wait.ValueBool()
	uninstall.DisableHooks = state.Disable_Webhooks.ValueBool()
	uninstall.Timeout = time.Duration(state.Timeout.ValueInt64()) * time.Second
	log.Printf("[DEBUG] Uninstall configuration: Wait=%t, DisableHooks=%t, Timeout=%d", uninstall.Wait, uninstall.DisableHooks, uninstall.Timeout)

	// Uninstall the release
	log.Printf("[INFO] Uninstalling Helm release: %s", name)
	res, err := uninstall.Run(name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error uninstalling release",
			fmt.Sprintf("Unable to uninstall Helm release %s: %s", name, err),
		)
		log.Printf("[ERROR] Unable to uninstall Helm release %s: %s", name, err)
		return
	}

	if res.Info != "" {
		resp.Diagnostics.Append(diag.NewWarningDiagnostic(
			"Helm uninstall returned an information message",
			res.Info,
		))
		log.Printf("[WARN] Helm uninstall returned an information message: %s", res.Info)
	}

	// Remove resource from state
	// resp.State.RemoveResource(ctx)

}

func chartPathOptions(d *HelmReleaseModel, meta *Meta, cpo *action.ChartPathOptions) (*action.ChartPathOptions, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	chartName := d.Chart.ValueString()
	repository := d.Repository.ValueString()

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
		repositoryURL, chartName, err = resolveChartName(repository, strings.TrimSpace(chartName))
		if err != nil {
			diags.AddError("Error Resolving Chart Name", fmt.Sprintf("Could not resolve chart name for repository %s and chart %s: %s", repository, chartName, err))
			return nil, "", diags
		}
	}

	version := getVersion(d, meta)

	cpo.CaFile = d.Repository_Ca_File.ValueString()
	cpo.CertFile = d.Repository_Cert_File.ValueString()
	cpo.KeyFile = d.Repository_Key_File.ValueString()
	cpo.Keyring = d.Keyring.ValueString()
	cpo.RepoURL = repositoryURL
	cpo.Verify = d.Verify.ValueBool()
	if !useChartVersion(chartName, cpo.RepoURL) {
		cpo.Version = version
	}
	cpo.Username = d.Repository_Username.ValueString()
	cpo.Password = d.Repository_Password.ValueString()
	cpo.PassCredentialsAll = d.Pass_Credentials.ValueBool()

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

func resolveChartName(repository, name string) (string, string, error) {
	_, err := url.ParseRequestURI(repository)
	if err == nil {
		return repository, name, nil
	}

	if strings.Index(name, "/") == -1 && repository != "" {
		name = fmt.Sprintf("%s/%s", repository, name)
	}

	return "", name, nil
}

func getVersion(d *HelmReleaseModel, meta *Meta) string {
	version := d.Version.ValueString()

	if version == "" && d.Devel.ValueBool() {
		tflog.Debug(context.Background(), "setting version to >0.0.0-0")
		version = ">0.0.0-0"
	} else {
		version = strings.TrimSpace(version)
	}

	return version
}

// c
func isChartInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func getChart(ctx context.Context, d *HelmReleaseModel, m *Meta, name string, cpo *action.ChartPathOptions) (*chart.Chart, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	m.Lock()
	defer m.Unlock()
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

func convertSetSensitiveToSetResourceModel(ctx context.Context, sensitive set_sensitiveResourceModel) setResourceModel {
	tflog.Debug(ctx, fmt.Sprintf("Converting set_sensitiveResourceModel: %+v", sensitive))

	converted := setResourceModel{
		Name:  sensitive.Name,
		Value: sensitive.Value,
		Type:  sensitive.Type,
	}

	tflog.Debug(ctx, fmt.Sprintf("Converted to setResourceModel: %+v", converted))
	return converted
}

func getValues(ctx context.Context, d *HelmReleaseModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

	// Processing "values" attribute
	for _, raw := range d.Values.Elements() {
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
	if !d.Set.IsNull() {
		tflog.Debug(ctx, "Processing Set attribute")
		var setList []setResourceModel
		setDiags := d.Set.ElementsAs(ctx, &setList, false)
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
	if !d.Set_list.IsUnknown() {
		tflog.Debug(ctx, "Processing Set_list attribute")
		var setListList []set_listResourceModel
		setListDiags := d.Set_list.ElementsAs(ctx, &setListList, false)
		diags.Append(setListDiags...)
		if diags.HasError() {
			tflog.Debug(ctx, "Error occurred while processing Set_list attribute")
			return nil, diags
		}

		for i, setList := range setListList {
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
	if !d.Set_Sensitive.IsNull() {
		tflog.Debug(ctx, "Processing Set_Sensitive attribute")
		var setSensitiveList []set_sensitiveResourceModel
		setSensitiveDiags := d.Set_Sensitive.ElementsAs(ctx, &setSensitiveList, false)
		diags.Append(setSensitiveDiags...)
		if diags.HasError() {
			tflog.Debug(ctx, "Error occurred while processing Set_Sensitive attribute")
			return nil, diags
		}

		for i, setSensitive := range setSensitiveList {
			tflog.Debug(ctx, fmt.Sprintf("Processing Set_Sensitive element at index %d: %v", i, setSensitive))
			setModel := convertSetSensitiveToSetResourceModel(ctx, setSensitive)
			setSensitiveDiags := getValue(base, setModel)
			diags.Append(setSensitiveDiags...)
			if diags.HasError() {
				tflog.Debug(ctx, fmt.Sprintf("Error occurred while processing Set_Sensitive element at index %d", i))
				return nil, diags
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Final merged values: %v", base))
	logDiags := logValues(ctx, base, d)
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
	if !state.Set_Sensitive.IsNull() {
		var setSensitiveList []set_sensitiveResourceModel
		diags := state.Set_Sensitive.ElementsAs(context.Background(), &setSensitiveList, false)
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
	metadata := []map[string]attr.Value{
		{
			"name":        types.StringValue(r.Name),
			"revision":    types.Int64Value(int64(r.Version)),
			"namespace":   types.StringValue(r.Namespace),
			"chart":       types.StringValue(r.Chart.Metadata.Name),
			"version":     types.StringValue(r.Chart.Metadata.Version),
			"app_version": types.StringValue(r.Chart.Metadata.AppVersion),
			"values":      types.StringValue(values),
		},
	}

	// Define the object type for metadata
	metadataObjectType := types.ObjectType{
		AttrTypes: metadataAttrTypes(),
	}

	// Convert each metadata map to an ObjectValue
	var metadataObjects []attr.Value
	for _, item := range metadata {
		objVal, err := types.ObjectValue(metadataAttrTypes(), item)
		if err != nil {
			diags.AddError("Error creating ObjectValue", fmt.Sprintf("Unable to create ObjectValue: %s", err))
			return diags
		}
		metadataObjects = append(metadataObjects, objVal)
	}

	// Convert the list of ObjectValues to a ListValue
	metadataList, diag := types.ListValue(metadataObjectType, metadataObjects)
	diags.Append(diag...)
	if diags.HasError() {
		tflog.Error(ctx, "Error converting metadata to ListValue", map[string]interface{}{
			"metadata": metadata,
			"error":    diags,
		})

		return diags
	}

	// Log metadata after conversion
	tflog.Debug(ctx, fmt.Sprintf("Metadata after conversion: %+v", metadataList))
	state.Metadata = metadataList
	return diags
}

func metadataAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"revision":    types.Int64Type,
		"namespace":   types.StringType,
		"chart":       types.StringType,
		"version":     types.StringType,
		"app_version": types.StringType,
		"values":      types.StringType,
	}
}

func extractSensitiveValues(state *HelmReleaseModel) map[string]string {
	sensitiveValues := make(map[string]string)

	if !state.Set_Sensitive.IsNull() {
		var setSensitiveList []set_sensitiveResourceModel
		diags := state.Set_Sensitive.ElementsAs(context.Background(), &setSensitiveList, false)
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
	tflog.Debug(ctx, fmt.Sprintf("%s getRelease wait for lock", name))
	m.Lock()
	defer m.Unlock()
	tflog.Debug(ctx, fmt.Sprintf("%s getRelease got lock, started", name))

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
func checkChartDependencies(ctx context.Context, d *HelmReleaseModel, c *chart.Chart, path string, m *Meta) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	p := getter.All(m.Settings)

	if req := c.Metadata.Dependencies; req != nil {
		err := action.CheckDependencies(c, req)
		if err != nil {
			if d.Dependency_Update.ValueBool() {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          d.Keyring.ValueString(),
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

func (r *HelmReleaseResource) StateUpgrade(ctx context.Context, version int, state map[string]interface{}, meta interface{}) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if state["pass_credentials"] == nil {
		state["pass_credentials"] = false
	}
	if state["wait_for_jobs"] == nil {
		state["wait_for_jobs"] = false
	}

	return state, diags
}

// We just want plan
func (r *HelmReleaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// resource is being destroyed
		return
	}
	var plan HelmReleaseModel
	var state *HelmReleaseModel
	log.Printf("Plan: %+v", state)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("Plan state on ModifyPlan: %+v", plan))
	tflog.Debug(ctx, fmt.Sprintf("Actual state on ModifyPlan: %+v", state))

	logID := fmt.Sprintf("[resourceDiff: %s]", plan.Name.ValueString())
	tflog.Debug(ctx, fmt.Sprintf("%s Start", logID))

	m := r.meta
	name := plan.Name.ValueString()
	namespace := plan.Namespace.ValueString()

	actionConfig, err := m.GetHelmConfiguration(ctx, namespace)
	if err != nil {
		resp.Diagnostics.AddError("Error getting Helm configuration", err.Error())
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Initial Values: Name=%s, Namespace=%s, Repository=%s, Repository_Username=%s, Repository_Password=%s, Chart=%s", logID,
		name, namespace, plan.Repository.ValueString(), plan.Repository_Username.ValueString(), plan.Repository_Password.ValueString(), plan.Chart.ValueString()))

	if plan.Repository.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("%s Repository is null", logID))
	}
	if plan.Repository_Username.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("%s Repository_Username is null", logID))
	}
	if plan.Repository_Password.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("%s Repository_Password is null", logID))
	}
	if plan.Chart.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("%s Chart is null", logID))
	}
	repositoryURL := plan.Repository.ValueString()
	repositoryUsername := plan.Repository_Username.ValueString()
	repositoryPassword := plan.Repository_Password.ValueString()
	chartName := plan.Chart.ValueString()
	ociDiags := OCIRegistryLogin(ctx, actionConfig, m.RegistryClient, repositoryURL, chartName, repositoryUsername, repositoryPassword)
	resp.Diagnostics.Append(ociDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always set desired state to DEPLOYED
	plan.Status = types.StringValue(release.StatusDeployed.String())

	recomputeMetadataFields := []string{
		"chart",
		"repository",
		"values",
		"set",
		"set_sensitive",
		"set_list",
	}
	hasChanges := false
	for _, field := range recomputeMetadataFields {
		// We are using Plan.GetAttribute to check if the attribute has changed
		var oldValue, newValue attr.Value
		req.Plan.GetAttribute(ctx, path.Root(field), &newValue)
		req.State.GetAttribute(ctx, path.Root(field), &oldValue)
		if !newValue.Equal(oldValue) {
			hasChanges = true
			break
		}
	}
	if hasChanges {
		tflog.Debug(ctx, fmt.Sprintf("%s Metadata has changes, setting to unknown", logID))
		plan.Metadata = types.ListUnknown(types.ObjectType{AttrTypes: metadataAttrTypes()})
	}

	if !useChartVersion(plan.Chart.ValueString(), plan.Repository.ValueString()) {
		var oldVersion, newVersion attr.Value
		req.Plan.GetAttribute(ctx, path.Root("version"), &newVersion)
		req.State.GetAttribute(ctx, path.Root("version"), &oldVersion)

		// Check if version has changed
		if !newVersion.Equal(oldVersion) {
			// Remove surrounding quotes if they exist
			oldVersionStr := strings.Trim(oldVersion.String(), "\"")
			newVersionStr := strings.Trim(newVersion.String(), "\"")

			// Ensure trimming 'v' prefix correctly
			oldVersionStr = strings.TrimPrefix(oldVersionStr, "v")
			newVersionStr = strings.TrimPrefix(newVersionStr, "v")

			if oldVersionStr != newVersionStr && newVersionStr != "" {
				// Setting Metadata to a computed value
				plan.Metadata = types.ListUnknown(types.ObjectType{AttrTypes: metadataAttrTypes()})
			}
		} else {
		}
	}

	client := action.NewInstall(actionConfig)
	cpo, chartName, diags := chartPathOptions(&plan, m, &client.ChartPathOptions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	chart, path, diags := getChart(ctx, &plan, m, chartName, cpo)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Got chart", logID))

	updated, diags := checkChartDependencies(ctx, &plan, chart, path, m)
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
		diags := resourceReleaseValidate(ctx, &plan, m, cpo)
		if diags.HasError() {
			for _, diag := range diags {
				resp.Diagnostics.Append(diag)
			}
			return
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("%s Release validated", logID))

	if m.ExperimentEnabled("manifest") {
		// Check if all necessary values are known
		known, diags := valuesKnown(ctx, req)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !known {
			tflog.Debug(ctx, "not all values are known, skipping dry run to render manifest")
			plan.Manifest = types.StringNull()
			plan.Version = types.StringNull()
			return
		}

		var postRenderer postrender.PostRenderer
		if !plan.Postrender.IsNull() {
			// Extract the list of postrender configurations
			var postrenderList []postrenderModel
			postrenderDiags := plan.Postrender.ElementsAs(ctx, &postrenderList, false)
			resp.Diagnostics.Append(postrenderDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
			if len(postrenderList) > 0 {
				prModel := postrenderList[0]

				binaryPath := prModel.Binary_Path.ValueString()
				argsList := prModel.Args.Elements()

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
		}
		if state == nil {
			install := action.NewInstall(actionConfig)
			install.ChartPathOptions = *cpo
			install.DryRun = true
			install.DisableHooks = plan.Disable_Webhooks.ValueBool()
			install.Wait = plan.Wait.ValueBool()
			install.WaitForJobs = plan.Wait_For_Jobs.ValueBool()
			install.Devel = plan.Devel.ValueBool()
			install.DependencyUpdate = plan.Dependency_Update.ValueBool()
			install.Timeout = time.Duration(plan.Timeout.ValueInt64()) * time.Second
			install.Namespace = plan.Namespace.ValueString()
			install.ReleaseName = plan.Name.ValueString()
			install.Atomic = plan.Atomic.ValueBool()
			install.SkipCRDs = plan.Skip_Crds.ValueBool()
			install.SubNotes = plan.Render_Subchart_Notes.ValueBool()
			install.DisableOpenAPIValidation = plan.Disable_Openapi_Validation.ValueBool()
			install.Replace = plan.Replace.ValueBool()
			install.Description = plan.Description.ValueString()
			install.CreateNamespace = plan.Create_Namespace.ValueBool()
			install.PostRenderer = postRenderer

			values, diags := getValues(ctx, &plan)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			tflog.Debug(ctx, fmt.Sprintf("%s performing dry run install", logID))
			dry, err := install.Run(chart, values)
			if err != nil {
				if strings.Contains(err.Error(), "Kubernetes cluster unreachable") {
					tflog.Debug(ctx, "cluster was unreachable at create time, marking manifest as computed")
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
			if !plan.Set_Sensitive.IsNull() {
				var setSensitiveList []set_sensitiveResourceModel
				setSensitiveDiags := plan.Set_Sensitive.ElementsAs(ctx, &setSensitiveList, false)
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

		_, err = getRelease(ctx, m, actionConfig, name)
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
		upgrade.DisableHooks = plan.Disable_Webhooks.ValueBool()
		upgrade.Atomic = plan.Atomic.ValueBool()
		upgrade.SubNotes = plan.Render_Subchart_Notes.ValueBool()
		upgrade.WaitForJobs = plan.Wait_For_Jobs.ValueBool()
		upgrade.Force = plan.Force_Update.ValueBool()
		upgrade.ResetValues = plan.Reset_Values.ValueBool()
		upgrade.ReuseValues = plan.Reuse_Values.ValueBool()
		upgrade.Recreate = plan.Recreate_Pods.ValueBool()
		upgrade.MaxHistory = int(plan.Max_History.ValueInt64())
		upgrade.CleanupOnFail = plan.Cleanup_On_Fail.ValueBool()
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
		if !plan.Set_Sensitive.IsNull() {
			var setSensitiveList []set_sensitiveResourceModel
			setSensitiveDiags := plan.Set_Sensitive.ElementsAs(ctx, &setSensitiveList, false)
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
		plan.Version = types.StringNull()
	}
	resp.Plan.Set(ctx, &plan)
}

func resourceReleaseValidate(ctx context.Context, d *HelmReleaseModel, meta *Meta, cpo *action.ChartPathOptions) diag.Diagnostics {
	var diags diag.Diagnostics

	cpo, name, chartDiags := chartPathOptions(d, meta, cpo)
	diags.Append(chartDiags...)
	if diags.HasError() {
		diags.AddError("Malformed values", fmt.Sprintf("Chart path options error: %s", chartDiags))
		return diags
	}

	values, valuesDiags := getValues(ctx, d)
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

func (r *HelmReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	namespace, name, err := parseImportIdentifier(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse import identifier",
			fmt.Sprintf("Unable to parse identifier %s: %s", req.ID, err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), namespace)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)

	if resp.Diagnostics.HasError() {
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

	// Set additional attributes (name, description, chart)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), release.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), release.Info.Description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("chart"), release.Chart.Metadata.Name)...)

	if resp.Diagnostics.HasError() {
		return
	}

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

func valuesKnown(ctx context.Context, req resource.ModifyPlanRequest) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	// List of attributes to check
	checkAttributes := []path.Path{
		path.Root("values"),
		path.Root("set"),
		path.Root("set_sensitive"),
		path.Root("set_list"),
	}

	for _, attrPath := range checkAttributes {
		var attr attr.Value
		// Get the attribute value from the plan
		diags = req.Plan.GetAttribute(ctx, attrPath, &attr)
		if diags.HasError() {
			return false, diags
		}

		// Check if the attribute is known and not null
		if !attr.IsUnknown() || attr.IsNull() {
			return false, nil
		}
	}

	return true, nil
}

func getDefaultAttributes() map[string]interface{} {
	return map[string]interface{}{
		"verify":                     false,
		"timeout":                    int64(300),
		"wait":                       true,
		"wait_for_jobs":              false,
		"disable_webhooks":           false,
		"atomic":                     false,
		"render_subchart_notes":      true,
		"disable_openapi_validation": false,
		"disable_crd_hooks":          false,
		"force_update":               false,
		"reset_values":               false,
		"reuse_values":               false,
		"recreate_pods":              false,
		"max_history":                int64(0),
		"skip_crds":                  false,
		"cleanup_on_fail":            false,
		"dependency_update":          false,
		"replace":                    false,
		"create_namespace":           false,
		"lint":                       false,
		"pass_credentials":           false,
	}
}
