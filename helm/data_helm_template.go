// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"k8s.io/helm/pkg/strvals"
	"sigs.k8s.io/yaml"
)

var (
	_ datasource.DataSource              = &HelmTemplate{}
	_ datasource.DataSourceWithConfigure = &HelmTemplate{}
)

func NewHelmTemplate() datasource.DataSource {
	return &HelmTemplate{}
}

// HelmTemplate represents the data source for rendering Helm chart templates
type HelmTemplate struct {
	meta *Meta
}

// HelmTemplateModel holds the attributes for configuring the Helm chart templates
type HelmTemplateModel struct {
	APIVersions              types.List       `tfsdk:"api_versions"`
	Atomic                   types.Bool       `tfsdk:"atomic"`
	Chart                    types.String     `tfsdk:"chart"`
	CreateNamespace          types.Bool       `tfsdk:"create_namespace"`
	CRDs                     types.List       `tfsdk:"crds"`
	DependencyUpdate         types.Bool       `tfsdk:"dependency_update"`
	Description              types.String     `tfsdk:"description"`
	Devel                    types.Bool       `tfsdk:"devel"`
	DisableOpenAPIValidation types.Bool       `tfsdk:"disable_openapi_validation"`
	DisableWebhooks          types.Bool       `tfsdk:"disable_webhooks"`
	ID                       types.String     `tfsdk:"id"`
	IncludeCRDs              types.Bool       `tfsdk:"include_crds"`
	IsUpgrade                types.Bool       `tfsdk:"is_upgrade"`
	Keyring                  types.String     `tfsdk:"keyring"`
	KubeVersion              types.String     `tfsdk:"kube_version"`
	Manifest                 types.String     `tfsdk:"manifest"`
	Manifests                types.Map        `tfsdk:"manifests"`
	Name                     types.String     `tfsdk:"name"`
	Namespace                types.String     `tfsdk:"namespace"`
	Notes                    types.String     `tfsdk:"notes"`
	PassCredentials          types.Bool       `tfsdk:"pass_credentials"`
	PostRender               *PostRenderModel `tfsdk:"postrender"`
	RenderSubchartNotes      types.Bool       `tfsdk:"render_subchart_notes"`
	Replace                  types.Bool       `tfsdk:"replace"`
	Repository               types.String     `tfsdk:"repository"`
	RepositoryCaFile         types.String     `tfsdk:"repository_ca_file"`
	RepositoryCertFile       types.String     `tfsdk:"repository_cert_file"`
	RepositoryKeyFile        types.String     `tfsdk:"repository_key_file"`
	RepositoryPassword       types.String     `tfsdk:"repository_password"`
	RepositoryUsername       types.String     `tfsdk:"repository_username"`
	ResetValues              types.Bool       `tfsdk:"reset_values"`
	ReuseValues              types.Bool       `tfsdk:"reuse_values"`
	Set                      types.Set        `tfsdk:"set"`
	SetList                  types.List       `tfsdk:"set_list"`
	SetSensitive             types.Set        `tfsdk:"set_sensitive"`
	SetWO                    types.List       `tfsdk:"set_wo"`
	ShowOnly                 types.List       `tfsdk:"show_only"`
	SkipCrds                 types.Bool       `tfsdk:"skip_crds"`
	SkipTests                types.Bool       `tfsdk:"skip_tests"`
	Timeout                  types.Int64      `tfsdk:"timeout"`
	Timeouts                 timeouts.Value   `tfsdk:"timeouts"`
	Validate                 types.Bool       `tfsdk:"validate"`
	Values                   types.List       `tfsdk:"values"`
	Version                  types.String     `tfsdk:"version"`
	Verify                   types.Bool       `tfsdk:"verify"`
	Wait                     types.Bool       `tfsdk:"wait"`
}

