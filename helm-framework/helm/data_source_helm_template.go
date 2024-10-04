package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/helm/pkg/strvals"
	"sigs.k8s.io/yaml"
)

var _ datasource.DataSource = &HelmTemplate{}
var _ datasource.DataSourceWithConfigure = &HelmTemplate{}

func NewHelmTemplate() datasource.DataSource {
	return &HelmTemplate{}
}

// HelmTemplate represents the data source for rendering Helm chart templates
type HelmTemplate struct {
	meta *Meta
}

// HelmTemplateModel holds the attributes for configuring the Helm chart templates
type HelmTemplateModel struct {
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Repository               types.String `tfsdk:"repository"`
	RepositoryKeyFile        types.String `tfsdk:"repository_key_file"`
	RepositoryCertFile       types.String `tfsdk:"repository_cert_file"`
	RepositoryCaFile         types.String `tfsdk:"repository_ca_file"`
	RepositoryUsername       types.String `tfsdk:"repository_username"`
	RepositoryPassword       types.String `tfsdk:"repository_password"`
	PassCredentials          types.Bool   `tfsdk:"pass_credentials"`
	Chart                    types.String `tfsdk:"chart"`
	Version                  types.String `tfsdk:"version"`
	Devel                    types.Bool   `tfsdk:"devel"`
	Values                   types.List   `tfsdk:"values"`
	Set                      types.Set    `tfsdk:"set"`
	SetList                  types.List   `tfsdk:"set_list"`
	SetSensitive             types.Set    `tfsdk:"set_sensitive"`
	SetString                types.Set    `tfsdk:"set_string"`
	Namespace                types.String `tfsdk:"namespace"`
	Verify                   types.Bool   `tfsdk:"verify"`
	Keyring                  types.String `tfsdk:"keyring"`
	Timeout                  types.Int64  `tfsdk:"timeout"`
	DisableWebhooks          types.Bool   `tfsdk:"disable_webhooks"`
	ReuseValues              types.Bool   `tfsdk:"reuse_values"`
	ResetValues              types.Bool   `tfsdk:"reset_values"`
	Atomic                   types.Bool   `tfsdk:"atomic"`
	SkipCrds                 types.Bool   `tfsdk:"skip_crds"`
	SkipTests                types.Bool   `tfsdk:"skip_tests"`
	RenderSubchartNotes      types.Bool   `tfsdk:"render_subchart_notes"`
	DisableOpenAPIValidation types.Bool   `tfsdk:"disable_openapi_validation"`
	Wait                     types.Bool   `tfsdk:"wait"`
	DependencyUpdate         types.Bool   `tfsdk:"dependency_update"`
	Replace                  types.Bool   `tfsdk:"replace"`
	Description              types.String `tfsdk:"description"`
	CreateNamespace          types.Bool   `tfsdk:"create_namespace"`
	Postrender               types.List   `tfsdk:"postrender"`
	ApiVersions              types.List   `tfsdk:"api_versions"`
	IncludeCrds              types.Bool   `tfsdk:"include_crds"`
	IsUpgrade                types.Bool   `tfsdk:"is_upgrade"`
	ShowOnly                 types.List   `tfsdk:"show_only"`
	Validate                 types.Bool   `tfsdk:"validate"`
	Manifests                types.Map    `tfsdk:"manifests"`
	CRDs                     types.List   `tfsdk:"crds"`
	Manifest                 types.String `tfsdk:"manifest"`
	Notes                    types.String `tfsdk:"notes"`
	KubeVersion              types.String `tfsdk:"kube_version"`
}

// SetValue represents the custom value to be merged with the Helm chart values
type SetValue struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

// SetListValue represents a custom list value to be merged with the Helm chart values.
// This type is used to specify lists of values that should be passed to the Helm chart during deployment.
type SetListValue struct {
	Name  types.String `tfsdk:"name"`
	Value types.List   `tfsdk:"value"`
}

// SetSensitiveValue represents a custom sensitive value to be merged with the Helm chart values.
type SetSensitiveValue struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type SetStringValue struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type Postrender struct {
	BinaryPath types.String `tfsdk:"binary_path"`
}

func (d *HelmTemplate) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.meta = req.ProviderData.(*Meta)
	}
}

