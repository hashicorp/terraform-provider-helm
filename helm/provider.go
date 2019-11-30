package helm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/mitchellh/go-homedir"

	// Import to initialize client auth plugins.

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Meta is the meta information structure for the provider
type Meta struct {
	data             *schema.ResourceData
	Settings         *cli.EnvSettings
	KubernetesConfig KubernetesConfig
	HelmDriver       string

	// Used to lock some operations
	sync.Mutex
}

// KubernetesConfig stores the k8s configuration
type KubernetesConfig struct {
	KubeConfig  string
	Context     string
	Username    string
	Password    string
	BearerToken string
	APIServer   string
	Insecure    bool
	CertFile    string
	KeyFile     string
	CAFile      string
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
				Description: "The path to the registry config file",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REGISTRY_CONFIG", helmpath.ConfigPath("registry.json")),
			},
			"repository_config_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the file containing repository names and URLs",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REPOSITORY_CONFIG", helmpath.ConfigPath("repositories.yaml")),
			},
			"repository_cache": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the file containing cached repository indexes",
				DefaultFunc: schema.EnvDefaultFunc("HELM_REPOSITORY_CACHE", helmpath.CachePath("repository")),
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
			"kubernetes": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Kubernetes configuration.",
				Elem:        kubernetesResource(),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"helm_release":    resourceRelease(),
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
	err := m.buildSettings(m.data)

	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Meta) buildSettings(d *schema.ResourceData) error {

	settings := cli.EnvSettings{
		Debug: d.Get("debug").(bool),
	}

	if v, ok := d.GetOk("plugins_path"); ok {
		settings.PluginsDirectory = v.(string)
	}

	if v, ok := d.GetOk("registry_config_path"); ok {
		settings.RegistryConfig = v.(string)
	}

	if v, ok := d.GetOk("repository_config_path"); ok {
		settings.RepositoryConfig = v.(string)
	}

	if v, ok := d.GetOk("repository_cache"); ok {
		settings.RepositoryCache = v.(string)
	}

	if v, ok := d.GetOk("helm_driver"); ok {
		m.HelmDriver = v.(string)
	}

	m.Settings = &settings
	m.getK8sConfig(d)

	return nil
}

var k8sPrefix = "kubernetes.0."

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

func (m *Meta) getK8sConfig(d *schema.ResourceData) error {
	kc := KubernetesConfig{}

	// Not sure if in_cluster is still valid here.
	if !k8sGet(d, "in_cluster").(bool) && k8sGet(d, "load_config_file").(bool) {
		if v, ok := k8sGetOk(d, "config_path"); ok {
			expanded, err := homedir.Expand(v.(string))
			if err != nil {
				debug("Error expanding path %s", err)
				return err
			}
			kc.KubeConfig = expanded
		}
	}

	if v, ok := k8sGetOk(d, "config_context"); ok {
		kc.Context = v.(string)
	}

	if v, ok := k8sGetOk(d, "username"); ok {
		kc.Username = v.(string)
	}

	if v, ok := k8sGetOk(d, "password"); ok {
		kc.Password = v.(string)
	}

	if v, ok := k8sGetOk(d, "token"); ok {
		kc.BearerToken = v.(string)
	}

	if v, ok := k8sGetOk(d, "insecure"); ok {
		kc.Insecure = v.(bool)
	}

	if v, ok := k8sGetOk(d, "client_certificate"); ok {
		v := v.(string)
		if path, err := prepareTempCertFile("cert", &v); err == nil {
			kc.CertFile = path
		}
	}

	if v, ok := k8sGetOk(d, "client_key"); ok {
		v := v.(string)
		if path, err := prepareTempCertFile("key", &v); err == nil {
			kc.KeyFile = path
		}
	}

	if v, ok := k8sGetOk(d, "cluster_ca_certificate"); ok {
		v := v.(string)
		if path, err := prepareTempCertFile("ca", &v); err == nil {
			kc.CAFile = path
		}
	}

	if v, ok := k8sGetOk(d, "host"); ok {
		kc.APIServer = v.(string)
	}

	m.KubernetesConfig = kc
	return nil
}

func prepareTempCertFile(prefix string, data *string) (string, error) {
	file, err := ioutil.TempFile(os.TempDir(), prefix+".*.pem")

	if err != nil {
		debug("Cannot create temporary file: %s", err)
		return "", err
	}

	b := bytes.NewBufferString(*data).Bytes()

	if _, err := file.Write(b); err != nil {
		debug("Cannot write to temporary file: %s", err)
		return "", err
	}
	return file.Name(), nil
}

// GetHelmConfiguration will return a new Helm configuration
func (m *Meta) GetHelmConfiguration(namespace string) (*action.Configuration, error) {
	m.Lock()
	defer m.Unlock()

	actionConfig := new(action.Configuration)
	cf := getKubernetesConfiguration(m.KubernetesConfig, namespace)

	if err := actionConfig.Init(cf, namespace, m.HelmDriver, debug); err != nil {
		return nil, err
	}

	return actionConfig, nil
}

func getKubernetesConfiguration(kc KubernetesConfig, namespace string) *genericclioptions.ConfigFlags {
	cf := genericclioptions.NewConfigFlags(true)

	cf.KubeConfig = &kc.KubeConfig
	cf.Context = &kc.Context
	cf.Username = &kc.Username
	cf.Password = &kc.Password
	cf.BearerToken = &kc.BearerToken
	cf.Insecure = &kc.Insecure
	cf.APIServer = &kc.APIServer
	cf.CertFile = &kc.CertFile
	cf.KeyFile = &kc.KeyFile
	cf.CAFile = &kc.CAFile
	cf.Namespace = &namespace

	return cf
}

func debug(format string, a ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintf(format, a...))
}