// SetValue represents the custom value to be merged with the Helm chart values
type SetValue struct {
	Name  types.String `tfsdk:"name"`
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
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
	Type  types.String `tfsdk:"type"`
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
			"api_versions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes api versions used for Capabilities.APIVersions.",
			},
			"atomic": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, the installation process purges the chart on fail. The 'wait' flag will be set automatically if 'atomic' is used.",
			},
			"chart": schema.StringAttribute{
				Required:    true,
				Description: "Chart name to be installed. A path may be used.",
			},
			"crds": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of rendered CRDs from the chart.",
			},
			"create_namespace": schema.BoolAttribute{
				Optional:    true,
				Description: "Create the namespace if it does not exist.",
			},
			"dependency_update": schema.BoolAttribute{
				Optional:    true,
				Description: "Run helm dependency update before installing the chart.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Add a custom description.",
			},
			"devel": schema.BoolAttribute{
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored.",
			},
			"disable_openapi_validation": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema.",
			},
			"disable_webhooks": schema.BoolAttribute{
				Optional:    true,
				Description: "Prevent hooks from running.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"include_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "Include CRDs in the templated output.",
			},
			"is_upgrade": schema.BoolAttribute{
				Optional:    true,
				Description: "Set .Release.IsUpgrade instead of .Release.IsInstall.",
			},
			"keyring": schema.StringAttribute{
				Optional:    true,
				Description: "Location of public keys used for verification. Used only if `verify` is true.",
			},
			"kube_version": schema.StringAttribute{
				Optional:    true,
				Description: "Kubernetes version used for Capabilities.KubeVersion.",
			},
			"manifest": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.",
			},
			"manifests": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Map of rendered chart templates indexed by the template name.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Release name",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "Namespace to install the release into.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Rendered notes if the chart contains a `NOTES.txt`.",
			},
			"pass_credentials": schema.BoolAttribute{
				Optional:    true,
				Description: "Pass credentials to all domains",
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
			"render_subchart_notes": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, render subchart notes along with the parent.",
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production.",
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Repository where to locate the requested chart. If it is a URL the chart is installed without installing the repository.",
			},
			"repository_ca_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's CA file",
			},
			"repository_cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's cert file",
			},
			"repository_key_file": schema.StringAttribute{
				Optional:    true,
				Description: "The repository's cert key file",
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
				Description: "When upgrading, reset the values to the ones built into the chart.",
			},
			"reuse_values": schema.BoolAttribute{
				Optional:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored.",
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
							Optional: true,
						},
						"type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Validators: []validator.String{
								stringvalidator.OneOf("auto", "string", "literal"),
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
							Optional: true,
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
								stringvalidator.OneOf("auto", "string", "literal"),
							},
						},
					},
				},
			},
			"set_wo": schema.ListNestedAttribute{
				Description: "Write-only custom values to be merged with the values.",
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
			"show_only": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only show manifests rendered from the given templates.",
			},
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present.",
			},
			"skip_tests": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, tests will not be rendered. By default, tests are rendered.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "Time in seconds to wait for any individual Kubernetes operation.",
			},
			"timeouts": timeouts.Attributes(ctx),
			"validate": schema.BoolAttribute{
				Optional:    true,
				Description: "Validate your manifests against the Kubernetes cluster you are currently pointing at. This is the same validation performed on an install.",
			},
			"values": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of values in raw yaml format to pass to helm.",
			},
			"verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Verify the package before installing it.",
			},
			"version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed.",
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
		},
	}
}

