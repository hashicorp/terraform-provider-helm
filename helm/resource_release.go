// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/postrender"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"
)

// errReleaseNotFound is the error when a Helm release is not found
var errReleaseNotFound = errors.New("release not found")

// defaultAttributes release attribute values
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

func resourceRelease() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceReleaseCreate,
		ReadContext:   resourceReleaseRead,
		DeleteContext: resourceReleaseDelete,
		UpdateContext: resourceReleaseUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: resourceHelmReleaseImportState,
		},
		CustomizeDiff: resourceDiff,
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
			"pass_credentials": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Pass credentials to all domains",
				Default:     defaultAttributes["pass_credentials"],
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
							Default:  "",
							// TODO: use ValidateDiagFunc once an SDK v2 version of StringInSlice exists.
							// https://github.com/hashicorp/terraform-plugin-sdk/issues/534
							ValidateFunc: validation.StringInSlice([]string{
								"auto", "string",
							}, false),
						},
					},
				},
			},
			"set_list": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Custom sensitive values to be merged with the values.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
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
			"disable_crd_hooks": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["disable_crd_hooks"],
				Description: "Prevent CRD hooks from, running, but run other hooks.  See helm install --no-crd-hook",
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
			"force_update": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["force_update"],
				Description: "Force resource update through delete/recreate if needed.",
			},
			"recreate_pods": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["recreate_pods"],
				Description: "Perform pods restart during upgrade/rollback",
			},
			"cleanup_on_fail": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["cleanup_on_fail"],
				Description: "Allow deletion of new resources created in this upgrade when upgrade fails",
			},
			"max_history": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     defaultAttributes["max_history"],
				Description: "Limit the maximum number of revisions saved per release. Use 0 for no limit",
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
			"wait_for_jobs": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["wait_for_jobs"],
				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the release.",
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
						"args": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "an argument to the post-renderer (can specify multiple)",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"lint": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["lint"],
				Description: "Run helm lint when planning",
			},
			"manifest": {
				Type:        schema.TypeString,
				Description: "The rendered manifest as JSON.",
				Computed:    true,
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Status of the deployed release.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name is the name of the release.",
						},
						"revision": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Version is an int32 which represents the version of the release.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Namespace is the kubernetes namespace of the release.",
						},
						"chart": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the chart.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A SemVer 2 conformant version string of the chart.",
						},
						"app_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version number of the application being deployed.",
						},
						"values": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Set of extra values, added to the chart. The sensitive data is cloaked. JSON encoded.",
						},
					},
				},
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceReleaseUpgrader().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceReleaseStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func resourceReleaseStateUpgradeV0(ctx context.Context, rawState map[string]any, meta any) (map[string]any, error) {
	if rawState["pass_credentials"] == nil {
		rawState["pass_credentials"] = false
	}
	if rawState["wait_for_jobs"] == nil {
		rawState["wait_for_jobs"] = false
	}
	return rawState, nil
}

func resourceReleaseUpgrader() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"pass_credentials": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Pass credentials to all domains",
				Default:     defaultAttributes["pass_credentials"],
			},
			"wait_for_jobs": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     defaultAttributes["wait_for_jobs"],
				Description: "If wait is enabled, will wait until all Jobs have been completed before marking the release as successful.",
			},
		},
	}
}

func resourceReleaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceReleaseExists(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}

	logID := fmt.Sprintf("[resourceReleaseRead: %s]", d.Get("name").(string))
	debug("%s Started", logID)

	m := meta.(*Meta)
	n := d.Get("namespace").(string)

	c, err := m.GetHelmConfiguration(n)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	r, err := getRelease(m, c, name)
	if err != nil {
		return diag.FromErr(err)
	}

	err = setReleaseAttributes(d, r, m)
	if err != nil {
		return diag.FromErr(err)
	}

	debug("%s Done", logID)

	return nil
}

