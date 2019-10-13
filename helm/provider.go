package helm

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/mitchellh/go-homedir"

	// Import to initialize client auth plugins.

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Meta is the meta information structure for the provider
type Meta struct {
	//K8sClient        kubernetes.Interface
	K8sConfig        *rest.Config
	DefaultNamespace string
	data             *schema.ResourceData
	settings         *cli.EnvSettings
	HelmDriver       string
	//actionConfig     *action.Configuration
}

var k8sPrefix = "kubernetes.0."

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
				Description: "The path to the helm registry config file",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REPOSITORY_CACHE", helmpath.DataPath("repository")),
			},
			"helm_driver": {
				Type:        schema.TypeString,
				Optional:    false,
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
			"kubernetes": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Kubernetes configuration.",
				Elem:        kubernetesResource(),
			},
		},
		// ResourcesMap: map[string]*schema.Resource{
		// 	"helm_release":    resourceRelease(),
		// 	"helm_repository": resourceRepository(),
		// },
		// DataSourcesMap: map[string]*schema.Resource{
		// 	"helm_repository": dataRepository(),
		// },
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

func kubernetesResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_HOST", ""),
				Description: "The hostname (in form of URI) of Kubernetes master. Can be sourced from `KUBE_HOST`.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_USER", ""),
				Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_USER`.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_PASSWORD", ""),
				Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_PASSWORD`.",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_BEARER_TOKEN", ""),
				Description: "The bearer token to use for authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_BEARER_TOKEN`.",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_INSECURE", false),
				Description: "Whether server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`.",
			},
			"client_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_CERT_DATA", ""),
				Description: "PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.",
			},
			"client_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_KEY_DATA", ""),
				Description: "PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.",
			},
			"cluster_ca_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLUSTER_CA_CERT_DATA", ""),
				Description: "PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.",
			},
			"config_path": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"KUBE_CONFIG",
						"KUBECONFIG",
					},
					"~/.kube/config"),
				Description: "Path to the kube config file, defaults to ~/.kube/config. Can be sourced from `KUBE_CONFIG`.",
			},
			"config_context": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX", ""),
				Description: "Context to choose from the config file. Can be sourced from `KUBE_CTX`.",
			},
			"in_cluster": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Retrieve config from Kubernetes cluster.",
			},
			"load_config_file": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_LOAD_CONFIG_FILE", true),
				Description: "By default the local config (~/.kube/config) is loaded when you use this provider. This option at false disable this behaviour.",
			},
		},
	}
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	m := &Meta{data: d}

	if err := m.buildK8sConfig(m.data, terraformVersion); err != nil {
		return nil, err
	}

	m.buildSettings(m.data)

	return m, nil
}

func (m *Meta) buildSettings(d *schema.ResourceData) {

	settings := &cli.EnvSettings{
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

	//settings.KubeConfig = m.K8sConfig

	m.settings = settings

	if v, ok := d.GetOkExists("helm_driver"); ok {
		m.HelmDriver = v.(string)
	}
}

func (m *Meta) buildK8sConfig(d *schema.ResourceData, terraformVersion string) error {
	_, hasStatic := d.GetOk("kubernetes")

	c, err := getK8sConfig(d)
	if err != nil {
		debug("could not get Kubernetes config: %s", err)
		if !hasStatic {
			return err
		}
	}

	cfg, err := c.ClientConfig()
	if err != nil {
		debug("could not get Kubernetes client config: %s", err)
		if !hasStatic {
			return err
		}
	}

	if cfg == nil {
		cfg = &rest.Config{}
	}

	// Overriding with static configuration
	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraformVersion)

	if v, ok := k8sGetOk(d, "host"); ok {
		cfg.Host = v.(string)
	}
	if v, ok := k8sGetOk(d, "username"); ok {
		cfg.Username = v.(string)
	}
	if v, ok := k8sGetOk(d, "password"); ok {
		cfg.Password = v.(string)
	}
	if v, ok := k8sGetOk(d, "token"); ok {
		cfg.BearerToken = v.(string)
	}
	if v, ok := k8sGetOk(d, "insecure"); ok {
		cfg.Insecure = v.(bool)
	}
	if v, ok := k8sGetOk(d, "cluster_ca_certificate"); ok {
		cfg.CAData = []byte(v.(string))
	}
	if v, ok := k8sGetOk(d, "client_certificate"); ok {
		cfg.CertData = []byte(v.(string))
	}
	if v, ok := k8sGetOk(d, "client_key"); ok {
		cfg.KeyData = []byte(v.(string))
	}

	m.K8sConfig = cfg
	return nil
}

func k8sGetOk(d *schema.ResourceData, key string) (interface{}, bool) {
	value, ok := d.GetOk(k8sPrefix + key)

	// For boolean attributes the zero value is Ok
	switch value.(type) {
	case bool:
		value, ok = d.GetOkExists(k8sPrefix + key)
	}

	// fix: DefaultFunc is not being triggerred on TypeList
	schema := kubernetesResource().Schema[key]
	if !ok && schema.DefaultFunc != nil {
		value, _ = schema.DefaultFunc()

		switch v := value.(type) {
		case string:
			ok = len(v) != 0
		case bool:
			ok = v
		}
	}

	return value, ok
}

func k8sGet(d *schema.ResourceData, key string) interface{} {
	value, _ := k8sGetOk(d, key)
	return value
}

func getK8sConfig(d *schema.ResourceData) (clientcmd.ClientConfig, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}

	if !k8sGet(d, "in_cluster").(bool) && k8sGet(d, "load_config_file").(bool) {
		configPathSplit := strings.Split(k8sGet(d, "config_path").(string), ":")
		precedence := make([]string, len(configPathSplit))
		for i, path := range configPathSplit {
			expanded, err := homedir.Expand(path)
			if err != nil {
				debug("Error expanding path %s", err)
				return nil, err
			}
			precedence[i] = expanded
		}

		rules.Precedence = precedence
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig

		context := k8sGet(d, "config_context").(string)
		if context != "" {
			overrides.CurrentContext = context
		}
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides), nil
}

// GetHelmConfiguration will return a new Helm configuration
func (m *Meta) GetHelmConfiguration() (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	// Not sure if this should always use true for namespaces
	if err := actionConfig.Init(m.settings, true, m.HelmDriver, debug); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

func debug(format string, a ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintf(format, a...))
}
