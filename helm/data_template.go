package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/timeconv"
)

func dataTemplate() *schema.Resource {
	return &schema.Resource{
		Read: dataTemplateRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Release name.",
			},
			"repository": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Repository where to locate the requested chart. If the location is an URL the chart is rendered without installing the repository.",
			},
			"chart": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Chart name to be rendered.",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Specify the exact chart version to render. If this is not specified, the latest version is rendered.",
			},
			"devel": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored.",
				// Suppress changes of this attribute if `version` is set
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return d.Get("version").(string) != ""
				},
			},
			"verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Verify the package before rendering it.",
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
			"templates": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Explicit list of chart templates to render. Paths to chart templates are relative to the root folder of the chart, e.g. `templates/deployment.yaml`. If not provided, all templates of the chart are rendered.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"values": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of values in raw yaml to be used when rendering the chart templates.",
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
					},
				},
			},
			"set_string": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Custom string values to be merged with the values.",
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
			"kube_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     fmt.Sprintf("%s.%s", chartutil.DefaultKubeVersion.Major, chartutil.DefaultKubeVersion.Minor),
				Description: fmt.Sprintf("Kubernetes version used as Capabilities.KubeVersion.Major/Minor (default \"%s.%s\")", chartutil.DefaultKubeVersion.Major, chartutil.DefaultKubeVersion.Minor),
			},
			"namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				Description: "Namespace of the rendered resources.",
			},
			"rendered": {
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
			"manifests": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Description: "Map of rendered chart templates indexed by the template name.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataTemplateRead(d *schema.ResourceData, meta interface{}) error {
	m := meta.(*Meta)

	name := d.Get("name").(string)

	c, _, err := getChart(d, m)
	if err != nil {
		return err
	}

	values, err := getValues(d)
	if err != nil {
		return err
	}

	ns := d.Get("namespace").(string)
	kubeVersion := d.Get("kube_version").(string)

	config := &chart.Config{Raw: string(values)}

	t := &templateCmd{
		namespace:        ns,
		chartPath:        "",
		releaseName:      name,
		releaseIsUpgrade: false,
		kubeVersion:      kubeVersion,
	}

	if templatesAttr, ok := d.GetOk("templates"); ok {
		templates := templatesAttr.([]interface{})
		t.renderFiles = []string{}

		for _, template := range templates {
			t.renderFiles = append(t.renderFiles, template.(string))
		}
	}

	out, err := t.render(c, config)
	if err != nil {
		return fmt.Errorf("cannot render template: %v", err)
	}

	d.SetId(name)

	err = d.Set("notes", out.notes)
	if err != nil {
		return err
	}

	err = d.Set("rendered", out.rendered.String())
	if err != nil {
		return err
	}

	err = d.Set("manifests", out.manifests)
	if err != nil {
		return err
	}

	return nil
}

type templateOut struct {
	notes     string
	manifests map[string]string
	rendered  strings.Builder
}

func newTemplateOut() *templateOut {
	t := &templateOut{}
	t.manifests = make(map[string]string)
	return t
}

// templateCmd originates from the implementation of the helm template command
// in github.com/helm/helm/cmd/helm/template.go. Fields which are unnecessary
// in the context of this data source have been omitted.
type templateCmd struct {
	namespace        string
	chartPath        string
	releaseName      string
	releaseIsUpgrade bool
	renderFiles      []string
	kubeVersion      string
}

// render originates from the implementation of the helm template command and
// have been adapted for the integration in this data source.
func (t *templateCmd) render(c *chart.Chart, config *chart.Config) (*templateOut, error) {
	tOut := newTemplateOut()

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      t.releaseName,
			IsInstall: !t.releaseIsUpgrade,
			IsUpgrade: t.releaseIsUpgrade,
			Time:      timeconv.Now(),
			Namespace: t.namespace,
		},
		KubeVersion: t.kubeVersion,
	}

	renderedTemplates, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, err
	}

	listManifests := manifest.SplitManifests(renderedTemplates)
	var manifestsToRender []manifest.Manifest

	// if we have a list of files to render, then check that each of the
	// provided files exists in the chart.
	if len(t.renderFiles) > 0 {
		for _, f := range t.renderFiles {
			missing := true

			for _, manifest := range listManifests {
				// manifest.Name is rendered using linux-style filepath separators on Windows as
				// well as macOS/linux.
				manifestPathSplit := strings.Split(manifest.Name, "/")
				// remove the chart name from the path
				manifestPathSplit = manifestPathSplit[1:]
				toJoin := append([]string{t.chartPath}, manifestPathSplit...)
				manifestPath := filepath.Join(toJoin...)

				// if the filepath provided matches a manifest path in the
				// chart, render that manifest
				if f == manifestPath {
					manifestsToRender = append(manifestsToRender, manifest)
					missing = false
				}
			}
			if missing {
				return nil, fmt.Errorf("could not find template %s in chart", f)
			}
		}
	} else {
		// no renderFiles provided, render all manifests in the chart
		manifestsToRender = listManifests
	}

	for _, m := range tiller.SortByKind(manifestsToRender) {
		data := m.Content
		b := filepath.Base(m.Name)
		if b == "NOTES.txt" {
			tOut.notes = data
			continue
		}
		if strings.HasPrefix(b, "_") {
			continue
		}

		tOut.manifests[m.Name] = data
		rendered := &tOut.rendered

		fmt.Fprintf(rendered, "---\n# Source: %s\n", m.Name)
		fmt.Fprintln(rendered, data)
	}

	return tOut, nil
}
