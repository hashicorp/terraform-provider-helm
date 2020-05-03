package helm

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/releaseutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// defaultTemplateAttributes template attribute values
var defaultTemplateAttributes = map[string]interface{}{
	"validate":     false,
	"include_crds": false,
	"is_upgrade":   false,
}

func dataTemplate() *schema.Resource {
	return &schema.Resource{
		Read: dataTemplateRead,
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
			"templates": {
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
			"manifest_bundle": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Concatenated rendered chart templates. This corresponds to the output of the `helm template` command.",
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

func dataTemplateRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)
	n := d.Get("namespace").(string)

	apiVersions := []string{}

	if apiVersionsAttr, ok := d.GetOk("api_versions"); ok {
		apiVersionsValues := apiVersionsAttr.([]interface{})

		for _, apiVersion := range apiVersionsValues {
			apiVersions = append(apiVersions, apiVersion.(string))
		}
	}

	templates := []string{}

	if templatesAttr, ok := d.GetOk("templates"); ok {
		templatesValues := templatesAttr.([]interface{})

		for _, showFile := range templatesValues {
			templates = append(templates, showFile.(string))
		}
	}

	debug("Getting Config")

	actionConfig, err := m.GetHelmConfiguration(n)
	if err != nil {
		return err
	}

	cpo, chartName, err := chartPathOptions(d, m)
	if err != nil {
		return err
	}
	debug("Getting chart")

	chart, path, err := getChart(d, m, chartName, cpo)
	if err != nil {
		return err
	}

	debug("Got Chart from Helm")

	p := getter.All(m.Settings)

	values, err := getValues(d)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chart)
	if !validInstallableChart {
		return err
	}

	updateDependency := d.Get("dependency_update").(bool)

	if req := chart.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chart, req); err != nil {
			if updateDependency {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        path,
					Keyring:          d.Get("keyring").(string),
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: m.Settings.RepositoryConfig,
					RepositoryCache:  m.Settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	client := action.NewInstall(actionConfig)
	client.ChartPathOptions = *cpo
	client.ClientOnly = false
	client.DryRun = true
	client.DisableHooks = d.Get("disable_webhooks").(bool)
	client.Wait = d.Get("wait").(bool)
	client.Devel = d.Get("devel").(bool)
	client.DependencyUpdate = updateDependency
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

	// Adapted from client configuration in src/github.com/helm/helm/cmd/helm/template.go
	client.DryRun = true
	// Do not set fixed release name like in helm template
	//client.ReleaseName = "RELEASE-NAME"
	client.Replace = true // Skip the name check
	client.ClientOnly = !d.Get("validate").(bool)
	client.APIVersions = chartutil.VersionSet(apiVersions)
	client.IncludeCRDs = d.Get("include_crds").(bool)

	debug("Rendering Chart")

	rel, err := client.Run(chart, values)

	if err != nil && rel == nil {
		return err
	}

	if rel == nil {
		return fmt.Errorf("unexpected result from client.Run")
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))

	if !client.DisableHooks {
		for _, m := range rel.Hooks {
			fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
		}
	}

	// Difference to the implementation of helm template in newTemplateCmd:
	// Independent of templates, names of the charts templates are always resolved from the manifests
	// to be able to populate the keys in the manifests computed attribute.
	var manifestsToRender []string

	splitManifests := releaseutil.SplitManifests(manifests.String())
	manifestsKeys := make([]string, 0, len(splitManifests))
	for k := range splitManifests {
		manifestsKeys = append(manifestsKeys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(manifestsKeys))

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

	// if we have a list of files to render, then check that each of the
	// provided files exists in the chart.
	if len(templates) > 0 {
		for _, f := range templates {
			missing := true

			for manifestKey, manifestName := range manifestNamesByKey {
				// manifest.Name is rendered using linux-style filepath separators on Windows as
				// well as macOS/linux.
				manifestPathSplit := strings.Split(manifestName, "/")
				manifestPath := filepath.Join(manifestPathSplit...)

				// if the filepath provided matches a manifest path in the
				// chart, render that manifest
				if matched, _ := filepath.Match(f, manifestPath); !matched {
					continue
				}
				manifestsToRender = append(manifestsToRender, manifestKey)
				missing = false
			}

			if missing {
				return fmt.Errorf("could not find template %s in chart", f)
			}
		}
	} else {
		manifestsToRender = manifestsKeys
	}

	// Map from rendered manifests to data source output
	computedManifests := make(map[string]string, 0)
	computedManifestBundle := &strings.Builder{}

	for _, manifestKey := range manifestsToRender {
		manifest := splitManifests[manifestKey]
		manifestName := manifestNamesByKey[manifestKey]

		// Manifests
		computedManifests[manifestName] = manifest

		// Manifest bundle
		fmt.Fprintf(computedManifestBundle, "---\n%s\n", manifest)
	}

	computedNotes := rel.Info.Notes

	d.SetId(name)

	err = d.Set("manifests", computedManifests)
	if err != nil {
		return err
	}

	err = d.Set("manifest_bundle", computedManifestBundle.String())
	if err != nil {
		return err
	}

	err = d.Set("notes", computedNotes)
	if err != nil {
		return err
	}

	return nil
}
