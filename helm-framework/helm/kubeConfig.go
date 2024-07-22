package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	//"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/go-homedir"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	//clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Struct holding k8s client config, burst limit for api requests, and mutex for sync
type KubeConfig struct {
	ClientConfig clientcmd.ClientConfig
	Burst        int
	sync.Mutex
}

// Converting KubeConfig to a REST config, which will be used to create k8s clients
func (k *KubeConfig) ToRESTConfig() (*rest.Config, error) {
	config, err := k.ToRawKubeConfigLoader().ClientConfig()
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
func (m *Meta) NewKubeConfig(ctx context.Context, namespace string) (*KubeConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}
	configPaths := []string{}
	if m == nil || m.Data == nil || m.Data.Kubernetes.IsNull() || m.Data.Kubernetes.IsUnknown() {
		fmt.Println("Debug - One or more structural elements are nil")
		return nil, fmt.Errorf("configuration error: missing required structural data")
	}

	// Extract the first element from the Kubernetes list
	var kubernetesConfig KubernetesConfigModel
	var kubernetesConfigs []KubernetesConfigModel
	// TODO look into this next
	diags := m.Data.Kubernetes.ElementsAs(ctx, &kubernetesConfigs, true)
	if diags.HasError() {
		fmt.Println("Error extracting Kubernetes config", diags[0])
		return nil, fmt.Errorf("configuration error: unable to extract Kubernetes config %#v", diags[0])
	}
	if len(kubernetesConfigs) > 0 {
		kubernetesConfig = kubernetesConfigs[0]
	}

	// Check ConfigPath
	tflog.Debug(ctx, "Debug - m.Data.Kubernetes", map[string]interface{}{"Kubernetes": m.Data.Kubernetes})
	if !kubernetesConfig.ConfigPath.IsNull() {
		if v := kubernetesConfig.ConfigPath.ValueString(); v != "" {
			configPaths = []string{v}
			fmt.Println("Debug - ConfigPath:", kubernetesConfig.ConfigPath.ValueString())
			tflog.Debug(ctx, "Debug - ConfigPath", map[string]interface{}{"ConfigPath": kubernetesConfig.ConfigPath.ValueString()})
		}
	}
	if !kubernetesConfig.ConfigPaths.IsNull() {
		additionalPaths := expandStringSlice(kubernetesConfig.ConfigPaths.Elements())
		configPaths = append(configPaths, additionalPaths...)
	}
	if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}
	fmt.Println("Initial configPaths:", configPaths)
	tflog.Debug(ctx, "Initial configPaths", map[string]interface{}{"configPaths": configPaths})
	fmt.Println("Debug - loader struct1:", loader)
	if len(configPaths) > 0 {
		fmt.Println("Processing config paths:", configPaths)
		tflog.Debug(ctx, "Processing config paths", map[string]interface{}{
			"configPaths": configPaths,
		})
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				fmt.Println("Error expanding home directory:", p, "Error:", err)
				tflog.Error(ctx, "Error expanding home directory", map[string]interface{}{
					"path":  p,
					"error": err,
				})
				return nil, err
			}
			fmt.Println("Using kubeconfig path:", path)
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
		tflog.Debug(ctx, "Debug - loader struct2", map[string]interface{}{
			"loader": loader,
		})

		// Check ConfigContext
		if !kubernetesConfig.ConfigContext.IsNull() {
			overrides.CurrentContext = kubernetesConfig.ConfigContext.ValueString()
			fmt.Println("Setting config context:", overrides.CurrentContext)
			tflog.Debug(ctx, "Setting config context", map[string]interface{}{
				"configContext": overrides.CurrentContext,
			})
		}
		if !kubernetesConfig.ConfigContextAuthInfo.IsNull() {
			overrides.Context.AuthInfo = kubernetesConfig.ConfigContextAuthInfo.ValueString()
			fmt.Println("Setting config context auth info:", overrides.Context.AuthInfo)
			tflog.Debug(ctx, "Setting config context auth info", map[string]interface{}{
				"configContextAuthInfo": overrides.Context.AuthInfo,
			})
		}
		if !kubernetesConfig.ConfigContextCluster.IsNull() {
			overrides.Context.Cluster = kubernetesConfig.ConfigContextCluster.ValueString()
			fmt.Println("Setting config context cluster:", overrides.Context.Cluster)
			tflog.Debug(ctx, "Setting config context cluster", map[string]interface{}{
				"configContextCluster": overrides.Context.Cluster,
			})
		}
	}

	// Check and assign remaining fields
	if !kubernetesConfig.Insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = kubernetesConfig.Insecure.ValueBool()
		fmt.Println("Setting insecure skip TLS verify:", overrides.ClusterInfo.InsecureSkipTLSVerify)
		tflog.Debug(ctx, "Setting insecure skip TLS verify", map[string]interface{}{
			"insecureSkipTLSVerify": overrides.ClusterInfo.InsecureSkipTLSVerify,
		})
	}
	if !kubernetesConfig.TlsServerName.IsNull() {
		overrides.ClusterInfo.TLSServerName = kubernetesConfig.TlsServerName.ValueString()
		fmt.Println("Setting TLS server name:", overrides.ClusterInfo.TLSServerName)
		tflog.Debug(ctx, "Setting TLS server name", map[string]interface{}{
			"tlsServerName": overrides.ClusterInfo.TLSServerName,
		})
	}
	if !kubernetesConfig.ClusterCaCertificate.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(kubernetesConfig.ClusterCaCertificate.ValueString())
		fmt.Println("Setting cluster CA certificate")
		tflog.Debug(ctx, "Setting cluster CA certificate")
	}
	if !kubernetesConfig.ClientCertificate.IsNull() {
		overrides.AuthInfo.ClientCertificateData = []byte(kubernetesConfig.ClientCertificate.ValueString())
		fmt.Println("Setting client certificate")
		tflog.Debug(ctx, "Setting client certificate")
	}
	if !kubernetesConfig.Host.IsNull() && kubernetesConfig.Host.ValueString() != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(kubernetesConfig.Host.ValueString(), "", schema.GroupVersion{}, defaultTLS)
		if err != nil {
			fmt.Println("Error setting host:", kubernetesConfig.Host.ValueString(), "Error:", err)
			tflog.Error(ctx, "Error setting host", map[string]interface{}{
				"host":  kubernetesConfig.Host.ValueString(),
				"error": err,
			})
			return nil, err
		}
		overrides.ClusterInfo.Server = host.String()
		fmt.Println("Setting host:", overrides.ClusterInfo.Server)
		tflog.Debug(ctx, "Setting host", map[string]interface{}{
			"host": overrides.ClusterInfo.Server,
		})
	}

	if !kubernetesConfig.Username.IsNull() {
		overrides.AuthInfo.Username = kubernetesConfig.Username.ValueString()
		fmt.Println("Setting username:", overrides.AuthInfo.Username)
		tflog.Debug(ctx, "Setting username", map[string]interface{}{
			"username": overrides.AuthInfo.Username,
		})
	}
	if !kubernetesConfig.Password.IsNull() {
		overrides.AuthInfo.Password = kubernetesConfig.Password.ValueString()
		fmt.Println("Setting password")
		tflog.Debug(ctx, "Setting password")
	}
	if !kubernetesConfig.ClientKey.IsNull() {
		overrides.AuthInfo.ClientKeyData = []byte(kubernetesConfig.ClientKey.ValueString())
		fmt.Println("Setting client key")
		tflog.Debug(ctx, "Setting client key")
	}
	if !kubernetesConfig.Token.IsNull() {
		overrides.AuthInfo.Token = kubernetesConfig.Token.ValueString()
		fmt.Println("Setting token:", overrides.AuthInfo.Token)
		tflog.Debug(ctx, "Setting token", map[string]interface{}{
			"token": overrides.AuthInfo.Token,
		})
	}
	if !kubernetesConfig.ProxyUrl.IsNull() {
		overrides.ClusterDefaults.ProxyURL = kubernetesConfig.ProxyUrl.ValueString()
		fmt.Println("Setting proxy URL:", overrides.ClusterDefaults.ProxyURL)
		tflog.Debug(ctx, "Setting proxy URL", map[string]interface{}{
			"proxyURL": overrides.ClusterDefaults.ProxyURL,
		})
	}

	// Extract the first element from the Exec list
	// var execConfig *ExecConfigModel
	// if !kubernetesConfig.Exec.IsUnknown() {
	// 	var execConfigs []ExecConfigModel
	// 	diags := kubernetesConfig.Exec.ElementsAs(ctx, &execConfigs, false)
	// 	if diags.HasError() {
	// 		fmt.Println("Error extracting Exec config")
	// 		return nil, fmt.Errorf("configuration error: unable to extract Exec config")
	// 	}
	// 	if len(execConfigs) > 0 {
	// 		execConfig = &execConfigs[0]
	// 	}
	// }

	// if execConfig != nil {
	// 	args := execConfig.Args.Elements()
	// 	env := execConfig.Env.Elements()

	// 	exec := &clientcmdapi.ExecConfig{
	// 		APIVersion: execConfig.ApiVersion.ValueString(),
	// 		Command:    execConfig.Command.ValueString(),
	// 		Args:       expandStringSlice(args),
	// 	}
	// 	for k, v := range env {
	// 		exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: k, Value: v.(basetypes.StringValue).ValueString()})
	// 	}
	// 	overrides.AuthInfo.Exec = exec
	// 	fmt.Println("Setting exec configuration:", exec)
	// 	tflog.Debug(ctx, "Setting exec configuration", map[string]interface{}{
	// 		"execConfig": exec,
	// 	})
	// }

	overrides.Context.Namespace = "default"
	if namespace != "" {
		overrides.Context.Namespace = namespace
		fmt.Println("Setting namespace:", overrides.Context.Namespace)
		tflog.Debug(ctx, "Setting namespace", map[string]interface{}{
			"namespace": overrides.Context.Namespace,
		})
	}

	// Creating the k8s client config, using the loaded and overrides.
	burstLimit := int(m.Data.BurstLimit.ValueInt64())
	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		fmt.Println("Failed to initialize kubernetes config")
		tflog.Error(ctx, "Failed to initialize kubernetes config")
		return nil, fmt.Errorf("failed to initialize kubernetes config")
	}
	fmt.Println("Successfully initialized kubernetes config")
	tflog.Info(ctx, "Successfully initialized kubernetes config")
	fmt.Printf("ClientConfig: %+v\n", client)
	fmt.Printf("BurstLimit: %d\n", burstLimit)
	return &KubeConfig{ClientConfig: client, Burst: burstLimit}, nil
}

func expandStringSlice(input []attr.Value) []string {
	result := make([]string, len(input))
	for i, v := range input {
		result[i] = v.(types.String).ValueString()
	}
	return result
}
