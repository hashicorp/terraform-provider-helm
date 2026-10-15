// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

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
	QPS          float32
	sync.Mutex

	// Helm asks for a discovery client and a REST mapper for every object it
	// builds. They are created once, so all of them share one discovery cache.
	discoveryMutex  sync.Mutex
	discoveryClient discovery.CachedDiscoveryInterface
	restMapper      meta.RESTMapper
}

// Converting KubeConfig to a REST config, which will be used to create k8s clients
func (k *KubeConfig) ToRESTConfig() (*rest.Config, error) {
	config, err := k.ToRawKubeConfigLoader().ClientConfig()
	return config, err
}

// Converting KubeConfig to a discovery client, which will be used to find api resources.
// The client is created once and cached, so its discovery cache is shared.
func (k *KubeConfig) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	k.discoveryMutex.Lock()
	defer k.discoveryMutex.Unlock()
	return k.cachedDiscoveryClient()
}

func (k *KubeConfig) cachedDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if k.discoveryClient != nil {
		return k.discoveryClient, nil
	}
	config, err := k.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	config.Burst = k.Burst
	config.QPS = k.QPS
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	k.discoveryClient = memory.NewMemCacheClient(discoveryClient)
	return k.discoveryClient, nil
}

// Converting KubeConfig to a REST mapper, which will be used to map REST resources to their API obj.
// The mapper is created once on the shared discovery client: an invalidated
// discovery client resets it as well.
func (k *KubeConfig) ToRESTMapper() (meta.RESTMapper, error) {
	k.discoveryMutex.Lock()
	defer k.discoveryMutex.Unlock()
	if k.restMapper != nil {
		return k.restMapper, nil
	}
	discoveryClient, err := k.cachedDiscoveryClient()
	if err != nil {
		return nil, err
	}

	// Using the appropriate types for the arguments
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)

	warningHandler := func(warning string) {
		fmt.Printf("Warning: %s\n", warning)
	}

	// Pass the warning handler to the NewShortcutExpander function
	k.restMapper = restmapper.NewShortcutExpander(mapper, discoveryClient, warningHandler)
	return k.restMapper, nil
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
		return nil, fmt.Errorf("configuration error: missing required structural data")
	}

	tflog.Debug(ctx, "Raw Kubernetes Data before conversion", map[string]interface{}{
		"KubernetesData": m.Data.Kubernetes,
	})

	// Needing to get the Kubernetes configuration as an obj
	var kubernetesConfig KubernetesConfigModel
	diags := m.Data.Kubernetes.As(ctx, &kubernetesConfig, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		for _, d := range diags {
			tflog.Error(ctx, "Kubernetes config conversion error", map[string]interface{}{
				"summary": d.Summary(),
				"detail":  d.Detail(),
			})
		}
		return nil, fmt.Errorf("configuration error: unable to extract Kubernetes config")
	}
	// Check ConfigPath
	if !kubernetesConfig.ConfigPath.IsNull() {
		if v := kubernetesConfig.ConfigPath.ValueString(); v != "" {
			configPaths = []string{v}
		}
	}
	if !kubernetesConfig.ConfigPaths.IsNull() {
		additionalPaths := expandStringSlice(kubernetesConfig.ConfigPaths.Elements())
		configPaths = append(configPaths, additionalPaths...)
	}
	if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		configPaths = filepath.SplitList(v)
	}
	tflog.Debug(ctx, "Initial configPaths", map[string]interface{}{"configPaths": configPaths})

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				tflog.Error(ctx, "Error expanding home directory", map[string]interface{}{
					"path":  p,
					"error": err,
				})
				return nil, err
			}
			expandedPaths = append(expandedPaths, path)
		}
		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		// Check ConfigContext
		if !kubernetesConfig.ConfigContext.IsNull() {
			overrides.CurrentContext = kubernetesConfig.ConfigContext.ValueString()
		}
		if !kubernetesConfig.ConfigContextAuthInfo.IsNull() {
			overrides.Context.AuthInfo = kubernetesConfig.ConfigContextAuthInfo.ValueString()
		}
		if !kubernetesConfig.ConfigContextCluster.IsNull() {
			overrides.Context.Cluster = kubernetesConfig.ConfigContextCluster.ValueString()
		}
	}

	// Check and assign remaining fields
	if !kubernetesConfig.Insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = kubernetesConfig.Insecure.ValueBool()
	}
	if !kubernetesConfig.TLSServerName.IsNull() {
		overrides.ClusterInfo.TLSServerName = kubernetesConfig.TLSServerName.ValueString()
	}
	if !kubernetesConfig.ClusterCACertificate.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(kubernetesConfig.ClusterCACertificate.ValueString())
	}
	if !kubernetesConfig.ClientCertificate.IsNull() {
		overrides.AuthInfo.ClientCertificateData = []byte(kubernetesConfig.ClientCertificate.ValueString())
	}
	if !kubernetesConfig.Host.IsNull() && kubernetesConfig.Host.ValueString() != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(kubernetesConfig.Host.ValueString(), "", schema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, err
		}
		overrides.ClusterInfo.Server = host.String()
	}

	if !kubernetesConfig.Username.IsNull() {
		overrides.AuthInfo.Username = kubernetesConfig.Username.ValueString()
	}
	if !kubernetesConfig.Password.IsNull() {
		overrides.AuthInfo.Password = kubernetesConfig.Password.ValueString()
	}
	if !kubernetesConfig.ClientKey.IsNull() {
		overrides.AuthInfo.ClientKeyData = []byte(kubernetesConfig.ClientKey.ValueString())
	}
	if !kubernetesConfig.Token.IsNull() {
		overrides.AuthInfo.Token = kubernetesConfig.Token.ValueString()
	}
	if !kubernetesConfig.ProxyURL.IsNull() {
		overrides.ClusterDefaults.ProxyURL = kubernetesConfig.ProxyURL.ValueString()
	}

	if kubernetesConfig.Exec != nil {
		execConfig := kubernetesConfig.Exec
		if !execConfig.APIVersion.IsNull() && !execConfig.Command.IsNull() {
			args := []string{}
			if !execConfig.Args.IsNull() && !execConfig.Args.IsUnknown() {
				args = expandStringSlice(execConfig.Args.Elements())
			}

			env := []clientcmdapi.ExecEnvVar{}
			if !execConfig.Env.IsNull() && !execConfig.Env.IsUnknown() {
				for k, v := range execConfig.Env.Elements() {
					env = append(env, clientcmdapi.ExecEnvVar{
						Name:  k,
						Value: v.(types.String).ValueString(),
					})
				}
			}

			overrides.AuthInfo.Exec = &clientcmdapi.ExecConfig{
				APIVersion:      execConfig.APIVersion.ValueString(),
				Command:         execConfig.Command.ValueString(),
				Args:            args,
				Env:             env,
				InteractiveMode: clientcmdapi.IfAvailableExecInteractiveMode,
			}
		}
	}

	overrides.Context.Namespace = "default"
	if namespace != "" {
		overrides.Context.Namespace = namespace
	}

	burstLimit := int(m.Data.BurstLimit.ValueInt64())
	qps := float32(m.Data.QPS.ValueFloat64())
	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		return nil, fmt.Errorf("failed to initialize kubernetes config")
	}
	tflog.Info(ctx, "Successfully initialized kubernetes config")
	return &KubeConfig{ClientConfig: client, Burst: burstLimit, QPS: qps}, nil
}

func expandStringSlice(input []attr.Value) []string {
	result := make([]string, len(input))
	for i, v := range input {
		result[i] = v.(types.String).ValueString()
	}
	return result
}
