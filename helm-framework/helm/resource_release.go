package helm

//just define aq resource called helm_release
// make the struct helmRelease
// Dont implement the schema, func bodies, and the crud operations.
// When plan is run, it will call configure.
// Make sure to implement in smaller steps. Small iterations
// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"strings"
// 	"sync"
// 	"time"

// 	//"sync"

// 	//"fmt"
// 	"github.com/hashicorp/terraform-plugin-framework/attr"
// 	"github.com/hashicorp/terraform-plugin-framework/diag"
// 	"github.com/hashicorp/terraform-plugin-framework/resource"
// 	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
// 	"github.com/hashicorp/terraform-plugin-framework/types"
// 	"github.com/hashicorp/terraform-plugin-log/tflog"
// 	"helm.sh/helm/v3/pkg/action"
// 	"helm.sh/helm/v3/pkg/cli"
// 	"helm.sh/helm/v3/pkg/registry"
// 	"helm.sh/helm/v3/pkg/release"
// 	//"github.com/hashicorp/terraform-plugin-framework/types"
// 	//"github.com/hashicorp/terraform-plugin-framework/resource/schema/attribute"
// )

// // Added functionality providing a way to pass and store the meta info in the HelmReleaseResource struct
// type HelmReleaseResource struct {
// 	// Pointing to the Meta struct
// 	meta *MetaRelease
// }

// func NewHelmReleaseResource(meta *MetaRelease) resource.Resource {
// 	return &HelmReleaseResource{
// 		meta: meta,
// 	}
// }

// type helmReleaseModel struct {
// 	Name                 types.String `tfsdk:"name"`
// 	Repository           types.String `tfsdk:"repository"`
// 	REPOSITORY_KEY_FILE  types.String `tfsdk:"repository_key_file"`
// 	REPOSITORY_CERT_FILE types.String `tfsdk:"repository_cert_file"`
// 	REPOSITORY_CA_FILE   types.String `tfsdk:"repository_ca_file"`
// 	REPOSITORY_USERNAME  types.String `tfsdk:"repository_username"`
// 	REPOSTIROY_PASSWORD  types.String `tfsdk:"repository_password"`
// 	PASS_CREDENTIALS     types.Bool   `tfsdk:"pass_credentials"`
// 	CHART                types.String `tfsdk:"chart"`
// 	VERSION              types.String `tfsdk:"version"`
// 	DEVEL                types.Bool   `tfsdk:"devel"`
// 	VALUES               types.List   `tfsdk:"devel"`
// 	// Might need changing
// 	SET []setResourceModel `tfsdk:"set"`
// 	// Might need changing
// 	SET_LIST []set_listResourceModel `tfsdk:"set_list"`
// 	// Might need changing
// 	SET_SENSITIVE              []set_sensitiveResourceModel `tfsdk:"set_sensitive"`
// 	NAMESPACE                  types.String                 `tfsdk:"namespace"`
// 	VERIFY                     types.Bool                   `tfsdk:"verify"`
// 	KEYRING                    types.String                 `tfsdk:"keyring"`
// 	TIMEOUT                    types.Int64                  `tfsdk:"timeout"`
// 	DISABLE_WEBHOOKS           types.Bool                   `tfsdk:"disable_webhooks"`
// 	DISABLE_CRD_HOOKS          types.Bool                   `tfsdk:"disable_crd_hooks"`
// 	RESET_VALUES               types.Bool                   `tfsdk:"reset_values"`
// 	FORCE_UPDATE               types.Bool                   `tfsdk:"force_update"`
// 	RECREATE_PODS              types.Bool                   `tfsdk:"recreate_pods"`
// 	CLEANUP_ON_FAIL            types.Bool                   `tfsdk:"cleanup_on_fail"`
// 	MAX_HISTROY                types.Int64                  `tfsdk:"max_history"`
// 	ATOMIC                     types.Bool                   `tfsdk:"atomic"`
// 	SKIP_CRDS                  types.Bool                   `tfsdk:"skip_crds"`
// 	RENDER_SUBCHART_NOTES      types.Bool                   `tfsdk:"render_subchart_notes"`
// 	DISABLE_OPENAPI_VALIDATION types.Bool                   `tfsdk:"disable_openapi_validation"`
// 	WAIT                       types.Bool                   `tfsdk:"wait"`
// 	WAIT_FOR_JOBS              types.Bool                   `tfsdk:"wait_for_jobs"`
// 	STATUS                     types.String                 `tfsdk:"STATUS"`
// 	DEPENDENCY_UPDATE          types.Bool                   `tfsdk:"dependency_update"`
// 	REPLACE                    types.Bool                   `tfsdk:"replace"`
// 	DESCRIPTION                types.String                 `tfsdk:"description"`
// 	CREATE_NAMESPACE           types.Bool                   `tfsdk:"create_namespace"`
// 	// Might need to change
// 	POSTRENDER types.List `tfsdk:"postrender"`
// 	LINT       types.Bool `tfsdk:"lint"`
// 	// Might need to change
// 	MANIFEST types.String `tfsdk:"manifest"`
// 	// Might need to change
// 	METADATA types.List `tfsdk:"metadata"`
// }