// Reads the current state of the data template and will update the state with the data fetched
func (d *HelmTemplate) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state HelmTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diagsTimeout := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diagsTimeout...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// setting default values to false is attributes are not provided in the config
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
	if state.IncludeCRDs.IsNull() || state.IncludeCRDs.IsUnknown() {
		state.IncludeCRDs = types.BoolValue(false)
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

	meta := d.meta

	var apiVersions []string
	if !state.APIVersions.IsNull() && !state.APIVersions.IsUnknown() {
		var apiVersionElements []types.String
		diags := state.APIVersions.ElementsAs(ctx, &apiVersionElements, false)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}

		for _, apiVersion := range apiVersionElements {
			apiVersions = append(apiVersions, apiVersion.ValueString())
		}
	}

	var showFiles []string

	if !state.ShowOnly.IsNull() && state.ShowOnly.Elements() != nil {
		var showOnlyElements []types.String
		diags := state.ShowOnly.ElementsAs(ctx, &showOnlyElements, false)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}

		for _, raw := range showOnlyElements {
			if raw.IsNull() || raw.ValueString() == "" {
				continue
			}
			showFiles = append(showFiles, raw.ValueString())
		}
	}

	actionConfig, err := meta.GetHelmConfiguration(ctx, state.Namespace.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Helm configuration",
			fmt.Sprintf("There was an error retrieving Helm configuration for namespace %q: %s", state.Namespace.ValueString(), err),
		)
		return
	}
	diags := OCIRegistryLogin(ctx, meta, actionConfig, meta.RegistryClient, state.Repository.ValueString(), state.Chart.ValueString(), state.RepositoryUsername.ValueString(), state.RepositoryPassword.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	client := action.NewInstall(actionConfig)

	cpo, chartName, cpoDiags := chartPathOptionsModel(&state, meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(cpoDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, chartPath, chartDiags := getChartModel(ctx, &state, meta, chartName, cpo)
	resp.Diagnostics.Append(chartDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, depDiags := checkChartDependenciesModel(ctx, &state, c, chartPath, meta)
	resp.Diagnostics.Append(depDiags...)
	if resp.Diagnostics.HasError() {
		return
	} else if updated {
		c, err = loader.Load(chartPath)
		if err != nil {
			resp.Diagnostics.AddError("Error loading chart", fmt.Sprintf("Could not reload chart after updating dependencies: %s", err))
			return
		}
	}

	values, valuesDiags := getValuesModel(ctx, &state)
	resp.Diagnostics.Append(valuesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := isChartInstallable(c); err != nil {
		resp.Diagnostics.AddError("Error checking if chart is installable", fmt.Sprintf("Chart is not installable: %s", err))
		return
	}
	client.ChartPathOptions = *cpo
	client.ClientOnly = false
	client.ReleaseName = state.Name.ValueString()
	client.GenerateName = false
	client.NameTemplate = ""
	client.OutputDir = ""
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

	if state.KubeVersion.ValueString() != "" {
		parsedVer, err := chartutil.ParseKubeVersion(state.KubeVersion.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to parse Kubernetes version",
				fmt.Sprintf("couldn't parse string %q into kube-version: %s", state.KubeVersion.ValueString(), err),
			)
			return
		}
		client.KubeVersion = parsedVer
	}

	client.DryRun = true
	client.Replace = true
	client.ClientOnly = !state.Validate.ValueBool()
	client.APIVersions = chartutil.VersionSet(apiVersions)
	client.IncludeCRDs = state.IncludeCRDs.ValueBool()

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
	var manifestsToRender []string

	splitManifests := releaseutil.SplitManifests(manifests.String())
	manifestsKeys := make([]string, 0, len(splitManifests))
	for k := range splitManifests {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

	var chartCRDs []string
	for _, crd := range rel.Chart.CRDObjects() {
		chartCRDs = append(chartCRDs, string(crd.File.Data))
	}

	// Mapping of manifest key to manifest template name
	manifestNamesByKey := make(map[string]string, len(manifestsKeys))

	manifestNameRegex := regexp.MustCompile("# Source: [^/]+/(.+)")

	for _, manifestKey := range manifestsKeys {
		manifest := splitManifests[manifestKey]
		submatch := manifestNameRegex.FindStringSubmatch(manifest)
		if len(submatch) == 0 {
			continue
		}
		manifestName := submatch[1]
		manifestNamesByKey[manifestKey] = manifestName
	}

	if len(showFiles) > 0 {
		for _, f := range showFiles {
			missing := true
			f = filepath.ToSlash(f)

			for manifestKey, manifestName := range manifestNamesByKey {
				manifestPathSplit := strings.Split(manifestName, "/")
				manifestPath := strings.Join(manifestPathSplit, "/")

				if matched, _ := filepath.Match(f, manifestPath); !matched {
					continue
				}

				manifestsToRender = append(manifestsToRender, manifestKey)
				missing = false
			}

			if missing {
				resp.Diagnostics.AddError(
					"Template Not Found",
					fmt.Sprintf("Could not find template %q in chart", f),
				)
			}
		}
	} else {
		manifestsToRender = manifestsKeys
	}

	// We need to sort the manifests so the order stays stable when they are
	// concatenated back together in the computedManifests map
	sort.Strings(manifestsToRender)

	// Map from rendered manifests to data source output
	computedManifests := make(map[string]string, 0)
	computedManifest := &strings.Builder{}

	for _, manifestKey := range manifestsToRender {
		manifest := splitManifests[manifestKey]
		manifestName := manifestNamesByKey[manifestKey]

		// Manifests
		computedManifests[manifestName] = fmt.Sprintf("%s---\n%s\n", computedManifests[manifestName], manifest)

		// Manifest bundle
		fmt.Fprintf(computedManifest, "---\n%s\n", manifest)
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
	state.ID = types.StringValue(state.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getValuesModel(ctx context.Context, model *HelmTemplateModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

	// Process "values" attribute
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

	// Process "set" attribute
	if !model.Set.IsNull() {
		var setList []SetValue
		setDiags := model.Set.ElementsAs(ctx, &setList, false)
		diags.Append(setDiags...)
		if diags.HasError() {
			return nil, diags
		}

		for _, set := range setList {
			setDiags := applySetValue(base, set)
			diags.Append(setDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	// Process "set_list" attribute
	if !model.SetList.IsUnknown() {
		var setListSlice []SetListValue
		setListDiags := model.SetList.ElementsAs(ctx, &setListSlice, false)
		diags.Append(setListDiags...)
		if diags.HasError() {
			return nil, diags
		}

		for _, setList := range setListSlice {
			setListDiags := applySetListValue(ctx, base, setList)
			diags.Append(setListDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	// Process "set_sensitive" attribute
	if !model.SetSensitive.IsNull() {
		var setSensitiveList []SetSensitiveValue
		setSensitiveDiags := model.SetSensitive.ElementsAs(ctx, &setSensitiveList, false)
		diags.Append(setSensitiveDiags...)
		if diags.HasError() {
			return nil, diags
		}

		for _, setSensitive := range setSensitiveList {
			setSensitiveDiags := applySetSensitiveValue(base, setSensitive)
			diags.Append(setSensitiveDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}
	if !model.SetWO.IsNull() && !model.SetWO.IsUnknown() {
		var setWOList []SetValue
		setWODiags := model.SetWO.ElementsAs(ctx, &setWOList, false)
		diags.Append(setWODiags...)
		if diags.HasError() {
			return nil, diags
		}
		for _, set := range setWOList {
			setDiags := applySetValue(base, set)
			diags.Append(setDiags...)
			if diags.HasError() {
				return nil, diags
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Final merged values: %v", base))
	logDiags := LogValuesModel(ctx, base, model)
	diags.Append(logDiags...)
	return base, diags
}

func isTestHook(h *release.Hook) bool {
	for _, e := range h.Events {
		if e == release.HookTest {
			return true
		}
	}
	return false
}

func chartPathOptionsModel(model *HelmTemplateModel, meta *Meta, cpo *action.ChartPathOptions) (*action.ChartPathOptions, string, diag.Diagnostics) {
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

	version := getVersionModel(model)

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

func getVersionModel(model *HelmTemplateModel) string {
	version := model.Version.ValueString()
	if version == "" && model.Devel.ValueBool() {
		return ">0.0.0-0"
	}
	return strings.TrimSpace(version)
}

func getChartModel(ctx context.Context, model *HelmTemplateModel, meta *Meta, name string, cpo *action.ChartPathOptions) (*chart.Chart, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	tflog.Debug(ctx, fmt.Sprintf("Helm settings: %+v", meta.Settings))

	path, err := cpo.LocateChart(name, meta.Settings)
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

func checkChartDependenciesModel(ctx context.Context, model *HelmTemplateModel, c *chart.Chart, path string, meta *Meta) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	p := getter.All(meta.Settings)

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
					RepositoryConfig: meta.Settings.RepositoryConfig,
					RepositoryCache:  meta.Settings.RepositoryCache,
					Debug:            meta.Settings.Debug,
				}
				tflog.Debug(ctx, "Downloading chart dependencies...")
				if err := man.Update(); err != nil {
					diags.AddError("Failed to update chart dependencies", fmt.Sprintf("Error: %s", err))
					return true, diags
				}
				return true, diags
			}
			diags.AddError("Missing chart dependencies", "Found in Chart.yaml, but missing in charts/ directory.")
			return false, diags
		}
	}
	tflog.Debug(ctx, "Chart dependencies are up to date.")
	return false, diags
}

func applySetValue(base map[string]interface{}, set SetValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := set.Name.ValueString()
	value := set.Value.ValueString()
	valueType := set.Type.ValueString()

	switch valueType {
	case "auto", "":
		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing value", fmt.Sprintf("Key %q with value %s: %s", name, value, err))
		}
	case "string":
		if err := strvals.ParseIntoString(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing string value", fmt.Sprintf("Key %q with value %s: %s", name, value, err))
		}
	case "literal":
		var literal interface{}
		if err := yaml.Unmarshal([]byte(fmt.Sprintf("%s: %s", name, value)), &literal); err != nil {
			diags.AddError("Failed parsing literal value", fmt.Sprintf("Key %q with literal value %s: %s", name, value, err))
			return diags
		}
		if m, ok := literal.(map[string]interface{}); ok {
			base[name] = m[name]
		} else {
			base[name] = literal
		}
	default:
		diags.AddError("Unexpected type", fmt.Sprintf("Unexpected type: %s", valueType))
	}
	return diags
}

func applySetListValue(ctx context.Context, base map[string]interface{}, setList SetListValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := setList.Name.ValueString()

	if setList.Value.IsNull() {
		diags.AddError("Null List Value", "The list value is null.")
		return diags
	}

	// Extract elements from the list value
	elements := setList.Value.Elements()

	listStringArray := make([]string, 0, len(elements))
	for _, element := range elements {
		if !element.IsNull() {
			strValue := element.(types.String).ValueString()
			listStringArray = append(listStringArray, strValue)
		}
	}

	listString := strings.Join(listStringArray, ",")

	// Parse the joined string into the base map
	if err := strvals.ParseInto(fmt.Sprintf("%s={%s}", name, listString), base); err != nil {
		diags.AddError("Error parsing list value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, listString, err))
		return diags
	}

	return diags
}

func applySetSensitiveValue(base map[string]interface{}, setSensitive SetSensitiveValue) diag.Diagnostics {
	var diags diag.Diagnostics

	name := setSensitive.Name.ValueString()
	value := setSensitive.Value.ValueString()
	valueType := setSensitive.Type.ValueString()

	switch valueType {
	case "auto", "":
		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing sensitive value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, value, err))
		}
	case "string":
		if err := strvals.ParseIntoString(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			diags.AddError("Failed parsing sensitive string value", fmt.Sprintf("Failed parsing key %q with value %s: %s", name, value, err))
		}
	default:
		diags.AddError("Unexpected type", fmt.Sprintf("Unexpected type for sensitive value: %s", valueType))
	}

	return diags
}

func LogValuesModel(ctx context.Context, values map[string]interface{}, state *HelmTemplateModel) diag.Diagnostics {
	var diags diag.Diagnostics

	asJSON, err := json.Marshal(values)
	if err != nil {
		diags.AddError("Error marshaling values to JSON", fmt.Sprintf("Failed to marshal values to JSON: %s", err))
		return diags
	}

	var clonedValues map[string]interface{}
	err = json.Unmarshal(asJSON, &clonedValues)
	if err != nil {
		diags.AddError("Error unmarshaling JSON to map", fmt.Sprintf("Failed to unmarshal JSON to map: %s", err))
		return diags
	}

	// Apply cloaking or masking for sensitive values
	cloakSetValuesModel(clonedValues, state)

	// Convert the modified map to YAML for logging purposes
	yamlData, err := yaml.Marshal(clonedValues)
	if err != nil {
		diags.AddError("Error marshaling map to YAML", fmt.Sprintf("Failed to marshal map to YAML: %s", err))
		return diags
	}

	// Log the final YAML representation of the values
	tflog.Debug(ctx, fmt.Sprintf("---[ values.yaml ]-----------------------------------\n%s\n", string(yamlData)))

	return diags
}

func cloakSetValuesModel(config map[string]interface{}, state *HelmTemplateModel) {
	if !state.SetSensitive.IsNull() {
		var setSensitiveList []SetSensitiveValue
		diags := state.SetSensitive.ElementsAs(context.Background(), &setSensitiveList, false)
		if diags.HasError() {
			tflog.Warn(context.Background(), "Error parsing SetSensitive elements", map[string]interface{}{
				"diagnostics": diags,
			})
			return
		}

		for _, set := range setSensitiveList {
			cloakSetValueModel(config, set.Name.ValueString())
		}
	}
	if !state.SetWO.IsNull() && !state.SetWO.IsUnknown() {
		var setWOList []SetValue
		diags := state.SetWO.ElementsAs(context.Background(), &setWOList, false)
		if diags.HasError() {
			tflog.Warn(context.Background(), "Error parsing SetWO elements", map[string]interface{}{"diagnostics": diags})
			return
		}
		for _, set := range setWOList {
			cloakSetValueModel(config, set.Name.ValueString())
		}
	}
}

const sensitiveContentModelValue = "(sensitive value)"

func cloakSetValueModel(values map[string]interface{}, valuePath string) {
	pathKeys := strings.Split(valuePath, ".")
	sensitiveKey := pathKeys[len(pathKeys)-1]
	parentPathKeys := pathKeys[:len(pathKeys)-1]

	currentMap := values
	for _, key := range parentPathKeys {
		v, ok := currentMap[key].(map[string]interface{})
		if !ok {
			return
		}
		currentMap = v
	}
	currentMap[sensitiveKey] = sensitiveContentModelValue
}
