package helm

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"

	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	memcached "k8s.io/client-go/discovery/cached/memory"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeConfig is a RESTClientGetter interface implementation
type KubeConfig struct {
	ClientConfig clientcmd.ClientConfig

	sync.Mutex
}

// ToRESTConfig implemented interface method
func (k *KubeConfig) ToRESTConfig() (*rest.Config, error) {
	config, err := k.ToRawKubeConfigLoader().ClientConfig()
	return config, err
}

// ToDiscoveryClient implemented interface method
func (k *KubeConfig) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := k.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	// The more groups you have, the more discovery requests you need to make.
	// given 25 groups (our groups + a few custom resources) with one-ish version each, discovery needs to make 50 requests
	// double it just so we don't end up here again for a while.  This config is only used for discovery.
	config.Burst = 100

	return memcached.NewMemCacheClient(discovery.NewDiscoveryClientForConfigOrDie(config)), nil
}

// ToRESTMapper implemented interface method
func (k *KubeConfig) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := k.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

// ToRawKubeConfigLoader implemented interface method
func (k *KubeConfig) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return k.ClientConfig
}

func newKubeConfig(configData *schema.ResourceData, namespace *string) (*KubeConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if v, ok := k8sGetOk(configData, "config_path"); ok && v != "" {
		configPaths = []string{v.(string)}
	} else if v, ok := k8sGetOk(configData, "config_paths"); ok {
		for _, p := range v.([]interface{}) {
			configPaths = append(configPaths, p.(string))
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		// NOTE we have to do this here because the schema
		// does not yet allow you to set a default for a TypeList
		configPaths = filepath.SplitList(v)
	}

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

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctx, ctxOk := k8sGetOk(configData, "config_context")
		authInfo, authInfoOk := k8sGetOk(configData, "config_context_auth_info")
		cluster, clusterOk := k8sGetOk(configData, "config_context_cluster")
		if ctxOk || authInfoOk || clusterOk {
			if ctxOk {
				overrides.CurrentContext = ctx.(string)
				log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoOk {
				overrides.Context.AuthInfo = authInfo.(string)
			}
			if clusterOk {
				overrides.Context.Cluster = cluster.(string)
			}
			log.Printf("[DEBUG] Using overidden context: %#v", overrides.Context)
		}
	}

	// Overriding with static configuration
	if v, ok := k8sGetOk(configData, "insecure"); ok {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := k8sGetOk(configData, "cluster_ca_certificate"); ok {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(configData, "client_certificate"); ok {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(configData, "host"); ok {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(v.(string), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, err
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v, ok := k8sGetOk(configData, "username"); ok {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := k8sGetOk(configData, "password"); ok {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := k8sGetOk(configData, "client_key"); ok {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(configData, "token"); ok {
		overrides.AuthInfo.Token = v.(string)
	}

	if v, ok := k8sGetOk(configData, "exec"); ok {
		exec := &clientcmdapi.ExecConfig{}
		if spec, ok := v.([]interface{})[0].(map[string]interface{}); ok {
			exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
			exec.APIVersion = spec["api_version"].(string)
			exec.Command = spec["command"].(string)
			exec.Args = expandStringSlice(spec["args"].([]interface{}))
			for kk, vv := range spec["env"].(map[string]interface{}) {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.(string)})
			}
		} else {
			log.Printf("[ERROR] Failed to parse exec")
			return nil, fmt.Errorf("failed to parse exec")
		}
		overrides.AuthInfo.Exec = exec
	}

	overrides.Context.Namespace = "default"

	if namespace != nil {
		overrides.Context.Namespace = *namespace
	}

	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
		log.Printf("[ERROR] Failed to initialize kubernetes config")
		return nil, nil
	}
	log.Printf("[INFO] Successfully initialized kubernetes config")

	return &KubeConfig{ClientConfig: client}, nil
}