func checkChartDependencies(d resourceGetter, c *chart.Chart, path string, m *Meta) (bool, error) {
	p := getter.All(m.Settings)

	if req := c.Metadata.Dependencies; req != nil {
		err := action.CheckDependencies(c, req)
		if err != nil {
			if d.Get("dependency_update").(bool) {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          d.Get("keyring").(string),
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: m.Settings.RepositoryConfig,
					RepositoryCache:  m.Settings.RepositoryCache,
					Debug:            m.Settings.Debug,
				}
				log.Println("[DEBUG] Downloading chart dependencies...")
				return true, man.Update()
			}
			return false, err
		}
		return false, err
	}
	log.Println("[DEBUG] Chart dependencies are up to date.")
	return false, nil
}

func resourceReleaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	logID := fmt.Sprintf("[resourceReleaseCreate: %s]", d.Get("name").(string))
	debug("%s Started", logID)

	m := meta.(*Meta)
	n := d.Get("namespace").(string)

	debug("%s Getting helm configuration", logID)
	actionConfig, err := m.GetHelmConfiguration(n)
	if err != nil {
		return diag.FromErr(err)
	}
	err = OCIRegistryLogin(actionConfig, d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	client := action.NewInstall(actionConfig)

	cpo, chartName, err := chartPathOptions(d, m, &client.ChartPathOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	debug("%s Getting chart", logID)
	c, path, err := getChart(d, m, chartName, cpo)
	if err != nil {
		return diag.FromErr(fmt.Errorf("could not download chart: %v", err))
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

	client.ClientOnly = false
	client.DryRun = false
	client.DisableHooks = d.Get("disable_webhooks").(bool)
	client.Wait = d.Get("wait").(bool)
	client.WaitForJobs = d.Get("wait_for_jobs").(bool)
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

	if cmd := d.Get("postrender.0.binary_path").(string); cmd != "" {
		av := d.Get("postrender.0.args")
		var args []string
		for _, arg := range av.([]interface{}) {
			if arg == nil {
				continue
			}
			args = append(args, arg.(string))
		}

		pr, err := postrender.NewExec(cmd, args...)

		if err != nil {
			return diag.FromErr(err)
		}

		client.PostRenderer = pr
	}

	debug("%s Installing chart", logID)

	rel, err := client.Run(c, values)

	if err != nil && rel == nil {
		return diag.FromErr(err)
	}

	if err != nil && rel != nil {
		exists, existsErr := resourceReleaseExists(d, meta)

		if existsErr != nil {
			return diag.FromErr(existsErr)
		}

		if !exists {
			return diag.FromErr(err)
		}

		debug("%s Release was created but returned an error", logID)

		if err := setReleaseAttributes(d, rel, m); err != nil {
			return diag.FromErr(err)
		}

		return diag.Diagnostics{
			{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Helm release %q was created but has a failed status. Use the `helm` command to investigate the error, correct it, then run Terraform again.", client.ReleaseName),
			},
			{
				Severity: diag.Error,
				Summary:  err.Error(),
			},
		}

	}

	err = setReleaseAttributes(d, rel, m)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceReleaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	m := meta.(*Meta)
	n := d.Get("namespace").(string)
	actionConfig, err := m.GetHelmConfiguration(n)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}
	err = OCIRegistryLogin(actionConfig, d, m)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}
	client := action.NewUpgrade(actionConfig)

	cpo, chartName, err := chartPathOptions(d, m, &client.ChartPathOptions)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}

	c, path, err := getChart(d, m, chartName, cpo)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}

	// check and update the chart's dependencies if needed
	updated, err := checkChartDependencies(d, c, path, m)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	} else if updated {
		// load the chart again if its dependencies have been updated
		c, err = loader.Load(path)
		if err != nil {
			d.Partial(true)
			return diag.FromErr(err)
		}
	}

	client.Devel = d.Get("devel").(bool)
	client.Namespace = d.Get("namespace").(string)
	client.Timeout = time.Duration(d.Get("timeout").(int)) * time.Second
	client.Wait = d.Get("wait").(bool)
	client.WaitForJobs = d.Get("wait_for_jobs").(bool)
	client.DryRun = false
	client.DisableHooks = d.Get("disable_webhooks").(bool)
	client.Atomic = d.Get("atomic").(bool)
	client.SkipCRDs = d.Get("skip_crds").(bool)
	client.SubNotes = d.Get("render_subchart_notes").(bool)
	client.DisableOpenAPIValidation = d.Get("disable_openapi_validation").(bool)
	client.Force = d.Get("force_update").(bool)
	client.ResetValues = d.Get("reset_values").(bool)
	client.ReuseValues = d.Get("reuse_values").(bool)
	client.Recreate = d.Get("recreate_pods").(bool)
	client.MaxHistory = d.Get("max_history").(int)
	client.CleanupOnFail = d.Get("cleanup_on_fail").(bool)
	client.Description = d.Get("description").(string)

	if cmd := d.Get("postrender.0.binary_path").(string); cmd != "" {
		av := d.Get("postrender.0.args")
		var args []string
		for _, arg := range av.([]interface{}) {
			if arg == nil {
				continue
			}
			args = append(args, arg.(string))
		}

		pr, err := postrender.NewExec(cmd, args...)

		if err != nil {
			d.Partial(true)
			return diag.FromErr(err)
		}

		client.PostRenderer = pr
	}

	values, err := getValues(d)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	r, err := client.Run(name, c, values)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}

	err = setReleaseAttributes(d, r, m)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceReleaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	m := meta.(*Meta)
	n := d.Get("namespace").(string)
	actionConfig, err := m.GetHelmConfiguration(n)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = d.Get("wait").(bool)
	uninstall.DisableHooks = d.Get("disable_webhooks").(bool)
	uninstall.Timeout = time.Duration(d.Get("timeout").(int)) * time.Second

	res, err := uninstall.Run(name)
	if err != nil {
		return diag.FromErr(err)
	}

	if res.Info != "" {
		return diag.Diagnostics{
			{
				Severity: diag.Warning,
				Summary:  "Helm uninstall returned an information message",
				Detail:   res.Info,
			},
		}
	}

	d.SetId("")
	return nil
}

func resourceDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	logID := fmt.Sprintf("[resourceDiff: %s]", d.Get("name").(string))
	debug("%s Start", logID)
	m := meta.(*Meta)
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

	actionConfig, err := m.GetHelmConfiguration(namespace)
	if err != nil {
		return err
	}
	err = OCIRegistryLogin(actionConfig, d, m)
	if err != nil {
		return err
	}

	// Always set desired state to DEPLOYED
	err = d.SetNew("status", release.StatusDeployed.String())
	if err != nil {
		return err
	}

	// Always recompute metadata if a new revision is going to be created
	recomputeMetadataFields := []string{
		"chart",
		"repository",
		"values",
		"set",
		"set_sensitive",
		"set_list",
	}
	if d.HasChanges(recomputeMetadataFields...) {
		d.SetNewComputed("metadata")
	}

	if !useChartVersion(d.Get("chart").(string), d.Get("repository").(string)) {
		if d.HasChange("version") {
			// only recompute metadata if the version actually changes
			// chart versioning is not consistent and some will add
			// a `v` prefix to the chart version after installation
			old, new := d.GetChange("version")
			oldVersion := strings.TrimPrefix(old.(string), "v")
			newVersion := strings.TrimPrefix(new.(string), "v")
			if oldVersion != newVersion {
				d.SetNewComputed("metadata")
			}
		}
	}

	var chartPathOpts action.ChartPathOptions
	cpo, chartName, err := chartPathOptions(d, m, &chartPathOpts)
	if err != nil {
		return err
	}

	// Get Chart metadata, if we fail - we're done
	chart, path, err := getChart(d, meta.(*Meta), chartName, cpo)
	if err != nil {
		return nil
	}
	debug("%s Got chart", logID)

	// check and update the chart's dependencies if needed
	updated, err := checkChartDependencies(d, chart, path, m)
	if err != nil {
		return err
	} else if updated {
		// load the chart again if its dependencies have been updated
		chart, err = loader.Load(path)
		if err != nil {
			return err
		}
	}

	// Validates the resource configuration, the values, the chart itself, and
	// the combination of both.
	//
	// Maybe here is not the most canonical place to include a validation
	// but is the only place to fail in `terraform plan`.
	if d.Get("lint").(bool) {
		if err := resourceReleaseValidate(d, meta.(*Meta), cpo); err != nil {
			return err
		}
	}
	debug("%s Release validated", logID)

	if m.ExperimentEnabled("manifest") {
		// NOTE we need to check that the values supplied to the release are
		// fully known at plan time otherwise we can't supply them to the
		// action to perform a dry run
		if !valuesKnown(d) {
			// NOTE it would be nice to surface a warning diagnostic here
			// but this is not possible with the SDK
			debug("not all values are known, skipping dry run to render manifest")
			d.SetNewComputed("manifest")
			return d.SetNewComputed("version")
		}

		var postRenderer postrender.PostRenderer
		if cmd := d.Get("postrender.0.binary_path").(string); cmd != "" {
			av := d.Get("postrender.0.args")
			args := []string{}
			for _, arg := range av.([]interface{}) {
				if arg == nil {
					continue
				}
				args = append(args, arg.(string))
			}
			pr, err := postrender.NewExec(cmd, args...)
			if err != nil {
				return err
			}
			postRenderer = pr
		}

		oldStatus, _ := d.GetChange("status")
		if oldStatus.(string) == "" {
			install := action.NewInstall(actionConfig)
			install.ChartPathOptions = *cpo
			install.DryRun = true
			install.DisableHooks = d.Get("disable_webhooks").(bool)
			install.Wait = d.Get("wait").(bool)
			install.WaitForJobs = d.Get("wait_for_jobs").(bool)
			install.Devel = d.Get("devel").(bool)
			install.DependencyUpdate = d.Get("dependency_update").(bool)
			install.Timeout = time.Duration(d.Get("timeout").(int)) * time.Second
			install.Namespace = d.Get("namespace").(string)
			install.ReleaseName = d.Get("name").(string)
			install.Atomic = d.Get("atomic").(bool)
			install.SkipCRDs = d.Get("skip_crds").(bool)
			install.SubNotes = d.Get("render_subchart_notes").(bool)
			install.DisableOpenAPIValidation = d.Get("disable_openapi_validation").(bool)
			install.Replace = d.Get("replace").(bool)
			install.Description = d.Get("description").(string)
			install.CreateNamespace = d.Get("create_namespace").(bool)
			install.PostRenderer = postRenderer

			values, err := getValues(d)
			if err != nil {
				return fmt.Errorf("error getting values: %v", err)
			}

			debug("%s performing dry run install", logID)
			dry, err := install.Run(chart, values)
			if err != nil {
				// NOTE if the cluster is not reachable then we can't run the install
				// this will happen if the user has their cluster creation in the
				// same apply. We are catching this case here and marking manifest
				// as computed to avoid breaking existing configs
				if strings.Contains(err.Error(), "Kubernetes cluster unreachable") {
					// NOTE it would be nice to return a diagnostic here to warn the user
					// that we can't generate the diff here because the cluster is not yet
					// reachable but this is not supported by CustomizeDiffFunc
					debug(`cluster was unreachable at create time, marking "manifest" as computed`)
					return d.SetNewComputed("manifest")
				}
				return err
			}

			jsonManifest, err := convertYAMLManifestToJSON(dry.Manifest)
			if err != nil {
				return err
			}
			manifest := redactSensitiveValues(string(jsonManifest), d)
			return d.SetNew("manifest", manifest)
		}

		// check if release exists
		_, err = getRelease(m, actionConfig, name)
		if err == errReleaseNotFound {
			if len(chart.Metadata.Version) > 0 {
				return d.SetNew("version", chart.Metadata.Version)
			}
			d.SetNewComputed("manifest")
			return d.SetNewComputed("version")
		} else if err != nil {
			return fmt.Errorf("error retrieving old release for a diff: %v", err)
		}

		upgrade := action.NewUpgrade(actionConfig)
		upgrade.ChartPathOptions = *cpo
		upgrade.Devel = d.Get("devel").(bool)
		upgrade.Namespace = d.Get("namespace").(string)
		upgrade.Timeout = time.Duration(d.Get("timeout").(int)) * time.Second
		upgrade.Wait = d.Get("wait").(bool)
		upgrade.DryRun = true // do not apply changes
		upgrade.DisableHooks = d.Get("disable_webhooks").(bool)
		upgrade.Atomic = d.Get("atomic").(bool)
		upgrade.SubNotes = d.Get("render_subchart_notes").(bool)
		upgrade.WaitForJobs = d.Get("wait_for_jobs").(bool)
		upgrade.Force = d.Get("force_update").(bool)
		upgrade.ResetValues = d.Get("reset_values").(bool)
		upgrade.ReuseValues = d.Get("reuse_values").(bool)
		upgrade.Recreate = d.Get("recreate_pods").(bool)
		upgrade.MaxHistory = d.Get("max_history").(int)
		upgrade.CleanupOnFail = d.Get("cleanup_on_fail").(bool)
		upgrade.Description = d.Get("description").(string)
		upgrade.PostRenderer = postRenderer

		values, err := getValues(d)
		if err != nil {
			return fmt.Errorf("error getting values for a diff: %v", err)
		}

		debug("%s performing dry run upgrade", logID)
		dry, err := upgrade.Run(name, chart, values)
		if err != nil && strings.Contains(err.Error(), "has no deployed releases") {
			if len(chart.Metadata.Version) > 0 && cpo.Version != "" {
				return d.SetNew("version", chart.Metadata.Version)
			}
			d.SetNewComputed("version")
			d.SetNewComputed("manifest")
			return nil
		} else if err != nil {
			return fmt.Errorf("error running dry run for a diff: %v", err)
		}

		jsonManifest, err := convertYAMLManifestToJSON(dry.Manifest)
		if err != nil {
			return err
		}
		manifest := redactSensitiveValues(string(jsonManifest), d)
		d.SetNew("manifest", manifest)
		debug("%s set manifest: %s", logID, jsonManifest)
	} else {
		d.Clear("manifest")
	}

	debug("%s Done", logID)

	// Set desired version from the Chart metadata if available
	if len(chart.Metadata.Version) > 0 {
		return d.SetNew("version", chart.Metadata.Version)
	}

	return d.SetNewComputed("version")
}

