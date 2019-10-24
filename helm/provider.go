package helm

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	// Import to initialize client auth plugins.

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Meta is the meta information structure for the provider
type Meta struct {
	data       *schema.ResourceData
	Settings   *cli.EnvSettings
	HelmDriver string

	// Used to lock some operations
	sync.Mutex
}

// Provider returns the provider schema to Terraform.
func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Debug indicates whether or not Helm is running in Debug mode.",
				DefaultFunc: schema.EnvDefaultFunc("HELM_DEBUG", false),
			},
			"plugins_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the helm plugins directory",
				DefaultFunc: schema.EnvDefaultFunc("HELM_PLUGINS", helmpath.DataPath("plugins")),
			},
			"registry_config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the helm registry config file",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REGISTRY_CONFIG", helmpath.DataPath("registry.json")),
			},
			"repository_config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the helm repository conf file",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REPOSITORY_CONFIG", helmpath.DataPath("repositories.yaml")),
			},
			"repository_cache": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the helm repository conf file",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REPOSITORY_CACHE", helmpath.DataPath("repository")),
			},
			"helm_driver": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The backend storage driver. Values are: configmap, secret, memory",
				DefaultFunc: schema.EnvDefaultFunc("HELM_DRIVER", "secret"),
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					drivers := []string{
						"configmap",
						"secret",
						"memory",
					}

					v := strings.ToLower(val.(string))

					for _, d := range drivers {
						if d == v {
							return
						}
					}
					errs = append(errs, fmt.Errorf("%s must be a valid storage driver", v))
					return
				},
			},
			"kube_config_path": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"KUBE_CONFIG",
						"KUBECONFIG",
					},
					"~/.kube/config"),
				Description: "Path to the kube config file, defaults to ~/.kube/config. Can be sourced from `KUBE_CONFIG`.",
			},
			"kube_config_context": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HELM_KUBECONTEXT", ""),
				Description: "Context to choose from the config file. Can be sourced from `HELM_KUBECONTEXT`.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"helm_repository": resourceRepository(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"helm_repository": dataRepository(),
		},
	}
	p.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(d, terraformVersion)
	}
	return p
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	m := &Meta{data: d}
	m.buildSettings(m.data)

	return m, nil
}

func (m *Meta) buildSettings(d *schema.ResourceData) {

	settings := cli.EnvSettings{
		Debug: d.Get("debug").(bool),
	}

	if v, ok := d.GetOkExists("plugins_path"); ok {
		settings.PluginsDirectory = v.(string)
	}

	if v, ok := d.GetOkExists("registry_config_path"); ok {
		settings.RegistryConfig = v.(string)
	}

	if v, ok := d.GetOkExists("repository_config_path"); ok {
		settings.RepositoryConfig = v.(string)
	}

	if v, ok := d.GetOk("kube_config_path"); ok {
		settings.KubeConfig = v.(string)
	}

	if v, ok := d.GetOkExists("kube_config_context"); ok {
		settings.KubeContext = v.(string)
	}

	if v, ok := d.GetOkExists("repository_cache"); ok {
		settings.RepositoryCache = v.(string)
	}

	if v, ok := d.GetOkExists("helm_driver"); ok {
		m.HelmDriver = v.(string)
	}

	m.Settings = &settings
}

// GetHelmConfiguration will return a new Helm configuration
func (m *Meta) GetHelmConfiguration() (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	// Not sure if this should always use true for namespaces
	if err := actionConfig.Init(m.Settings, true, m.HelmDriver, debug); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

func debug(format string, a ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintf(format, a...))
}
