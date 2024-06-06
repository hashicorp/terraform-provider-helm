package helm

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
func (m *Meta) newKubeConfig(ctx context.Context, namespace string) (*KubeConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}
	configPaths := []string{}
	if v := m.Data.Kubernetes.ConfigPath.ValueString(); v != "" {
		configPaths = []string{v}
	} else if !m.Data.Kubernetes.ConfigPaths.IsNull() {
		configPaths = expandStringSlice(m.Data.Kubernetes.ConfigPaths.Elements())
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			tflog.Debug(ctx, "Using kubeconfig", map[string]interface{}{
				"path": path,
			})
			expandedPaths = append(expandedPaths, path)
		}
		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		overrides.CurrentContext = m.Data.Kubernetes.ConfigContext.ValueString()
		overrides.Context.AuthInfo = m.Data.Kubernetes.ConfigContextAuthInfo.ValueString()
		overrides.Context.Cluster = m.Data.Kubernetes.ConfigContextCluster.ValueString()

	}
	overrides.ClusterInfo.InsecureSkipTLSVerify = m.Data.Kubernetes.Insecure.ValueBool()
	overrides.ClusterInfo.TLSServerName = m.Data.Kubernetes.TlsServerName.ValueString()
	overrides.ClusterInfo.CertificateAuthorityData = []byte(m.Data.Kubernetes.ClusterCaCertificate.ValueString())
	overrides.AuthInfo.ClientCertificateData = []byte(m.Data.Kubernetes.ClientCertificate.ValueString())

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
	overrides.AuthInfo.Username = m.Data.Kubernetes.Username.ValueString()
	overrides.AuthInfo.Password = m.Data.Kubernetes.Password.ValueString()
	overrides.AuthInfo.ClientKeyData = []byte(m.Data.Kubernetes.ClientKey.ValueString())
	overrides.AuthInfo.Token = m.Data.Kubernetes.Token.ValueString()
	overrides.ClusterDefaults.ProxyURL = m.Data.Kubernetes.ProxyUrl.ValueString()

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
	overrides.Context.Namespace = "default"
	if namespace != "" {
		overrides.Context.Namespace = namespace
	}
	// Creating the k8s client config, using the loaded and overrides.
	burstLimit := int(m.Data.BurstLimit.ValueInt64())
	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		tflog.Error(ctx, "Failed to initialize kubernetes config")
		return nil, nil
	}
	tflog.Info(ctx, "Successfully initialized kubernetes config")
	return &KubeConfig{ClientConfig: client, Burst: burstLimit}, nil
}

func expandStringSlice(input []attr.Value) []string {
	result := make([]string, len(input))
	for i, v := range input {
		result[i] = v.(types.String).ValueString()
	}
	return result
}