func setReleaseAttributes(d *schema.ResourceData, r *release.Release, meta interface{}) error {
	d.SetId(r.Name)

	if err := d.Set("version", r.Chart.Metadata.Version); err != nil {
		return err
	}

	if err := d.Set("namespace", r.Namespace); err != nil {
		return err
	}

	if err := d.Set("status", r.Info.Status.String()); err != nil {
		return err
	}

	cloakSetValues(r.Config, d)
	values := "{}"
	if r.Config != nil {
		v, err := json.Marshal(r.Config)
		if err != nil {
			return err
		}
		values = string(v)
	}

	m := meta.(*Meta)
	if m.ExperimentEnabled("manifest") {
		jsonManifest, err := convertYAMLManifestToJSON(r.Manifest)
		if err != nil {
			return err
		}
		manifest := redactSensitiveValues(string(jsonManifest), d)
		d.Set("manifest", manifest)
	}

	return d.Set("metadata", []map[string]interface{}{{
		"name":        r.Name,
		"revision":    r.Version,
		"namespace":   r.Namespace,
		"chart":       r.Chart.Metadata.Name,
		"version":     r.Chart.Metadata.Version,
		"app_version": r.Chart.Metadata.AppVersion,
		"values":      values,
	}})
}

func cloakSetValues(config map[string]interface{}, d resourceGetter) {
	for _, raw := range d.Get("set_sensitive").(*schema.Set).List() {
		set := raw.(map[string]interface{})
		cloakSetValue(config, set["name"].(string))
	}
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

func resourceReleaseExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	logID := fmt.Sprintf("[resourceReleaseExists: %s]", d.Get("name").(string))
	debug("%s Start", logID)

	m := meta.(*Meta)
	n := d.Get("namespace").(string)

	c, err := m.GetHelmConfiguration(n)
	if err != nil {
		return false, err
	}

	name := d.Get("name").(string)
	_, err = getRelease(m, c, name)

	debug("%s Done", logID)

	if err == nil {
		return true, nil
	}

	if err == errReleaseNotFound {
		return false, nil
	}

	return false, err
}

