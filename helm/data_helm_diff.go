// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/sergi/go-diff/diffmatchpatch"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/releaseutil"
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
	Chart                    types.String     `tfsdk:"chart"`
	CreateNamespace          types.Bool       `tfsdk:"create_namespace"`
	DependencyUpdate         types.Bool       `tfsdk:"dependency_update"`
	Description              types.String     `tfsdk:"description"`
	Devel                    types.Bool       `tfsdk:"devel"`
	DisableOpenAPIValidation types.Bool       `tfsdk:"disable_openapi_validation"`
	DisableWebhooks          types.Bool       `tfsdk:"disable_webhooks"`
	ID                       types.String     `tfsdk:"id"`
	IncludeCRDs              types.Bool       `tfsdk:"include_crds"`
	Keyring                  types.String     `tfsdk:"keyring"`
	KubeVersion              types.String     `tfsdk:"kube_version"`
	Name                     types.String     `tfsdk:"name"`
	Namespace                types.String     `tfsdk:"namespace"`
	PassCredentials          types.Bool       `tfsdk:"pass_credentials"`
	PostRender               *PostRenderModel `tfsdk:"postrender"`
	RenderSubchartNotes      types.Bool       `tfsdk:"render_subchart_notes"`
	Repository               types.String     `tfsdk:"repository"`
	RepositoryCaFile         types.String     `tfsdk:"repository_ca_file"`
	RepositoryCertFile       types.String     `tfsdk:"repository_cert_file"`
	RepositoryKeyFile        types.String     `tfsdk:"repository_key_file"`
	RepositoryPassword       types.String     `tfsdk:"repository_password"`
	RepositoryUsername       types.String     `tfsdk:"repository_username"`
	Set                      types.Set        `tfsdk:"set"`
	SetList                  types.List       `tfsdk:"set_list"`
	SetSensitive             types.Set        `tfsdk:"set_sensitive"`
	SetWO                    types.List       `tfsdk:"set_wo"`
	SkipCrds                 types.Bool       `tfsdk:"skip_crds"`
	Timeout                  types.Int64      `tfsdk:"timeout"`
	Timeouts                 timeouts.Value   `tfsdk:"timeouts"`
	Validate                 types.Bool       `tfsdk:"validate"`
	Values                   types.List       `tfsdk:"values"`
	Version                  types.String     `tfsdk:"version"`
	Verify                   types.Bool       `tfsdk:"verify"`
	Diff                     types.String     `tfsdk:"diff"`
	DiffJSON                 types.String     `tfsdk:"diff_json"`
	HasChanges               types.Bool       `tfsdk:"has_changes"`
	CurrentManifest          types.String     `tfsdk:"current_manifest"`
	ProposedManifest         types.String     `tfsdk:"proposed_manifest"`
}

type ResourceInfo struct {
	Kind      string
	Name      string
	Namespace string
	Content   string
}