// type setResourceModel struct {
// 	NAME  types.String `tfsdk:"name"`
// 	VALUE types.String `tfsdk:"value"`
// 	TYPE  types.String `tfsdk:"type"`
// }

// type set_listResourceModel struct {
// 	NAME  types.String `tfsdk:"name"`
// 	VALUE types.String `tfsdk:"value"`
// }

// type set_sensitiveResourceModel struct {
// 	NAME  types.String `tfsdk:"name"`
// 	VALUE types.String `tfsdk:"value"`
// 	TYPE  types.String `tfsdk:"type"`
// }

// type postrender struct {
// 	BINARY_PATH types.String `tfsdk:"binary_path"`
// 	ARGS        types.List   `tfsdk:"args"`
// }

// type MetaRelease struct {
// 	data           *helmReleaseModel
// 	Settings       *cli.EnvSettings
// 	RegistryClient *registry.Client
// 	HelmDriver     string
// 	sync.Mutex
// }

// // passCredentialsDefault := boolDefaultAttributes["pass_credentials"]
// func (r *HelmReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
// 	/* ... */
// }

// func (r *HelmReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
// 	//passCredentialsDefault := boolAttributes["pass_credentials"]
// 	resp.Schema = schema.Schema{
// 		Description: "Schema to define attributes that are avaiavle in the resource",
// 		Attributes: map[string]schema.Attribute{
// 			"name": schema.StringAttribute{
// 				Required:    true,
// 				Description: "Release name. The length must not be longer than 53 characters",
// 			},
// 			"repository": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "Repository where to locate the requested chart. If is a URL the chart is installed without installing the repository",
// 			},
// 			"repository_key_file": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "The repositories cert key file",
// 			},
// 			"repository_cert_file": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "The repositories cert file",
// 			},
// 			"respository_ca_file": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "The Repositories CA file",
// 			},
// 			"repository_username": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "Username for HTTP basic authentication",
// 			},
// 			"repository_password": schema.StringAttribute{
// 				Optional:    true,
// 				Sensitive:   true,
// 				Description: "Password for HTTP basic authentication",
// 			},
// 			"pass_credentials": schema.BoolAttribute{
// 				Optional:    true,
// 				Description: "Pass credentials to all domains",
// 				//Default:     types.Bool(false),
// 			},
// 			"chart": schema.StringAttribute{
// 				Required:    true,
// 				Description: "Chart name to be installed. A path may be used,",
// 			},
// 			"version": schema.StringAttribute{
// 				Optional:    true,
// 				Computed:    true,
// 				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed ",
// 			},
// 			"devel": schema.BoolAttribute{
// 				Optional:    true,
// 				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If 'version' is set, this is ignored",
// 				// Suppress changes of this attribute if `version` is set
// 				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
// 				//return d.Get("version").(string) != ""
// 				//},
// 			},
// 			// TODO
// 			"values": schema.ListAttribute{
// 				Optional:    true,
// 				Description: "List of values in raw yamls format to pass to helm",
// 				//Elem:        &schema.Schema{Type: schema.TypeString},
// 			},
// 			// TODO
// 			"set": schema.ListNestedAttribute{
// 				Optional:    true,
// 				Description: "Custom values to be merged with the values",
// 				NestedObject: schema.NestedAttributeObject{
// 					Attributes: map[string]schema.Attribute{
// 						"name": schema.StringAttribute{
// 							Required: true,
// 						},
// 						"value": schema.StringAttribute{
// 							Required: true,
// 						},
// 						"type": schema.StringAttribute{
// 							Optional: true,
// 							//Default: "",
// 						},
// 					},
// 				},
// 			},
// 			// TODO
// 			"set_list": schema.ListNestedAttribute{
// 				Optional:    true,
// 				Description: "Custom sensitive values to be merged with the values",
// 				NestedObject: schema.NestedAttributeObject{
// 					Attributes: map[string]schema.Attribute{
// 						"name": schema.StringAttribute{
// 							Required: true,
// 						},
// 						"value": schema.ListAttribute{
// 							Required: true,
// 							//Elem:     &schema.Schema{Type: schema.TypeString},

