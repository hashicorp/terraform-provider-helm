package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

var _ datasource.DataSource = &DataTemplate{}
var _ datasource.DataSourceWithConfigure = &DataTemplate{}

func NewDataTemplate() datasource.DataSource {
	return &DataTemplate{}
}

type DataTemplate struct {
	meta *Meta
}

type DataTemplateModel struct {
	Name                     types.String        `tfsdk:"name"`
	Repository               types.String        `tfsdk:"repository"`
	RepositoryKeyFile        types.String        `tfsdk:"repository_key_file"`
	RepositoryCertFile       types.String        `tfsdk:"repository_cert_file"`
	RepositoryCaFile         types.String        `tfsdk:"repository_ca_file"`
	RepositoryUsername       types.String        `tfsdk:"repository_username"`
	RepositoryPassword       types.String        `tfsdk:"repository_password"`
	PassCredentials          types.Bool          `tfsdk:"pass_credentials"`
	Chart                    types.String        `tfsdk:"chart"`
	Version                  types.String        `tfsdk:"version"`
	Devel                    types.Bool          `tfsdk:"devel"`
	Values                   []types.String      `tfsdk:"values"`
	Set                      []SetValue          `tfsdk:"set"`
	SetList                  []SetListValue      `tfsdk:"set_list"`
	SetSensitive             []SetSensitiveValue `tfsdk:"set_sensitive"`
	SetString                []SetStringValue    `tfsdk:"set_string"`
	Namespace                types.String        `tfsdk:"namespace"`
	Verify                   types.Bool          `tfsdk:"verify"`
	Keyring                  types.String        `tfsdk:"keyring"`
	Timeout                  types.Int64         `tfsdk:"timeout"`
	DisableWebhooks          types.Bool          `tfsdk:"disable_webhooks"`
	ReuseValues              types.Bool          `tfsdk:"reise_values"`
	ResetValues              types.Bool          `tfsdk:"reset_values"`
	Atomic                   types.Bool          `tfsdk:"atomic"`
	SkipCrds                 types.Bool          `tfsdk:"skip_crds"`
	SkipTests                types.Bool          `tfsdk:"skip_tests"`
	RenderSubchartNotes      types.Bool          `tfsdk:"render_subchart_notes"`
	DisableOpenAPIValidation types.Bool          `tfsdk:"disable_openapi_validation"`
	Wait                     types.Bool          `tfsdk:"wait"`
	DependencyUpdate         types.Bool          `tfsdk:"dependency_update"`
	Replace                  types.Bool          `tfsdk:"replace"`
	Description              types.String        `tfsdk:"description"`
	CreateNamespace          types.Bool          `tfsdk:"create_namespace"`
	Postrender               []Postrender        `tfsdk:"postrender"`
	ApiVersions              []types.String      `tfsdk:"api_versions"`
	IncludeCrds              types.Bool          `tfsdk:"include_crds"`
	IsUpgrade                types.Bool          `tfsdk:"is_upgrade"`
	ShowOnly                 []types.String      `tfsdk:"show_only"`
	Validate                 types.Bool          `tfsdk:"validate"`
	Manifests                map[string]string   `tfsdk:"manifests"`
	CRDs                     []types.String      `tfsdk:"crds"`
	Manifest                 types.String        `tfsdk:"manifest"`
	Notes                    types.String        `tfsdk:"notes"`
	KubeVersion              types.String        `tfsdk:"kube_version"`
}

type SetValue struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	Type  types.String `tfsdk:"type"`
}

type SetListValue struct {
	Name  types.String   `tfsdk:"name"`
	Value []types.String `tfsdk:"value"`
}

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

func (d *DataTemplate) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		//Casting the provider data to the HelmProvider type assigning it to the provider field in data template struct
		d.meta = req.ProviderData.(*Meta)
	}
}

func (d *DataTemplate) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
}

