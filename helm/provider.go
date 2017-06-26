package helm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/helm"
	helm_env "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	tiller_env "k8s.io/helm/pkg/tiller/environment"
	"k8s.io/helm/pkg/tlsutil"
)

var (
	tlsCaCertFile string // path to TLS CA certificate file
	tlsCertFile   string // path to TLS certificate file
	tlsKeyFile    string // path to TLS key file
	tlsVerify     bool   // enable TLS and verify remote certificates
	tlsEnable     bool   // enable TLS

	kubeContext  string
	tillerTunnel *kube.Tunnel
	settings     helm_env.EnvSettings
)

// Provider returns the provider schema to Terraform.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"helm_host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(helm_env.HostEnvVar, ""),
				Description: "Set an alternative Tiller host. The format is host:port.",
			},
			"helm_home": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(helm_env.HomeEnvVar, helm_env.DefaultHelmHome),
				Description: "Set an alternative location for Helm files. By default, these are stored in '~/.helm'.",
			},
			"plugins_disable": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(helm_env.PluginDisableEnvVar, "true"),
				Description: "Disable plugins. Set HELM_NO_PLUGINS=1 to disable plugins.",
			},
			"tiller_namespace": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     tiller_env.DefaultTillerNamespace,
				Description: "Set an alternative Tiller namespace.",
			},
			"kube_config": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{
						"KUBE_CONFIG",
						"KUBECONFIG",
					},
					"~/.kube/config"),
				Description: "Path to the kube config file, defaults to ~/.kube/config",
			},

			"tls": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable TLS for request.",
			},
			"tls_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable TLS for request and verify remote.",
			},
			"tls_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "$HELM_HOME/key.pem",
				Description: "Path to TLS key file",
			},
			"tls_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "$HELM_HOME/cert.pem",
				Description: "Path to TLS certificate file.",
			},
			"tls_ca": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "$HELM_HOME/ca.pem",
				Description: "Path to TLS CA certificate file.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"helm_chart":      resourceChart(),
			"helm_repository": resourceRepository(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	tlsCaCertFile = d.Get("tls_ca").(string)
	tlsCertFile = d.Get("tls_cert").(string)
	tlsKeyFile = d.Get("tls_key").(string)
	tlsVerify = d.Get("tls_verify").(bool)
	tlsEnable = d.Get("tls").(bool)

	settings.Home = helmpath.Home(d.Get("helm_home").(string))
	settings.TillerHost = d.Get("helm_host").(string)
	settings.TillerNamespace = d.Get("tiller_namespace").(string)

	if err := setupConnection(); err != nil {
		return nil, err
	}

	return newClient()
}

func debug(format string, a ...interface{}) {
	log.Printf("[DEBUG] %s", fmt.Sprintf(format, a...))
}

// following functions copy and pasted from:
// https://github.com/kubernetes/helm/blob/9f9b3e872979d97fc9edd57c9ca2f86feaae0259/cmd/helm/helm.go
func setupConnection() error {
	if settings.TillerHost == "" {
		config, client, err := getKubeClient(kubeContext)
		if err != nil {
			return err
		}

		tunnel, err := portforwarder.New(settings.TillerNamespace, client, config)
		if err != nil {
			fmt.Println(err, settings.TillerNamespace)
			return err
		}

		settings.TillerHost = fmt.Sprintf("localhost:%d", tunnel.Local)
		debug("Created tunnel using local port: '%d'\n", tunnel.Local)

	}

	// Set up the gRPC config.
	debug("SERVER: %q\n", settings.TillerHost)

	// Plugin support.
	return nil
}

func getKubeClient(context string) (*rest.Config, kubernetes.Interface, error) {
	config, err := configForContext(context)
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get Kubernetes client: %s", err)
	}
	return config, client, nil
}

func newClient() (helm.Interface, error) {
	options := []helm.Option{helm.Host(settings.TillerHost)}

	if tlsVerify || tlsEnable {
		tlsopts := tlsutil.Options{KeyFile: tlsKeyFile, CertFile: tlsCertFile, InsecureSkipVerify: true}
		if tlsVerify {
			tlsopts.CaCertFile = tlsCaCertFile
			tlsopts.InsecureSkipVerify = false
		}

		tlscfg, err := tlsutil.ClientConfig(tlsopts)
		if err != nil {
			return nil, err
		}

		options = append(options, helm.WithTLS(tlscfg))
	}

	return helm.NewClient(options...), nil
}

// configForContext creates a Kubernetes REST client configuration for a given kubeconfig context.
func configForContext(context string) (*rest.Config, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not get Kubernetes config for context %q: %s", context, err)
	}
	return config, nil
}