// 						},
// 					},
// 				},
// 			},
// 			// TODO
// 			"set_sensitive": schema.ListNestedAttribute{
// 				Optional:    true,
// 				Description: "Custom sensitive values to be merged with the values",
// 				NestedObject: schema.NestedAttributeObject{
// 					Attributes: map[string]schema.Attribute{
// 						"name": schema.StringAttribute{
// 							Required: true,
// 						},
// 						"value": schema.StringAttribute{
// 							Required:  true,
// 							Sensitive: true,
// 						},
// 						// TODO
// 						"type": schema.StringAttribute{
// 							Optional: true,
// 						},
// 					},
// 				},
// 			},
// 			"namespace": schema.StringAttribute{
// 				Optional: true,
// 				//ForceNew: true,
// 				Description: "Namespace to install the release into",
// 				//DefaultFunc: schema.EnvDefaultFunc("HELM_NAMESPACE", "default"),
// 			},
// 			"verify": schema.BoolAttribute{
// 				Optional: true,
// 				//Default: defaultAttributes["verify"],
// 				Description: "Verify the package before installing it.",
// 			},
// 			"keyring": schema.StringAttribute{
// 				Optional: true,
// 				//Default:     os.ExpandEnv("$HOME/.gnupg/pubring.gpg"),
// 				Description: "Location of public keys used for verification, Used on if 'verify is true'",
// 				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
// 				//return !d.Get("verify").(bool)
// 				//},
// 			},
// 			"timeout": schema.Int64Attribute{
// 				Optional: true,
// 				//Default: defaultAttributes["timeouts"],
// 				Description: "Time in seconds to wait for any indvidiual kubernetes operation",
// 			},
// 			"disable_webhooks": schema.BoolAttribute{
// 				Optional: true,
// 				//Default: defaultAttributes["disbale_webhooks"],
// 				Description: "Prevent hooks from running",
// 			},
// 			"disable_crd_hooks": schema.BoolAttribute{
// 				Optional: true,
// 				//Default: defaultAttributes["disable_crd_hooks"],
// 				Description: "Prevent CRD hooks from, running, but run other hooks.  See helm install --no-crd-hook",
// 			},
// 			"reuse_values": schema.BoolAttribute{
// 				Optional:    true,
// 				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
// 				//Default: defaultAttributes["reuse_values"],
// 			},
// 			"reset_values": schema.BoolAttribute{
// 				Optional:    true,
// 				Description: "When upgrading, reset the values to the ones built into the chart",
// 				//Default:     defaultAttributes["reset_values"],
// 			},
// 			"force_update": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["force_update"],
// 				Description: "Force resource update through delete/recreate if needed.",
// 			},
// 			"recreate_pods": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["recreate_pods"],
// 				Description: "Perform pods restart during upgrade/rollback",
// 			},
// 			"cleanup_on_fail": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["cleanup_on_fail"],
// 				Description: "Allow deletion of new resources created in this upgrade when upgrade fails",
// 			},
// 			"max_history": schema.Int64Attribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["max_history"],
// 				Description: "Limit the maximum number of revisions saved per release. Use 0 for no limit",
// 			},
// 			"atomic": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["atomic"],
// 				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
// 			},
// 			"skip_crds": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["skip_crds"],
// 				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
// 			},
// 			"render_subchart_notes": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["render_subchart_notes"],
// 				Description: "If set, render subchart notes along with the parent",
// 			},
// 			"disable_openapi_validation": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["disable_openapi_validation"],
// 				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
// 			},
// 			"wait": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["wait"],
// 				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
// 			},
// 			"wait_for_jobs": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["wait_for_jobs"],
// 				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
// 			},
// 			"status": schema.StringAttribute{
// 				Computed:    true,
// 				Description: "Status of the release",
// 			},
// 			"dependency_update": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["dependency_update"],
// 				Description: "Run helm dependency update before installing the chart",
// 			},
// 			"replace": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["replace"],
// 				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
// 			},
// 			"description": schema.StringAttribute{
// 				Optional:    true,
// 				Description: "Add a custom description",
// 				//DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
// 				//return new == ""
// 				//},
// 			},
// 			"create_namespace": schema.BoolAttribute{
// 				Optional: true,
// 				//Default:     defaultAttributes["create_namespace"],
// 				Description: "Create the namespace if it does not exist",
// 			},
// 			// TODO
// 			"postrender": schema.ListNestedAttribute{
// 				//MaxItems: 1,
// 				Optional:    true,
// 				Description: "Postrender command config",
// 				NestedObject: schema.NestedAttributeObject{
// 					Attributes: map[string]schema.Attribute{
// 						"binary_path": schema.StringAttribute{
// 							Required:    true,
// 							Description: "The common binary path",
// 						},
// 						"args": schema.ListAttribute{
// 							Optional:    true,
// 							Description: "An argument to the post-renderer (can specify multiple)",
// 							//Elem:        &schema.Schema{Type: schema.TypeString},

