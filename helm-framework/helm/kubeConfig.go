package helm

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/go-homedir"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Struct holding k8s client config, burst limit for api requests, and mutex for sync
type KubeConfig struct {
	ClientConfig clientcmd.ClientConfig
	Burst        int
	sync.Mutex
}

// Converting KubeConfig to a REST config, which will be used to create k8s clients
func (k *KubeConfig) ToRESTConfig() (*rest.Config, error) {
	config, err := k.ClientConfig.ClientConfig()
	return config, err
}

// Converting KubeConfig to a discovery client, which will be used to find api resources
func (k *KubeConfig) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := k.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	config.Burst = k.Burst
	return memory.NewMemCacheClient(discovery.NewDiscoveryClientForConfigOrDie(config)), nil
}

// Converting KubeConfig to a REST mapper, which will be used to map REST resources to their API obj
func (k *KubeConfig) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := k.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	// Use the appropriate types for the arguments
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)

	return expander, nil
}

// Function returning raw k8s client config
func (k *KubeConfig) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return k.ClientConfig
}

// Generates a k8s client config, based on providers settings and namespace, which this config will be used to interact with the k8s cluster
func (m *Meta) newKubeConfig(namespace *string) (*KubeConfig, error) {
	//Used to specify customer settings for k8s client config
	overrides := &clientcmd.ConfigOverrides{}
	//Used to load the k8s config files
	loader := &clientcmd.ClientConfigLoadingRules{}
	// Holds paths to k8s config files
	configPaths := []string{}

	//If config path is set, we will use it
	if v := m.Data.Kubernetes.ConfigPath.ValueString(); v != "" {
		configPaths = []string{v}
		//If config paths is set, we append all paths to config paths
	} else if !m.Data.Kubernetes.ConfigPaths.IsNull() {
		for _, p := range m.Data.Kubernetes.ConfigPaths.Elements() {
			configPaths = append(configPaths, p.(types.String).ValueString())
		}
		//if KUBE_CONFIG_PATHS is set, we split into indvidual paths
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}
	//If there are any config paths, we expand them to their full path
	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] Using kubeconfig: %s", path)
			expandedPaths = append(expandedPaths, path)
		}
		// If there's only one path, we set it as the explicit path
		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
			// If there is not, we set the precedence for the paths to be used by thr loaded
		} else {
			loader.Precedence = expandedPaths
		}
		// If ConfigContext is set, we overred the current context
		if ctx := m.Data.Kubernetes.ConfigContext.ValueString(); ctx != "" {
			overrides.CurrentContext = ctx
			log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
		}
		// If ConfigContextAuthInfo is set, we override the auth info
		if authInfo := m.Data.Kubernetes.ConfigContextAuthInfo.ValueString(); authInfo != "" {
			overrides.Context.AuthInfo = authInfo
		}
		//If ConfigContextCluster is set, we override the cluster
		if cluster := m.Data.Kubernetes.ConfigContextCluster.ValueString(); cluster != "" {
			overrides.Context.Cluster = cluster
		}
		log.Printf("[DEBUG] Using overridden context: %#v", overrides.Context)
	}
	//Checking whether or not to override the tls releated settings
	if v := m.Data.Kubernetes.Insecure.ValueBool(); v {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v
	}
	if v := m.Data.Kubernetes.TlsServerName.ValueString(); v != "" {
		overrides.ClusterInfo.TLSServerName = v
	}
	if v := m.Data.Kubernetes.ClusterCaCertificate.ValueString(); v != "" {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(v)
	}
	if v := m.Data.Kubernetes.ClientCertificate.ValueString(); v != "" {
		overrides.AuthInfo.ClientCertificateData = []byte(v)
	}
	//Sets the k8s api server urls, in considerations to the TLS settings
	if v := m.Data.Kubernetes.Host.ValueString(); v != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(v, "", schema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, err
		}
		overrides.ClusterInfo.Server = host.String()
	}
	//Checking whether or not to override auth details such as "username, pw, ClientKey, and token"
	if v := m.Data.Kubernetes.Username.ValueString(); v != "" {
		overrides.AuthInfo.Username = v
	}
	if v := m.Data.Kubernetes.Password.ValueString(); v != "" {
		overrides.AuthInfo.Password = v
	}
	if v := m.Data.Kubernetes.ClientKey.ValueString(); v != "" {
		overrides.AuthInfo.ClientKeyData = []byte(v)
	}
	if v := m.Data.Kubernetes.Token.ValueString(); v != "" {
		overrides.AuthInfo.Token = v
	}
	if v := m.Data.Kubernetes.ProxyUrl.ValueString(); v != "" {
		overrides.ClusterDefaults.ProxyURL = v
	}
	//calling the func to get their values before passing them
	// If Exec settings are provided by the user, we create an ExecConfig, which will be used for command authentication
	if v := m.Data.Kubernetes.Exec; v != nil {
		args := v.Args.Elements()
		env := v.Env.Elements()

		exec := &clientcmdapi.ExecConfig{
			APIVersion: v.ApiVersion.ValueString(),
			Command:    v.Command.ValueString(),
			Args:       expandStringSlice(args),
		}
		for k, v := range env {
			exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: k, Value: v.(types.String).ValueString()})
		}
		overrides.AuthInfo.Exec = exec
	}
	//Sets the namespace to the provided value, if the value is not provided it will be default
	overrides.Context.Namespace = "default"
	if namespace != nil {
		overrides.Context.Namespace = *namespace
	}
	// Creating the k8s client config, using the loaded and overrides.
	burstLimit := int(m.Data.BurstLimit.ValueInt64())
	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		log.Printf("[ERROR] Failed to initialize kubernetes config")
		return nil, nil
	}
	log.Printf("[INFO] Successfully initialized kubernetes config")
	// Returning KubeConfig objkect, setting the burst limit for api req
	return &KubeConfig{ClientConfig: client, Burst: burstLimit}, nil
}

func expandStringSlice(input []attr.Value) []string {
	result := make([]string, len(input))
	for i, v := range input {
		result[i] = v.(types.String).ValueString()
	}
	return result
}