type ModifiedResource struct {
	Kind       string
	Name       string
	Namespace  string
	OldContent string
	NewContent string
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
		Description: "Data source to compute the diff between a deployed Helm release and a proposed rendered chart.",
		Attributes: map[string]schema.Attribute{
			"api_versions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes api versions used for Capabilities.APIVersions.",
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
			"skip_crds": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present.",
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
			"diff": schema.StringAttribute{
				Computed:    true,
				Description: "Unified diff output",
			},
			"diff_json": schema.StringAttribute{
				Computed:    true,
				Description: "Structured JSON diff",
			},
			"has_changes": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether any changes were detected",
			},
			"current_manifest": schema.StringAttribute{
				Computed:    true,
				Description: "Currently deployed manifest",
			},
			"proposed_manifest": schema.StringAttribute{
				Computed:    true,
				Description: "Proposed manifest",
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
	if state.DisableWebhooks.IsNull() || state.DisableWebhooks.IsUnknown() {
		state.DisableWebhooks = types.BoolValue(false)
	}
	if state.SkipCrds.IsNull() || state.SkipCrds.IsUnknown() {
		state.SkipCrds = types.BoolValue(false)
	}
	if state.RenderSubchartNotes.IsNull() || state.RenderSubchartNotes.IsUnknown() {
		state.RenderSubchartNotes = types.BoolValue(false)
	}
	if state.DisableOpenAPIValidation.IsNull() || state.DisableOpenAPIValidation.IsUnknown() {
		state.DisableOpenAPIValidation = types.BoolValue(false)
	}
	if state.DependencyUpdate.IsNull() || state.DependencyUpdate.IsUnknown() {
		state.DependencyUpdate = types.BoolValue(false)
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
	client := action.NewInstall(actionConfig)

	cpo, chartName, cpoDiags := chartPathOptionsModel((*HelmTemplateModel)(unsafe.Pointer(&state)), meta, &client.ChartPathOptions)
	resp.Diagnostics.Append(cpoDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, chartPath, chartDiags := getChartModel(ctx, (*HelmTemplateModel)(unsafe.Pointer(&state)), meta, chartName, cpo)
	resp.Diagnostics.Append(chartDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, depDiags := checkChartDependenciesModel(ctx, (*HelmTemplateModel)(unsafe.Pointer(&state)), c, chartPath, meta)
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

	values, valuesDiags := getValuesModel(ctx, (*HelmTemplateModel)(unsafe.Pointer(&state)))
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
	client.DependencyUpdate = state.DependencyUpdate.ValueBool()
	client.DisableHooks = state.DisableWebhooks.ValueBool()
	client.DisableOpenAPIValidation = state.DisableOpenAPIValidation.ValueBool()
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

	getClient := action.NewGet(actionConfig)
	currentRelease, err := getClient.Run(state.Name.ValueString())
	var currentManifest string
	if err != nil {
		currentManifest = ""
	} else {
		currentManifest = currentRelease.Manifest
	}

	var proposedManifest bytes.Buffer
	fmt.Fprintln(&proposedManifest, strings.TrimSpace(rel.Manifest))
	if !client.DisableHooks {
		for _, m := range rel.Hooks {
			fmt.Fprintf(&proposedManifest, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
		}
	}

	diff, diffJSON, hasChanges := compareManifests(
		ctx,
		currentManifest,
		proposedManifest.String(),
	)

	state.CurrentManifest = types.StringValue(currentManifest)
	state.ProposedManifest = types.StringValue(proposedManifest.String())
	state.Diff = types.StringValue(diff)
	state.DiffJSON = types.StringValue(diffJSON)
	state.HasChanges = types.BoolValue(hasChanges)
	state.ID = types.StringValue(state.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func compareManifests(ctx context.Context, current, proposed string) (diff string, diffJSON string, hasChanges bool) {
	currentResources := releaseutil.SplitManifests(current)
	proposedResources := releaseutil.SplitManifests(proposed)
	tflog.Debug(ctx, fmt.Sprintf("Current resources: %d, Proposed resources: %d",
		len(currentResources), len(proposedResources)))

	added, modified, deleted := matchResources(ctx, currentResources, proposedResources)

	diff = generateUnifiedDiff(added, modified, deleted)

	diffJSON = generateDiffJSON(added, modified, deleted)

	hasChanges = len(added) > 0 || len(modified) > 0 || len(deleted) > 0

	return diff, diffJSON, hasChanges

}

func matchResources(ctx context.Context, current, proposed map[string]string) (added []ResourceInfo, modified []ModifiedResource, deleted []ResourceInfo) {
	currentParsed := parseResources(ctx, current)
	proposedParsed := parseResources(ctx, proposed)

	currentMap := make(map[string]ResourceInfo)
	for _, res := range currentParsed {
		key := fmt.Sprintf("%s/%s/%s", res.Kind, res.Namespace, res.Name)
		currentMap[key] = res
	}

	proposedMap := make(map[string]ResourceInfo)
	for _, res := range proposedParsed {
		key := fmt.Sprintf("%s/%s/%s", res.Kind, res.Namespace, res.Name)
		proposedMap[key] = res
	}

	for key, proposedRes := range proposedMap {
		currentRes, exists := currentMap[key]
		if exists {
			if currentRes.Content != proposedRes.Content {
				modified = append(modified, ModifiedResource{
					Kind:       proposedRes.Kind,
					Name:       proposedRes.Name,
					Namespace:  proposedRes.Namespace,
					OldContent: currentRes.Content,
					NewContent: proposedRes.Content,
				})
			}
		} else {
			added = append(added, proposedRes)
		}
	}

	for key, currentRes := range currentMap {
		_, exists := proposedMap[key]
		if !exists {
			deleted = append(deleted, currentRes)
		}
	}

	// Sort resources for consistent output
	sort.Slice(added, func(i, j int) bool {
		return added[i].Kind+"/"+added[i].Namespace+"/"+added[i].Name <
			added[j].Kind+"/"+added[j].Namespace+"/"+added[j].Name
	})
	sort.Slice(modified, func(i, j int) bool {
		return modified[i].Kind+"/"+modified[i].Namespace+"/"+modified[i].Name <
			modified[j].Kind+"/"+modified[j].Namespace+"/"+modified[j].Name
	})
	sort.Slice(deleted, func(i, j int) bool {
		return deleted[i].Kind+"/"+deleted[i].Namespace+"/"+deleted[i].Name <
			deleted[j].Kind+"/"+deleted[j].Namespace+"/"+deleted[j].Name
	})

	return added, modified, deleted
}

func parseResources(ctx context.Context, manifests map[string]string) []ResourceInfo {
	var resources []ResourceInfo

	for _, content := range manifests {
		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &resource); err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Failed to parse resource: %v", err))
			continue
		}

		kind, _ := resource["kind"].(string)

		metadata, ok := resource["metadata"].(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := metadata["name"].(string)
		namespace, _ := metadata["namespace"].(string)
		if namespace == "" {
			namespace = "default"
		}

		resources = append(resources, ResourceInfo{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
			Content:   content,
		})
	}

	return resources
}

func generateUnifiedDiff(added []ResourceInfo, modified []ModifiedResource, deleted []ResourceInfo) string {
	var buf bytes.Buffer

	for _, res := range deleted {
		fmt.Fprintf(&buf, "%s, %s, %s has been removed:\n", res.Namespace, res.Name, res.Kind)
		for _, line := range strings.Split(res.Content, "\n") {
			if line != "" {
				fmt.Fprintf(&buf, "- %s\n", line)
			}
		}
		fmt.Fprintln(&buf)
	}

	for _, res := range added {
		fmt.Fprintf(&buf, "%s, %s, %s has been added:\n", res.Namespace, res.Name, res.Kind)
		for _, line := range strings.Split(res.Content, "\n") {
			if line != "" {
				fmt.Fprintf(&buf, "+ %s\n", line)
			}
		}
		fmt.Fprintln(&buf)
	}

	dmp := diffmatchpatch.New()
	for _, res := range modified {
		fmt.Fprintf(&buf, "%s, %s, %s has changed:\n", res.Namespace, res.Name, res.Kind)

		diffs := dmp.DiffMain(res.OldContent, res.NewContent, false)
		diffs = dmp.DiffCleanupSemantic(diffs)

		for _, diff := range diffs {
			text := diff.Text
			if text == "" {
				continue
			}

			lines := strings.Split(text, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}

				switch diff.Type {
				case diffmatchpatch.DiffDelete:
					fmt.Fprintf(&buf, "- %s\n", line)
				case diffmatchpatch.DiffInsert:
					fmt.Fprintf(&buf, "+ %s\n", line)
				case diffmatchpatch.DiffEqual:
					fmt.Fprintf(&buf, "  %s\n", line)
				}
			}
		}
		fmt.Fprintln(&buf)
	}

	return buf.String()
}

func generateDiffJSON(added []ResourceInfo, modified []ModifiedResource, deleted []ResourceInfo) string {
	result := map[string]interface{}{
		"added":    resourceInfoJSON(added),
		"modified": modifiedResourceJSON(modified),
		"deleted":  resourceInfoJSON(deleted),
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return string(jsonBytes)
}

func resourceInfoJSON(resources []ResourceInfo) []map[string]string {
	var result []map[string]string
	for _, res := range resources {
		result = append(result, map[string]string{
			"kind":      res.Kind,
			"name":      res.Name,
			"namespace": res.Namespace,
		})
	}
	return result
}

func modifiedResourceJSON(resources []ModifiedResource) []map[string]string {
	var result []map[string]string
	for _, res := range resources {
		result = append(result, map[string]string{
			"kind":      res.Kind,
			"name":      res.Name,
			"namespace": res.Namespace,
		})
	}
	return result
}