// 						},
// 					},
// 				},
// 			},
// 			"lint": schema.BoolAttribute{
// 				Optional: true,
// 				//Default: defaultAttributes["lint"],
// 				Description: "Run helm lint when planning",
// 			},
// 			"manifest": schema.StringAttribute{
// 				Description: "The rendered manifest as 	JSON.",
// 				Computed:    true,
// 			},
// 			// TODO
// 			"metadata": schema.ListNestedAttribute{
// 				Computed:    true,
// 				Description: "Stats of the deployed release.",
// 				NestedObject: schema.NestedAttributeObject{
// 					Attributes: map[string]schema.Attribute{
// 						"name": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "Name is the name of the release",
// 						},
// 						"revision": schema.Int64Attribute{
// 							Computed:    true,
// 							Description: "Version is an int32 which represents the version of the release",
// 						},
// 						"namespace": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "Namespace is the kubernetes namespace of the release",
// 						},
// 						"chart": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "The name of the chart",
// 						},
// 						"version": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "A SemVer 2 conformant version string of the chart",
// 						},
// 						"app_version": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "The version number of the application being deployed",
// 						},
// 						"values": schema.StringAttribute{
// 							Computed:    true,
// 							Description: "Set of extra values. added to the chart. The sensitive data is cloaked. JSON encdoed.",
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }

// // Creating a new resource bassed off the schema data
// func (r *HelmReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
// 	/* ... */
// }

// func (r *HelmReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
// 	//var state helmReleaseModel

// }

// // Reading the state of the resource, this func will be called when the state refreshes by plan and apply
// func (r *HelmReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
// 	// State holds the current state of the resource
// 	var state helmReleaseModel

// 	// Reading the current state of the resource from state storage into the state variable we defined
// 	diags := req.State.Get(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Ensure meta is available and correctly typed
// 	meta := r.meta
// 	if meta == nil {
// 		resp.Diagnostics.AddError(
// 			"Meta not set",
// 			"The meta information is not set for the resource",
// 		)
// 		return
// 	}