func (d *HelmTemplate) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (d *HelmTemplate) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source to render Helm chart templates.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Release name",
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If it is a URL the chart is installed without installing the repository.",
			},
			"repository_key_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's cert key file",
			},
			"repository_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's cert file",
			},
			"repository_ca_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's CA file",
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
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used.",
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed.",
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored.",
			},
			"values": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of values in raw yaml format to pass to helm.",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "Namespace to install the release into.",
			},
			"verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Verify the package before installing it.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Description: "Location of public keys used for verification. Used only if `verify` is true.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "Time in seconds to wait for any individual Kubernetes operation.",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Description: "Prevent hooks from running.",
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored.",
			},
			"reset_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reset the values to the ones built into the chart.",
			},
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, the installation process purges the chart on fail. The 'wait' flag will be set automatically if 'atomic' is used.",
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present.",
			},
			"skip_tests": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, tests will not be rendered. By default, tests are rendered.",
			},
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, render subchart notes along with the parent.",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema.",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Description: "Run helm dependency update before installing the chart.",
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Add a custom description.",
			},
			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Description: "Create the namespace if it does not exist.",
			},
			"api_versions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes api versions used for Capabilities.APIVersions.",
			},
			"include_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "Include CRDs in the templated output.",
			},
			"is_upgrade": schema.BoolAttribute{
				Optional:    true,
				Description: "Set .Release.IsUpgrade instead of .Release.IsInstall.",
			},
			"show_only": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only show manifests rendered from the given templates.",
			},
			"validate": schema.BoolAttribute{
				Optional:    true,
				Description: "Validate your manifests against the Kubernetes cluster you are currently pointing at. This is the same validation performed on an install.",
			},
			"manifests": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Map of rendered chart templates indexed by the template name.",
			},
			"crds": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of rendered CRDs from the chart.",
			},
			"manifest": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Rendered notes if the chart contains a `NOTES.txt`.",
			},
			"kube_version": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes version used for Capabilities.KubeVersion.",
			},
		},
		Blocks: map[string]schema.Block{
			// Definitions for set, set_list, set_sensitive, set_string, and postrender as previously modified
			"set": schema.SetNestedBlock{
				Description: "Custom values to be merged with the values.",
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
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string"),
							},
						},
					},
				},
			},
			"set_list": schema.ListNestedBlock{
				Description: "Custom sensitive values to be merged with the values.",
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
				Description: "Custom sensitive values to be merged with the values.",
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
			"set_string": schema.SetNestedBlock{
				Description: "Custom string values to be merged with the values.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"value": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"postrender": schema.ListNestedBlock{
				Description: "Postrender command configuration",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"binary_path": schema.StringAttribute{
							Required:    true,
							Description: "The command binary path.",
						},
					},
				},
			},
		},
	}
}

func convertStringsToTypesStrings(input []string) []types.String {
	output := make([]types.String, len(input))
	for i, v := range input {
		output[i] = types.StringValue(v)
	}
	return output
}

