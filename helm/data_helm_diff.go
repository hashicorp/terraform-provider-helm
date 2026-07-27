// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/sergi/go-diff/diffmatchpatch"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/yaml"
)

var (
	_ datasource.DataSource              = &HelmDiff{}
	_ datasource.DataSourceWithConfigure = &HelmDiff{}
)

func NewHelmDiff() datasource.DataSource {
	return &HelmDiff{}
}

type HelmDiff struct {
	meta *Meta
}

type HelmDiffModel struct {
	APIVersions              types.List       `tfsdk:"api_versions"`
	Atomic                   types.Bool       `tfsdk:"atomic"`
	Chart                    types.String     `tfsdk:"chart"`
	CreateNamespace          types.Bool       `tfsdk:"create_namespace"`
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
	Validate                 types.Bool       `tfsdk:"validate"`
	Values                   types.List       `tfsdk:"values"`
	Version                  types.String     `tfsdk:"version"`
	Verify                   types.Bool       `tfsdk:"verify"`
	Wait                     types.Bool       `tfsdk:"wait"`

	CurrentManifest  types.String `tfsdk:"current_manifest"`
	ProposedManifest types.String `tfsdk:"proposed_manifest"`
	Diff             types.String `tfsdk:"diff"`
}

func (d *HelmDiff) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData != nil {
		d.meta = req.ProviderData.(*Meta)
	}
}

func (d *HelmDiff) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_diff"
}

func (d *HelmDiff) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data source to diff the rendered Helm chart templates against the currently deployed release.",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Release name",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "Namespace to install the release into.",
			},
			"notes": schema.StringAttribute{
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

			// Output attributes
			"current_manifest": schema.StringAttribute{
				Computed:    true,
				Description: "The rendered manifest of the currently deployed release.",
			},
			"proposed_manifest": schema.StringAttribute{
				Computed:    true,
				Description: "The rendered manifest of the proposed chart templates with the configured values.",
			},
			"diff": schema.StringAttribute{
				Computed:    true,
				Description: "The diff between the current and proposed manifests in unified diff format.",
			},
		},
	}
}

