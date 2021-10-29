package helm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/hashicorp/terraform-provider-helm/helm/manifest"
)

// defaultTemplateAttributes template attribute values
var defaultTemplateAttributes = map[string]interface{}{
	"validate":     false,
	"include_crds": false,
	"is_upgrade":   false,
	"skip_tests":   false,
}

func dataTemplate() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataTemplateRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Release name.",
			},
			"repository": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Repository where to locate the requested chart. If is a URL the chart is installed without installing the repository.",
			},
			"repository_key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The repositories cert key file",
			},
			"repository_cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The repositories cert file",
			},
			"repository_ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Repositories CA File",
			},
			"repository_username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username for HTTP basic authentication",
			},
			"repository_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password for HTTP basic authentication",
			},
			"chart": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Chart name to be installed. A path may be used.",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to install. If this is not specified, the latest version is installed.",
			},
			"devel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored",
				// Suppress changes of this attribute if `version` is set
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Get("version").(string) != ""
				},
			},
			"values": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of values in raw yaml format to pass to helm.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"set": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom values to be merged with the values.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							// TODO: use ValidateDiagFunc once an SDK v2 version of StringInSlice exists.
							// https://github.com/hashicorp/terraform-plugin-sdk/issues/534
							ValidateFunc: validation.StringInSlice([]string{
								"auto", "string",
							}, false),
						},
					},
				},
			},
			"set_sensitive": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom sensitive values to be merged with the values.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								"auto", "string",
							}, false),
						},
					},
				},
			},
			"set_string": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom string values to be merged with the values.",
				Deprecated: "This argument is deprecated and will be removed in the next major" +
					" version. Use `set` argument with `type` equals to `string`",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Namespace to install the release into.",
				DefaultFunc: schema.EnvDefaultFunc("HELM_NAMESPACE", "default"),
			},
			"verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["verify"],
				Description: "Verify the package before installing it.",
			},
			"keyring": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     os.ExpandEnv("$HOME/.gnupg/pubring.gpg"),
				Description: "Location of public keys used for verification. Used only if `verify` is true",
				// Suppress changes of this attribute if `verify` is false
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return !d.Get("verify").(bool)
				},
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     defaultAttributes["timeout"],
				Description: "Time in seconds to wait for any individual kubernetes operation.",
			},
			"disable_webhooks": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["disable_webhooks"],
				Description: "Prevent hooks from running.",
			},
			"reuse_values": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When upgrading, reuse the last release's values and merge in any overrides. If 'reset_values' is specified, this is ignored",
				Default:     defaultAttributes["reuse_values"],
			},
			"reset_values": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When upgrading, reset the values to the ones built into the chart",
				Default:     defaultAttributes["reset_values"],
			},
			"atomic": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["atomic"],
				Description: "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used",
			},
			"skip_crds": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["skip_crds"],
				Description: "If set, no CRDs will be installed. By default, CRDs are installed if not already present",
			},
			"skip_tests": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["skip_tests"],
				Description: "If set, tests will not be rendered. By default, tests are rendered",
			},
			"render_subchart_notes": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["render_subchart_notes"],
				Description: "If set, render subchart notes along with the parent",
			},
			"disable_openapi_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["disable_openapi_validation"],
				Description: "If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema",
			},
			"wait": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["wait"],
				Description: "Will wait until all resources are in a ready state before marking the release as successful.",
			},
			"dependency_update": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["dependency_update"],
				Description: "Run helm dependency update before installing the chart",
			},
			"replace": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["replace"],
				Description: "Re-use the given name, even if that name is already used. This is unsafe in production",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Add a custom description",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == ""
				},
			},
			"create_namespace": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["create_namespace"],
				Description: "Create the namespace if it does not exist",
			},
			"postrender": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Postrender command configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"binary_path": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The command binary path.",
						},
					},
				},
			},
			"api_versions": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Kubernetes api versions used for Capabilities.APIVersions",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"include_crds": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultTemplateAttributes["include_crds"],
				Description: "Include CRDs in the templated output",
			},
			"is_upgrade": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultTemplateAttributes["is_upgrade"],
				Description: "Set .Release.IsUpgrade instead of .Release.IsInstall",
			},
			"show_only": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Only show manifests rendered from the given templates",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"validate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultTemplateAttributes["validate"],
				Description: "Validate your manifests against the Kubernetes cluster you are currently pointing at. This is the same validation performed on an install",
			},
			"manifests": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "Map of rendered chart templates indexed by the template name.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"manifest": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.",
			},
			"crds": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "Map of the charts crds.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"crd": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Concatenated chart crds.",
			},
			"notes": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Rendered notes if the chart contains a `NOTES.txt`.",
			},
		},
	}
}

func dataTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	logID := fmt.Sprintf("[dataTemplateRead: %s]", d.Get("name").(string))
	debug("%s Started", logID)

	m := meta.(*Meta)

	name := d.Get("name").(string)
	n := d.Get("namespace").(string)

	var apiVersions []string

	if apiVersionsAttr, ok := d.GetOk("api_versions"); ok {
		apiVersionsValues := apiVersionsAttr.([]interface{})

		for _, apiVersion := range apiVersionsValues {
			apiVersions = append(apiVersions, apiVersion.(string))
		}
	}

	var showFiles []string
	if showOnlyAttr, ok := d.GetOk("show_only"); ok {
		showOnlyAttrValue := showOnlyAttr.([]interface{})

		for _, showFile := range showOnlyAttrValue {
			showFiles = append(showFiles, showFile.(string))
		}
	}

	debug("%s Getting Config", logID)

	actionConfig, err := m.GetHelmConfiguration(n)
	if err != nil {
		return diag.FromErr(err)
	}

	cpo, chartName, err := chartPathOptions(d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	debug("%s Getting chart", logID)
	c, path, err := getChart(d, m, chartName, cpo)
	if err != nil {
		return diag.FromErr(err)
	}

	// check and update the chart's dependencies if needed
	updated, err := checkChartDependencies(d, c, path, m)
	if err != nil {
		return diag.FromErr(err)
	} else if updated {
		// load the chart again if its dependencies have been updated
		c, err = loader.Load(path)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	debug("%s Preparing for installation", logID)

	values, err := getValues(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = isChartInstallable(c)
	if err != nil {
		return diag.FromErr(err)
	}

	client := action.NewInstall(actionConfig)
	client.ChartPathOptions = *cpo
	client.ClientOnly = false
	client.DryRun = true
	client.DisableHooks = d.Get("disable_webhooks").(bool)
	client.Wait = d.Get("wait").(bool)
	client.Devel = d.Get("devel").(bool)
	client.DependencyUpdate = d.Get("dependency_update").(bool)
	client.Timeout = time.Duration(d.Get("timeout").(int)) * time.Second
	client.Namespace = d.Get("namespace").(string)
	client.ReleaseName = d.Get("name").(string)
	client.GenerateName = false
	client.NameTemplate = ""
	client.OutputDir = ""
	client.Atomic = d.Get("atomic").(bool)
	client.SkipCRDs = d.Get("skip_crds").(bool)
	client.SubNotes = d.Get("render_subchart_notes").(bool)
	client.DisableOpenAPIValidation = d.Get("disable_openapi_validation").(bool)
	client.Replace = d.Get("replace").(bool)
	client.Description = d.Get("description").(string)
	client.CreateNamespace = d.Get("create_namespace").(bool)

	// The following source has been adapted from the source of the helm template command
	// https://github.com/helm/helm/blob/v3.5.3/cmd/helm/template.go#L67
	client.DryRun = true
	// NOTE Do not set fixed release name as client.ReleaseName like in helm template command
	client.Replace = true // Skip the name check
	client.ClientOnly = !d.Get("validate").(bool)
	client.APIVersions = chartutil.VersionSet(apiVersions)

	includeCrds := d.Get("include_crds").(bool)
	skipTests := d.Get("skip_tests").(bool)

	debug("%s Rendering Chart", logID)

	rel, err := client.Run(c, values)
	if err != nil {
		return diag.FromErr(err)
	}

	// Collects the result of Helm template
	templateBuilder := strings.Builder{}

	// Add CRDs to manifest output if enabled
	// The reason this is done outside of the client is due
	// to how Helm handles multidoc yaml in the charts crds directory.
	// This solution mankes the behavior similar to yaml from the templates
	// directory.
	templateCrd := manifest.CrdToManifest(c.CRDObjects())
	if includeCrds {
		templateBuilder.WriteString(templateCrd)
	}

	// Add hook manifests if enabled.
	if !client.DisableHooks {
		templatedHook := manifest.HooksToManifest(rel.Hooks, skipTests)
		templateBuilder.WriteString(templatedHook)
	}

	// Add rendered manifests.
	templateBuilder.WriteString(strings.TrimSpace(rel.Manifest))

	// Compute output for all manifests and crd manifests
	allManifests, allManifest, err := manifest.Compute(templateBuilder.String(), showFiles)
	if err != nil {
		return diag.FromErr(err)
	}
	crdManifests, crdManifest, err := manifest.Compute(templateCrd, []string{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)
	err = d.Set("manifests", allManifests)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("manifest", allManifest)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("crds", crdManifests)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("crd", crdManifest)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("notes", rel.Info.Notes)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