func (d *DataTemplate) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source to render Helm chart templates.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Release name",
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If is a URL the chart is installed without installing the repository.",
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
				Optional:  true,
				Sensitive: true,
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
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed. ",
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored",
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc
			},
			"values": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of values in raw yaml format to pass to helm.",
			},
			"set": schema.SetNestedAttribute{
				Description: "Custom values to be merged with the values.",
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
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string"),
							},
						},
					},
				},
			},
			"set_list": schema.ListNestedAttribute{
				Description: "Custom sensitive values to be merged with the values ",
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
				Description: "Custom sensitive values to be merged with the values.",
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
			//looking into this
			//DEPRECIATED
			"set_string": schema.SetNestedAttribute{
				Description: "Custom string values to be merged with the values.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
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
			"namespace": schema.StringAttribute{
				Description: "Namespace to install the release info.",
				Optional:    true,
			},
			"verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Verify the package before installing it.",
			},
			"keyring": schema.StringAttribute{
				Optional: true,
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc
				Description: "Location of public keys used for verification. Used only if `verify` is true",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "Time in seconds to wait for any individual kubernetes operation.",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Description: "Prevent hooks from running.",
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
			},
			"reset_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reset the values to the ones built into the chart",
			},
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
			},
			"skip_tests": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, tests will not be rendered. By default, tests are rendered",
			},
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, render subchart notes along with the parent",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Description: "Run helm dependency update before installing the chart",
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
			},
			"description": schema.StringAttribute{
				Optional: true,
				//Currently looking into this, it is a big talking point in the migration for other engineers
				//DiffSuppressFunc
				Description: "Add a custom description",
			},
			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Description: "Create the namespace if it does not exist",
			},

			"postrender": schema.ListNestedAttribute{
				Description: "Postrender command configuration",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"binary_path": schema.StringAttribute{
							Required:    true,
							Description: "The command binary path.",
						},
					},
				},
			},
			"api_versions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes api versions used for Capabilities.APIVersions",
			},
			"include_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "Include CRDs in the templated output",
			},
			"is_upgrade": schema.BoolAttribute{
				Optional:    true,
				Description: "Set .Release.IsUpgrade instead of .Release.IsInstall",
			},
			"show_only": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only show manifests rendered from the given templates",
			},
			"validate": schema.BoolAttribute{
				Optional:    true,
				Description: "Validate your manifests against the Kubernetes cluster you are currently pointing at. This is the same validation performed on an install",
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
				Description: "Kubernetes version used for Capabilities.KubeVersion",
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
func (d *DataTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m := d.meta

	actionConfig, err := m.GetHelmConfiguration(ctx, state.Namespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Helm configuration",
			fmt.Sprintf("Error getting Helm configuration: %s", err),
		)
		return
	}

	err = OCIRegistryPerformLogin(ctx, m.RegistryClient, state.Repository.ValueString(), state.RepositoryUsername.ValueString(), state.RepositoryPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error logging into OCI Registry",
			fmt.Sprintf("Error logging into OCI Registry: %s", err),
		)
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
	if len(state.ShowOnly) > 0 {
		for _, f := range state.ShowOnly {
			fString := filepath.ToSlash(f.ValueString())
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

	state.CRDs = convertStringsToTypesStrings(chartCRDs)
	state.Manifests = computedManifests
	state.Manifest = types.StringValue(computedManifest.String())
	state.Notes = types.StringValue(rel.Info.Notes)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getTemplateValues(ctx context.Context, model *DataTemplateModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

	// Process "values" field
	for _, raw := range model.Values {
		if raw.IsNull() {
			continue
		}

		values := raw.ValueString()
		if values == "" {
			continue
		}

		currentMap := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(values), &currentMap); err != nil {
			diags.AddError("Failed to unmarshal values", fmt.Sprintf("---> %v %s", err, values))
			return nil, diags
		}

		base = mergeMaps(base, currentMap)
	}

	// Process "set" field
	for _, raw := range model.Set {
		set := raw
		if err := getDataValue(base, set); err.HasError() {
			diags.Append(err...)
			return nil, diags
		}
	}

	// Process "set_list" field
	for _, raw := range model.SetList {
		setList := raw
		if err := getDataListValue(ctx, base, setList); err.HasError() {
			diags.Append(err...)
			return nil, diags
		}
	}

	// Process "set_sensitive" field
	for _, raw := range model.SetSensitive {
		set := raw
		if err := getDataSensitiveValue(base, set); err.HasError() {
			diags.Append(err...)
			return nil, diags
		}
	}

	return base, logDataValues(ctx, base, model)
}

// For the type SetSensitiveValue
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

func logDataValues(ctx context.Context, values map[string]interface{}, model *DataTemplateModel) diag.Diagnostics {
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

	cloakDataSetValues(c, model)

	y, err := yaml.Marshal(c)
	if err != nil {
		diags.AddError("Error marshaling map to YAML", fmt.Sprintf("Failed to marshal map to YAML: %s", err))
	}

	tflog.Debug(ctx, fmt.Sprintf("---[ values.yaml ]-----------------------------------\n%s\n", string(y)))

	return diags
}

func cloakDataSetValues(config map[string]interface{}, model *DataTemplateModel) {
	for _, set := range model.SetSensitive {
		cloakSetValue(config, set.Name.ValueString())
	}
}

func getDataListValue(ctx context.Context, base map[string]interface{}, set SetListValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()
	listValue := set.Value
	listStringArray := make([]string, len(listValue))
	for i, v := range listValue {
		listStringArray[i] = v.ValueString()
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

func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}