func (d *HelmDiff) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state HelmDiffModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Description.IsNull() || state.Description.ValueString() == "" {
		state.Description = types.StringValue("")
	}
	if state.Devel.IsNull() || state.Devel.IsUnknown() {
		if !state.Version.IsNull() && state.Version.ValueString() != "" {
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

	// Get the currently deployed release manifest
	currentManifest, err := getDeployedReleaseManifest(ctx, actionConfig, state.Name.ValueString())
	if err != nil && err != errReleaseNotFound {
		resp.Diagnostics.AddError(
			"Failed to get deployed release",
			fmt.Sprintf("Error retrieving release %q: %s", state.Name.ValueString(), err),
		)
		return
	}

	// Render the proposed manifest using a dry-run install
	client := action.NewInstall(actionConfig)
	cpo, chartName, cpoDiags := diffChartPathOptions(&state, meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(cpoDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	path, err := meta.LocateChart(cpo, chartName)
	if err != nil {
		resp.Diagnostics.AddError("Error locating chart", fmt.Sprintf("Unable to locate chart %s: %s", chartName, err))
		return
	}

	c, err := loader.Load(path)
	if err != nil {
		resp.Diagnostics.AddError("Error loading chart", fmt.Sprintf("Unable to load chart %s: %s", path, err))
		return
	}

	if state.DependencyUpdate.ValueBool() {
		p := getter.All(meta.Settings)
		if req := c.Metadata.Dependencies; req != nil {
			err := action.CheckDependencies(c, req)
			if err != nil {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          state.Keyring.ValueString(),
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: meta.Settings.RepositoryConfig,
					RepositoryCache:  meta.Settings.RepositoryCache,
					Debug:            meta.Settings.Debug,
				}
				tflog.Debug(ctx, "Downloading chart dependencies...")
				if err := man.Update(); err != nil {
					resp.Diagnostics.AddError("Failed to update chart dependencies", fmt.Sprintf("Error: %s", err))
					return
				}
				c, err = loader.Load(path)
				if err != nil {
					resp.Diagnostics.AddError("Error loading chart", fmt.Sprintf("Could not reload chart after updating dependencies: %s", err))
					return
				}
			}
		}
	}

	values, valuesDiags := diffGetValues(ctx, &state)
	resp.Diagnostics.Append(valuesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := isChartInstallable(c); err != nil {
		resp.Diagnostics.AddError("Error checking if chart is installable", fmt.Sprintf("Chart is not installable: %s", err))
		return
	}
	client.ChartPathOptions = *cpo
	client.ClientOnly = !state.Validate.ValueBool()
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
	client.DryRun = true
	client.ClientOnly = !state.Validate.ValueBool()
	client.APIVersions = chartutil.VersionSet(apiVersions)
	client.IncludeCRDs = state.IncludeCRDs.ValueBool()

	rel, err := client.Run(c, values)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error running Helm dry-run install",
			fmt.Sprintf("Error running Helm dry-run install: %s", err),
		)
		return
	}

	proposedManifest := buildProposedManifest(rel, state.SkipTests.ValueBool(), client.DisableHooks)

	diff := computeManifestDiff(currentManifest, proposedManifest)

	state.CurrentManifest = types.StringValue(currentManifest)
	state.ProposedManifest = types.StringValue(proposedManifest)
	state.Diff = types.StringValue(diff)
	state.Notes = types.StringValue(rel.Info.Notes)
	state.ID = types.StringValue(state.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func getDeployedReleaseManifest(ctx context.Context, cfg *action.Configuration, name string) (string, error) {
	get := action.NewGet(cfg)
	res, err := get.Run(name)
	if err != nil {
		if strings.Contains(err.Error(), "release: not found") {
			return "", errReleaseNotFound
		}
		return "", err
	}
	return res.Manifest, nil
}

func buildProposedManifest(rel *release.Release, skipTests bool, disableHooks bool) string {
	var manifestBuilder strings.Builder
	manifestBuilder.WriteString(strings.TrimSpace(rel.Manifest))
	if !disableHooks {
		for _, m := range rel.Hooks {
			if skipTests && isTestHook(m) {
				continue
			}
			fmt.Fprintf(&manifestBuilder, "\n---\n# Source: %s\n%s\n", m.Path, m.Manifest)
		}
	}
	return manifestBuilder.String()
}

func computeManifestDiff(current, proposed string) string {
	if current == "" && proposed == "" {
		return ""
	}
	if current == "" {
		lines := strings.Split(strings.TrimRight(proposed, "\n"), "\n")
		diff := fmt.Sprintf("--- \n+++ \n@@ -0,0 +1,%d @@\n", len(lines))
		for _, line := range lines {
			diff += "+" + line + "\n"
		}
		return diff
	}
	if proposed == "" {
		lines := strings.Split(strings.TrimRight(current, "\n"), "\n")
		diff := fmt.Sprintf("--- \n+++ \n@@ -1,%d +0,0 @@\n", len(lines))
		for _, line := range lines {
			diff += "-" + line + "\n"
		}
		return diff
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(current, proposed, true)
	diffs = dmp.DiffCleanupSemantic(diffs)

	return dmp.DiffPrettyText(diffs)
}

func diffChartPathOptions(model *HelmDiffModel, meta *Meta, cpo *action.ChartPathOptions) (*action.ChartPathOptions, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	chartName := model.Chart.ValueString()
	repository := model.Repository.ValueString()

	var repositoryURL string
	if registry.IsOCI(repository) {
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

	version := model.Version.ValueString()
	if version == "" && model.Devel.ValueBool() {
		version = ">0.0.0-0"
	}
	version = strings.TrimSpace(version)

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

func diffGetValues(ctx context.Context, model *HelmDiffModel) (map[string]interface{}, diag.Diagnostics) {
	base := map[string]interface{}{}
	var diags diag.Diagnostics

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

	return base, diags
}