type resourceGetter interface {
	Get(string) interface{}
}

func getVersion(d resourceGetter, m *Meta) (version string) {
	version = d.Get("version").(string)

	if version == "" && d.Get("devel").(bool) {
		debug("setting version to >0.0.0-0")
		version = ">0.0.0-0"
	} else {
		version = strings.TrimSpace(version)
	}

	return
}

func getChart(d resourceGetter, m *Meta, name string, cpo *action.ChartPathOptions) (*chart.Chart, string, error) {
	//Load function blows up if accessed concurrently
	m.Lock()
	defer m.Unlock()

	// Checks if chart is a URL; checks if it's a valid URL to a .tgz file of the chart
	url, err := http.Get(name)
	if err == nil {
		contentType := url.Header.Get("Content-Type")

		if contentType != "binary/octet-stream" && contentType != "application/x-gzip" && contentType != "application/x-compressed-tar" {
			panic(contentType)
			return nil, "", fmt.Errorf("Not an absolute URL to the .tgz of the Chart")
		}
	}

	path, err := cpo.LocateChart(name, m.Settings)
	if err != nil {
		return nil, "", err
	}

	c, err := loader.Load(path)
	if err != nil {
		return nil, "", err
	}

	return c, path, nil
}

// Merges source and destination map, preferring values from the source map
// Taken from github.com/helm/pkg/cli/values/options.go
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func getValues(d resourceGetter) (map[string]interface{}, error) {
	base := map[string]interface{}{}

	for _, raw := range d.Get("values").([]interface{}) {
		if raw == nil {
			continue
		}

		values := raw.(string)
		if values == "" {
			continue
		}

		currentMap := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(values), &currentMap); err != nil {
			return nil, fmt.Errorf("---> %v %s", err, values)
		}

		base = mergeMaps(base, currentMap)
	}

	for _, raw := range d.Get("set").(*schema.Set).List() {
		set := raw.(map[string]interface{})
		if err := getValue(base, set); err != nil {
			return nil, err
		}
	}

	for _, raw := range d.Get("set_list").([]interface{}) {
		set_list := raw.(map[string]interface{})
		if err := getListValue(base, set_list); err != nil {
			return nil, err
		}
	}

	for _, raw := range d.Get("set_sensitive").(*schema.Set).List() {
		set := raw.(map[string]interface{})
		if err := getValue(base, set); err != nil {
			return nil, err
		}
	}

	return base, logValues(base, d)
}