// Reads the current state of the data template and will update the state with the data fetched
func (d *HelmTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state HelmTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	//setting default values to false is attributes are not provided in the config
	if state.Description.IsNull() || state.Description.ValueString() == "" {
		state.Description = types.StringValue("")
	}
	if state.Devel.IsNull() || state.Devel.IsUnknown() {
		if !state.Version.IsNull() && state.Version.ValueString() != "" {
			// Version is set, suppress devel change
			state.Devel = types.BoolValue(false)
		}
	}
	if state.Keyring.IsNull() || state.Keyring.IsUnknown() {
		if !state.Verify.IsNull() && state.Verify.ValueBool() {
			state.Keyring = types.StringValue(os.ExpandEnv("$HOME/.gnupg/pubring.gpg"))
		} else {
			state.Keyring = types.StringValue("")
		}
	}
	if !state.IncludeCrds.IsNull() || state.IncludeCrds.IsUnknown() {
		state.IncludeCrds = types.BoolValue(false)
	}
	if state.IsUpgrade.IsNull() || state.IsUpgrade.IsUnknown() {
		state.IsUpgrade = types.BoolValue(false)
	}
	if state.DisableWebhooks.IsNull() || state.DisableWebhooks.IsUnknown() {
		state.DisableWebhooks = types.BoolValue(false)
	}
	if state.ReuseValues.IsNull() || state.ReuseValues.IsUnknown() {
		state.ReuseValues = types.BoolValue(false)
	}
	if state.ResetValues.IsNull() || state.ResetValues.IsUnknown() {
		state.ResetValues = types.BoolValue(false)
	}
	if state.Atomic.IsNull() || state.Atomic.IsUnknown() {
		state.Atomic = types.BoolValue(false)
	}
	if state.SkipCrds.IsNull() || state.SkipCrds.IsUnknown() {
		state.SkipCrds = types.BoolValue(false)
	}
	if state.SkipTests.IsNull() || state.SkipTests.IsUnknown() {
		state.SkipTests = types.BoolValue(false)
	}
	if state.RenderSubchartNotes.IsNull() || state.RenderSubchartNotes.IsUnknown() {
		state.RenderSubchartNotes = types.BoolValue(false)
	}
	if state.DisableOpenAPIValidation.IsNull() || state.DisableOpenAPIValidation.IsUnknown() {
		state.DisableOpenAPIValidation = types.BoolValue(false)
	}
	if state.Wait.IsNull() || state.Wait.IsUnknown() {
		state.Wait = types.BoolValue(false)
	}
	if state.DependencyUpdate.IsNull() || state.DependencyUpdate.IsUnknown() {
		state.DependencyUpdate = types.BoolValue(false)
	}
	if state.Replace.IsNull() || state.Replace.IsUnknown() {
		state.Replace = types.BoolValue(false)
	}
	if state.CreateNamespace.IsNull() || state.CreateNamespace.IsUnknown() {
		state.CreateNamespace = types.BoolValue(false)
	}
	if state.Validate.IsNull() || state.Validate.IsUnknown() {
		state.Validate = types.BoolValue(false)
	}
	if state.Verify.IsNull() || state.Verify.IsUnknown() {
		state.Verify = types.BoolValue(false)
	}
	if state.Timeout.IsNull() || state.Timeout.IsUnknown() {
		state.Timeout = types.Int64Value(300)
	}
	if state.Namespace.IsNull() || state.Namespace.IsUnknown() {
		defaultNamespace := os.Getenv("HELM_NAMESPACE")
		if defaultNamespace == "" {
			defaultNamespace = "default"
		}
		state.Namespace = types.StringValue(defaultNamespace)
	}

	m := d.meta

	actionConfig, err := m.GetHelmConfiguration(ctx, state.Namespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Helm configuration",
			fmt.Sprintf("Error getting Helm configuration might be: %s", err),
		)
		return
	}
	diags := OCIRegistryLogin(ctx, actionConfig, m.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.RepositoryUsername.ValueString(), state.RepositoryPassword.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	client := action.NewInstall(actionConfig)
	client.ReleaseName = state.Name.ValueString()
	client.Namespace = state.Namespace.ValueString()
	client.Timeout = time.Duration(state.Timeout.ValueInt64()) * time.Second
	client.Wait = state.Wait.ValueBool()
	client.DependencyUpdate = state.DependencyUpdate.ValueBool()
	client.DisableHooks = state.DisableWebhooks.ValueBool()
	client.DisableOpenAPIValidation = state.DisableOpenAPIValidation.ValueBool()
	client.Atomic = state.Atomic.ValueBool()
	client.Replace = state.Replace.ValueBool()
	client.SkipCRDs = state.SkipCrds.ValueBool()
	client.SubNotes = state.RenderSubchartNotes.ValueBool()
	client.Devel = state.Devel.ValueBool()
	client.Description = state.Description.ValueString()
	client.CreateNamespace = state.CreateNamespace.ValueBool()
	client.ClientOnly = !state.Validate.ValueBool()

	if state.KubeVersion.ValueString() != "" {
		parsedVer, err := chartutil.ParseKubeVersion(state.KubeVersion.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing Kubernetes version",
				fmt.Sprintf("Couldn't parse Kubernetes version %q: %s", state.KubeVersion.ValueString(), err),
			)
			return
		}
		client.KubeVersion = parsedVer
	}

	chartPath, err := client.ChartPathOptions.LocateChart(state.Chart.ValueString(), cli.New())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error locating chart",
			fmt.Sprintf("Error locating chart: %s", err),
		)
		return
	}

	c, err := loader.Load(chartPath)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error loading chart",
			fmt.Sprintf("Error loading chart: %s", err),
		)
		return
	}

	values, diags := getTemplateValues(ctx, &state)
	if diags.HasError() {
		for _, diag := range diags {
			resp.Diagnostics.AddError("Error getting values", fmt.Sprintf("%s", diag))
		}
		return
	}

	rel, err := client.Run(c, values)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error running Helm install",
			fmt.Sprintf("Error running Helm install: %s", err),
		)
		return
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	if !client.DisableHooks {
		for _, m := range rel.Hooks {
			if state.SkipTests.ValueBool() && isTestHook(m) {
				continue
			}
			fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
		}
	}

	splitManifests := releaseutil.SplitManifests(manifests.String())
	manifestsKeys := make([]string, 0, len(splitManifests))
	for k := range splitManifests {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

	var manifestsToRender []string
	if !state.ShowOnly.IsNull() && state.ShowOnly.Elements() != nil {
		for _, raw := range state.ShowOnly.Elements() {
			if raw.IsNull() {
				continue
			}
			value := raw.(basetypes.StringValue).ValueString()
			fString := filepath.ToSlash(value)
			missing := true
			for manifestKey, manifestName := range splitManifests {
				manifestPathSplit := strings.Split(manifestName, "/")
				manifestPath := strings.Join(manifestPathSplit, "/")
				if matched, _ := filepath.Match(fString, manifestPath); !matched {
					continue
				}
				manifestsToRender = append(manifestsToRender, manifestKey)
				missing = false
			}
			if missing {
				resp.Diagnostics.AddError(
					"Error finding template",
					fmt.Sprintf("Could not find template %q in chart", fString),
				)
				return
			}
		}
	} else {
		manifestsToRender = manifestsKeys
	}

	sort.Strings(manifestsToRender)
	computedManifests := make(map[string]string, 0)
	computedManifest := &strings.Builder{}
	for _, manifestKey := range manifestsToRender {
		manifest := splitManifests[manifestKey]
		manifestName := splitManifests[manifestKey]
		computedManifests[manifestName] = fmt.Sprintf("%s---\n%s\n", computedManifests[manifestName], manifest)
		fmt.Fprintf(computedManifest, "---\n%s\n", manifest)
	}

	var chartCRDs []string
	for _, crd := range rel.Chart.CRDObjects() {
		chartCRDs = append(chartCRDs, string(crd.File.Data))
	}
	// Convert chartCRDs to types.List
	listElements := make([]attr.Value, len(chartCRDs))
	for i, crd := range chartCRDs {
		listElements[i] = types.StringValue(crd)
	}
	listValue, diags := types.ListValue(types.StringType, listElements)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.CRDs = listValue
	// Convert computedManifests to types.Map
	elements := make(map[string]attr.Value, len(computedManifests))
	for k, v := range computedManifests {
		elements[k] = types.StringValue(v)
	}
	mapValue, diags := types.MapValue(types.StringType, elements)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.Manifests = mapValue

	state.Manifest = types.StringValue(computedManifest.String())
	state.Notes = types.StringValue(rel.Info.Notes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getTemplateValues(ctx context.Context, model *HelmTemplateModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

	// Process "values" field
	if !model.Values.IsNull() && !model.Values.IsUnknown() {
		var values []types.String
		diags = model.Values.ElementsAs(ctx, &values, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, raw := range values {
			if raw.IsNull() {
				continue
			}

			valueStr := raw.ValueString()
			if valueStr == "" {
				continue
			}

			currentMap := map[string]interface{}{}
			if err := yaml.Unmarshal([]byte(valueStr), &currentMap); err != nil {
				diags.AddError("Failed to unmarshal values", fmt.Sprintf("---> %v %s", err, valueStr))
				return nil, diags
			}

			base = mergeMaps(base, currentMap)
		}
	}

	// Process "set" field
	if !model.Set.IsNull() && !model.Set.IsUnknown() {
		var sets []SetValue
		diags = model.Set.ElementsAs(ctx, &sets, false)
		if diags.HasError() {
			return nil, diags
		}

		for _, raw := range sets {
			set := raw
			if err := getDataValue(base, set); err.HasError() {
				diags.Append(err...)
				return nil, diags
			}
		}
	}

	// Process "set_list" field
	if !model.SetList.IsNull() && !model.SetList.IsUnknown() {
		var setListSlice []SetListValue
		diags = model.SetList.ElementsAs(ctx, &setListSlice, false)
		if diags.HasError() {
			return nil, diags
		}
		for _, setList := range setListSlice {
			setListDiags := getDataSourceListValue(ctx, base, setList)
			diags.Append(setListDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	// Process "set_sensitive" field
	if !model.SetSensitive.IsNull() && !model.SetList.IsUnknown() {
		var setSensitiveList []SetSensitiveValue
		diags = model.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
		if diags.HasError() {
			return nil, diags
		}
		for _, setSensitive := range setSensitiveList {
			setSensitiveDiags := getDataSensitiveValue(base, setSensitive)
			diags.Append(setSensitiveDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	return base, logDataValues(ctx, base, model)
}
func getDataSourceListValue(ctx context.Context, base map[string]interface{}, set SetListValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()
	listValue := set.Value

	// Check if the list is null or unknown
	if listValue.IsNull() || listValue.IsUnknown() {
		return diags
	}

	// Get the elements from the list
	elements := listValue.Elements()
	listStringArray := make([]string, len(elements))
	for i, v := range elements {
		listStringArray[i] = v.(basetypes.StringValue).ValueString()
	}

	nonEmptyListStringArray := make([]string, 0, len(listStringArray))
	for _, s := range listStringArray {
		if s != "" {
			nonEmptyListStringArray = append(nonEmptyListStringArray, s)
		}
	}
	listString := strings.Join(nonEmptyListStringArray, ",")
	if err := strvals.ParseInto(fmt.Sprintf("%s={%s}", name, listString), base); err != nil {
		diags.AddError("Error parsing list value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, listString, err))
		return diags
	}

	return diags
}

func getDataSensitiveValue(base map[string]interface{}, set SetSensitiveValue) diag.Diagnostics {
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

// For the type SetValue
func getDataValue(base map[string]interface{}, set SetValue) diag.Diagnostics {
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

func logDataValues(ctx context.Context, values map[string]interface{}, model *HelmTemplateModel) diag.Diagnostics {
	var diags diag.Diagnostics

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

	diags.Append(cloakDataSetValues(ctx, c, model)...)

	y, err := yaml.Marshal(c)
	if err != nil {
		diags.AddError("Error marshaling map to YAML", fmt.Sprintf("Failed to marshal map to YAML: %s", err))
	}

	tflog.Debug(ctx, fmt.Sprintf("---[ values.yaml ]-----------------------------------\n%s\n", string(y)))

	return diags
}

func cloakDataSetValues(ctx context.Context, config map[string]interface{}, model *HelmTemplateModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !model.SetSensitive.IsNull() && !model.SetSensitive.IsUnknown() {
		var setSensitiveList []SetSensitiveValue
		diags = model.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
		if diags.HasError() {
			return diags
		}

		for _, set := range setSensitiveList {
			cloakSetValue(config, set.Name.ValueString())
		}
	}

	return diags
}

func getDataListValue(base map[string]interface{}, set SetListValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()
	listValue := set.Value

	// Check if the list is null or unknown
	if listValue.IsNull() || listValue.IsUnknown() {
		return diags
	}

	// Get the elements from the list
	elements := listValue.Elements()
	listStringArray := make([]string, len(elements))
	for i, v := range elements {
		listStringArray[i] = v.(basetypes.StringValue).ValueString()
	}

	nonEmptyListStringArray := make([]string, 0, len(elements))
	for _, s := range listStringArray {
		if s != "" {
			nonEmptyListStringArray = append(nonEmptyListStringArray, s)
		}
	}
	listString := strings.Join(nonEmptyListStringArray, ",")
	if err := strvals.ParseInto(fmt.Sprintf("%s={%s}", name, listString), base); err != nil {
		diags.AddError("Error parsing list value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, listString, err))
		return diags
	}

	return diags
}

func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}