// 	// Checks if the helm release exists, and will return a boolean indicating the existence of the helm release
// 	exists, diags := resourceReleaseExists(ctx, state.Name.ValueString(), state.NAMESPACE.ValueString(), meta)
// 	// If there is an error, the resource does not exist
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		if !exists {
// 			// if it does not exist, it removes it from the state, and returns here
// 			resp.State.RemoveResource(ctx)
// 		}
// 		return
// 	}

// 	logID := fmt.Sprintf("[resourceReleaseRead: %s]", state.Name.ValueString())
// 	tflog.Debug(ctx, fmt.Sprintf("%s Started", logID))

// 	// Call GetHelmConfiguration obtaining helm config for the specified namespace
// 	c, err := meta.GetHelmConfiguration(ctx, state.NAMESPACE.ValueString())
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"Error getting Helm configuration",
// 			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", state.NAMESPACE.ValueString(), err),
// 		)
// 		return
// 	}

// 	// Call getRelease to retrieve the helm release by using the configuration acquired from GetHelmConfiguration
// 	release, err := getRelease(ctx, meta, c, state.Name.ValueString())
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"Error getting release",
// 			fmt.Sprintf("Unable to get Helm release %s: %s", state.Name.ValueString(), err),
// 		)
// 		return
// 	}

// 	// Calls setReleaseAttributes updating the state variable with attributes of the helm release
// 	diags = setReleaseAttributes(ctx, &state, release, meta)
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		resp.Diagnostics.AddError(
// 			"Error setting release attributes",
// 			fmt.Sprintf("Unable to set attributes for helm release %s", state.Name.ValueString()),
// 		)
// 		return
// 	}

// 	tflog.Debug(ctx, fmt.Sprintf("%s Done", logID))

// 	// Updating the state with the new state, which reflects the current state of the helm release
// 	diags = resp.State.Set(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// }

// func (r *HelmReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
// 	// State holds the current state of the resource
// 	var state helmReleaseModel
// 	diags := req.State.Get(ctx, &state)
// 	resp.Diagnostics.Append(diags...)
// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Checking if the meta field is available
// 	meta := r.meta
// 	if meta == nil {
// 		resp.Diagnostics.AddError(
// 			"Meta not set",
// 			"The meta information is not set for the resource",
// 		)
// 		return
// 	}
// 	// gets the namespace
// 	namespace := state.NAMESPACE.ValueString()
// 	// calls GetHelmConfiguration to obtain the Helm config for the namespace
// 	actionConfig, err := meta.GetHelmConfiguration(ctx, namespace)
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"Error getting helm configuration",
// 			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
// 		)
// 		return
// 	}

// 	name := state.Name.ValueString()
// 	// Create a uninstall action with the config obtained
// 	uninstall := action.NewUninstall(actionConfig)
// 	uninstall.Wait = state.WAIT.ValueBool()
// 	uninstall.DisableHooks = state.DISABLE_WEBHOOKS.ValueBool()
// 	uninstall.Timeout = time.Duration(state.TIMEOUT.ValueInt64()) * time.Second
// 	// Running the uninstall action on the helm release
// 	res, err := uninstall.Run(name)
// 	if err != nil {
// 		resp.Diagnostics.AddError(
// 			"Error uninstalling release",
// 			fmt.Sprintf("Unable to uninstall Helm release %s: %s", name, err),
// 		)
// 		return
// 	}
// 	// Checks if the uninstall action returned a message
// 	if res.Info != "" {
// 		resp.Diagnostics.Append(diag.NewWarningDiagnostic(
// 			"Helm uninstall returned an information message",
// 			res.Info,
// 		))
// 	}
// 	// Removing the resourcer from the state.
// 	resp.State.RemoveResource(ctx)
// }

// func resourceReleaseExists(ctx context.Context, name, namespace string, meta *MetaRelease) (bool, diag.Diagnostics) {
// 	logID := fmt.Sprintf("[resourceReleaseExists: %s]", name)
// 	tflog.Debug(ctx, fmt.Sprintf("%s Start", logID))

// 	var diags diag.Diagnostics

// 	c, err := meta.GetHelmConfiguration(ctx, namespace)
// 	if err != nil {
// 		diags.AddError(
// 			"Error getting helm configuration",
// 			fmt.Sprintf("Unable to get Helm configuration for namespace %s: %s", namespace, err),
// 		)
// 		return false, diags
// 	}

// 	_, err = getRelease(ctx, meta, c, name)

// 	tflog.Debug(ctx, fmt.Sprintf("%s Done", logID))

// 	if err == nil {
// 		return true, diags
// 	}

// 	if err == errReleaseNotFound {
// 		return false, diags
// 	}

// 	diags.AddError(
// 		"Error checking release existence",
// 		fmt.Sprintf("Error checking release %s in namespace %s: %s", name, namespace, err),
// 	)
// 	return false, diags
// }

// // This function will return a action.Configuration obj, which will help manage helm operations
// func (m *MetaRelease) GetHelmConfiguration(ctx context.Context, namespace string) (*action.Configuration, error) {
// 	// Locking the meta struct
// 	m.Lock()
// 	defer m.Unlock()

// 	tflog.Debug(ctx, "[INFO] GetHelmConfiguration start")
// 	// Initalizing actionConfig, which will be used to installing, upgrading or uninstalling helm charts
// 	actionConfig := new(action.Configuration)

// 	// Calls the newKubeConfig, to create a k8s configuration.
// 	kc, err := newKubeConfig(m.data, &namespace)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// initalizes the action.Config with the k8s configuration.
// 	// Using an inline closure to adapt tflog.Debug. Converting the arguments of action.DebugLog into a map tflog.debug can accept.
// 	if err := actionConfig.Init(kc, namespace, m.HelmDriver, func(format string, v ...interface{}) {
// 		tflog.Debug(ctx, format, map[string]interface{}{"args": v})
// 	}); err != nil {
// 		return nil, err
// 	}
// 	tflog.Debug(ctx, "[INFO] GetHelmConfiguration start")
// 	// grabs the action.Config object
// 	return actionConfig, nil
// }

// var errReleaseNotFound = fmt.Errorf("release: not found")

// // Retrieves a helm release by the kluster name, also using the helm configuration
// func getRelease(ctx context.Context, m *MetaRelease, cfg *action.Configuration, name string) (*release.Release, error) {
// 	tflog.Debug(ctx, fmt.Sprintf("%s getRelease wait for lock", name))
// 	m.Lock()
// 	defer m.Unlock()
// 	tflog.Debug(ctx, fmt.Sprintf("%s getRelease got lock, started", name))

// 	get := action.NewGet(cfg)
// 	tflog.Debug(ctx, fmt.Sprintf("%s getRelease post action created", name))

// 	res, err := get.Run(name)
// 	tflog.Debug(ctx, fmt.Sprintf("%s getRelease post run", name))

// 	if err != nil {
// 		tflog.Debug(ctx, fmt.Sprintf("getRelease for %s occurred", name))
// 		tflog.Debug(ctx, fmt.Sprintf("%v", err))
// 		if strings.Contains(err.Error(), "release: not found") {
// 			tflog.Error(ctx, errReleaseNotFound.Error())
// 			return nil, errReleaseNotFound
// 		}
// 		tflog.Debug(ctx, fmt.Sprintf("Could not get release %s", err))
// 		tflog.Error(ctx, err.Error())
// 		return nil, err
// 	}

// 	tflog.Debug(ctx, fmt.Sprintf("%s getRelease completed", name))
// 	return res, nil
// }

// // Ensures the state reflects the current state of the helm release in the k8s cluster, this function will be called in the read to update the state with the current attr of the helm release when terraform refreshed the state
// // Alse used in the create. To set the state attributes after a helm release is created
// func setReleaseAttributes(ctx context.Context, state *helmReleaseModel, r *release.Release, meta *MetaRelease) diag.Diagnostics {
// 	var diags diag.Diagnostics

// 	// Update state with attributes from the helm release
// 	state.Name = types.StringValue(r.Name)
// 	state.VERSION = types.StringValue(r.Chart.Metadata.Version)
// 	state.NAMESPACE = types.StringValue(r.Namespace)
// 	state.STATUS = types.StringValue(r.Info.Status.String())

// 	// Calling cloakSetValues to handle sensitive values in the release config.
// 	cloakSetValues(r.Config, state)
// 	values := "{}"
// 	if r.Config != nil {
// 		v, err := json.Marshal(r.Config)
// 		if err != nil {
// 			diags.AddError(
// 				"Error marshaling values",
// 				fmt.Sprintf("unable to marshal values: %s", err),
// 			)
// 			return diags
// 		}
// 		// Storing the JSON string to values
// 		values = string(v)
// 	}

// 	// Handling the helm release if manifest experiment is enabled
// 	if meta.ExperimentEnabled("manifest") {
// 		jsonManifest, err := convertYAMLManifestToJSON(r.Manifest)
// 		if err != nil {
// 			diags.AddError(
// 				"Error converting manifest to JSON",
// 				fmt.Sprintf("Unable to convert manifest to JSON: %s", err),
// 			)
// 			return diags
// 		}
// 		manifest := redactSensitiveValues(string(jsonManifest), state)
// 		state.MANIFEST = types.StringValue(manifest)
// 	}

// 	// Creating a map for the metadata regarding the helm release.
// 	metadata := []map[string]interface{}{
// 		{
// 			"name":        r.Name,
// 			"revision":    r.Version,
// 			"namespace":   r.Namespace,
// 			"chart":       r.Chart.Metadata.Name,
// 			"version":     r.Chart.Metadata.Version,
// 			"app_version": r.Chart.Metadata.AppVersion,
// 			"values":      values,
// 		},
// 	}

// 	// Updating the state.METADATA with the metadata map
// 	listValue, err := types.ListValueFrom(ctx, types.ObjectType{
// 		AttrTypes: map[string]attr.Type{
// 			"name":        types.StringType,
// 			"revision":    types.Int64Type,
// 			"namespace":   types.StringType,
// 			"chart":       types.StringType,
// 			"version":     types.StringType,
// 			"app_version": types.StringType,
// 			"values":      types.StringType,
// 		},
// 	}, metadata)
// 	if err != nil {
// 		diags.AddError(
// 			"Error setting metadata",
// 			fmt.Sprintf("Unable to set metadata: %s", err),
// 		)
// 		return diags
// 	}

// 	state.METADATA = listValue

// 	return diags
// }

// const sensitiveContentValue = "(sensitive value)"

// // Iterates over the SET_SENSITIVE attribute. For each item, it call cloackSetValue with config map and name of sens value
// func cloakSetValues(config map[string]interface{}, state *helmReleaseModel) {
// 	for _, set := range state.SET_SENSITIVE {
// 		cloakSetValue(config, set.NAME.ValueString())
// 	}
// }

// // value path = hierarchical path to the sens value in the map. This func replaces a sens value with place holder value
// func cloakSetValue(values map[string]interface{}, valuePath string) {
// 	// Splitting the value path to 2 keys
// 	pathKeys := strings.Split(valuePath, ".")
// 	// Extracting the last key from pathKeys, which will be the key for the sens value
// 	sensitiveKey := pathKeys[len(pathKeys)-1]
// 	//Extract all keys, but the last one. These keys are the path to the parent map which contains the sens key
// 	parentPathKeys := pathKeys[:len(pathKeys)-1]
// 	// temp var, to the root map. Helps us traverse through the nested maps
// 	m := values
// 	// navigating to the parent map, that contains the sens key
// 	for _, key := range parentPathKeys {
// 		// for each key, we attempt to access the nested map m
// 		v, ok := m[key].(map[string]interface{})
// 		if !ok {
// 			return
// 		}
// 		// Updating m to point to the nested map
// 		m = v
// 	}
// 	// Setting sensitiveKey in the final map to the placeholder
// 	m[sensitiveKey] = sensitiveContentValue
// }