func getListValue(base, set map[string]interface{}) error {
	name := set["name"].(string)
	listValue := set["value"].([]interface{}) // this is going to be a list

	listStringArray := make([]string, len(listValue))

	for i, s := range listValue {
		listStringArray[i] = s.(string)
	}
	listString := strings.Join(listStringArray, ",")
	if err := strvals.ParseInto(fmt.Sprintf("%s={%s}", name, listString), base); err != nil {
		return fmt.Errorf("failed parsing key %q with value %s, %s", name, listString, err)
	}

	return nil
}

func getValue(base, set map[string]interface{}) error {
	name := set["name"].(string)
	value := set["value"].(string)
	valueType := set["type"].(string)

	switch valueType {
	case "auto", "":
		if err := strvals.ParseInto(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			return fmt.Errorf("failed parsing key %q with value %s, %s", name, value, err)
		}
	case "string":
		if err := strvals.ParseIntoString(fmt.Sprintf("%s=%s", name, value), base); err != nil {
			return fmt.Errorf("failed parsing key %q with value %s, %s", name, value, err)
		}
	default:
		return fmt.Errorf("unexpected type: %s", valueType)
	}

	return nil
}

func logValues(values map[string]interface{}, d resourceGetter) error {
	// copy array to avoid change values by the cloak function.
	asJSON, _ := json.Marshal(values)
	var c map[string]interface{}
	err := json.Unmarshal(asJSON, &c)
	if err != nil {
		return err
	}

	cloakSetValues(c, d)

	y, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	log.Printf(
		"---[ values.yaml ]-----------------------------------\n%s\n",
		string(y),
	)

	return nil
}

func getRelease(m *Meta, cfg *action.Configuration, name string) (*release.Release, error) {
	debug("%s getRelease wait for lock", name)
	m.Lock()
	defer m.Unlock()
	debug("%s getRelease got lock, started", name)

	get := action.NewGet(cfg)
	debug("%s getRelease post action created", name)

	res, err := get.Run(name)
	debug("%s getRelease post run", name)

	if err != nil {
		debug("getRelease for %s errored", name)
		debug("%v", err)
		if strings.Contains(err.Error(), "release: not found") {
			return nil, errReleaseNotFound
		}

		debug("could not get release %s", err)

		return nil, err
	}

	debug("%s getRelease done", name)

	return res, nil
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

func isChartInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func chartPathOptions(d resourceGetter, m *Meta, cpo *action.ChartPathOptions) (*action.ChartPathOptions, string, error) {
	chartName := d.Get("chart").(string)
	repository := d.Get("repository").(string)

	var repositoryURL string
	if registry.IsOCI(repository) {
		// LocateChart expects the chart name to contain the full OCI path
		// see: https://github.com/helm/helm/blob/main/pkg/action/install.go#L678
		u, err := url.Parse(repository)
		if err != nil {
			return nil, "", err
		}
		u.Path = path.Join(u.Path, chartName)
		chartName = u.String()
	} else {
		var err error
		repositoryURL, chartName, err = resolveChartName(repository, strings.TrimSpace(chartName))
		if err != nil {
			return nil, "", err
		}
	}

	version := getVersion(d, m)

	cpo.CaFile = d.Get("repository_ca_file").(string)
	cpo.CertFile = d.Get("repository_cert_file").(string)
	cpo.KeyFile = d.Get("repository_key_file").(string)
	cpo.Keyring = d.Get("keyring").(string)
	cpo.RepoURL = repositoryURL
	cpo.Verify = d.Get("verify").(bool)
	if !useChartVersion(chartName, cpo.RepoURL) {
		cpo.Version = version
	}
	cpo.Username = d.Get("repository_username").(string)
	cpo.Password = d.Get("repository_password").(string)
	cpo.PassCredentialsAll = d.Get("pass_credentials").(bool)
	return cpo, chartName, nil
}

func resourceHelmReleaseImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	namespace, name, err := parseImportIdentifier(d.Id())
	if err != nil {
		return nil, errors.Errorf("Unable to parse identifier %s: %s", d.Id(), err)
	}

	m := meta.(*Meta)

	c, err := m.GetHelmConfiguration(namespace)
	if err != nil {
		return nil, err
	}

	r, err := getRelease(m, c, name)
	if err != nil {
		return nil, err
	}

	err = d.Set("name", r.Name)
	if err != nil {
		return nil, err
	}

	err = d.Set("description", r.Info.Description)
	if err != nil {
		return nil, err
	}

	err = d.Set("chart", r.Chart.Metadata.Name)
	if err != nil {
		return nil, err
	}

	for key, value := range defaultAttributes {
		err = d.Set(key, value)
		if err != nil {
			return nil, err
		}
	}

	if err := setReleaseAttributes(d, r, m); err != nil {
		return nil, err
	}

	return schema.ImportStatePassthroughContext(ctx, d, meta)
}

func parseImportIdentifier(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := errors.Errorf("Unexpected ID format (%q), expected namespace/name", id)
		return "", "", err
	}

	return parts[0], parts[1], nil
}

func resourceReleaseValidate(d resourceGetter, meta interface{}, cpo *action.ChartPathOptions) error {
	cpo, name, err := chartPathOptions(d, meta.(*Meta), cpo)
	if err != nil {
		return fmt.Errorf("malformed values: \n\t%s", err)
	}

	values, err := getValues(d)
	if err != nil {
		return err
	}

	return lintChart(meta.(*Meta), name, cpo, values)
}

func lintChart(m *Meta, name string, cpo *action.ChartPathOptions, values map[string]interface{}) (err error) {
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

// valuesKnown returns true if all of the values supplied to the release are known at plan time
func valuesKnown(d *schema.ResourceDiff) bool {
	rawPlan := d.GetRawPlan()
	checkAttributes := []string{
		"values",
		"set",
		"set_sensitive",
		"set_list",
	}
	for _, attr := range checkAttributes {
		if !rawPlan.GetAttr(attr).IsWhollyKnown() {
			return false
		}
	}
	return true
}

func useChartVersion(chart string, repo string) bool {
	// checks if chart is a URL or OCI registry

	if url, err := http.Get(chart); err == nil && !registry.IsOCI(chart) {
		return url.Header.Get("Content-Type") == "binary/octet-stream"
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
