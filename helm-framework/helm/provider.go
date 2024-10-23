package helm

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var _ provider.Provider = &HelmProvider{}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &HelmProvider{}
	}
}

type Meta struct {
	providerData   *HelmProvider
	Data           *HelmProviderModel
	Settings       *cli.EnvSettings
	RegistryClient *registry.Client
	HelmDriver     string
	// Experimental feature toggles
	Experiments map[string]bool
	Mutex       sync.Mutex
}

// Models for our provider helm block
type HelmProviderModel struct {
	Debug                types.Bool   `tfsdk:"debug"`
	PluginsPath          types.String `tfsdk:"plugins_path"`
	RegistryConfigPath   types.String `tfsdk:"registry_config_path"`
	RepositoryConfigPath types.String `tfsdk:"repository_config_path"`
	RepositoryCache      types.String `tfsdk:"repository_cache"`
	HelmDriver           types.String `tfsdk:"helm_driver"`
	BurstLimit           types.Int64  `tfsdk:"burst_limit"`
	Kubernetes           types.List   `tfsdk:"kubernetes"`
	Registries           types.List   `tfsdk:"registries"`
	Experiments          types.List   `tfsdk:"experiments"`
}

type ExperimentsConfigModel struct {
	Manifest types.Bool `tfsdk:"manifest"`
}

type RegistryConfigModel struct {
	URL      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type KubernetesConfigModel struct {
	Host                  types.String `tfsdk:"host"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Insecure              types.Bool   `tfsdk:"insecure"`
	TLSServerName         types.String `tfsdk:"tls_server_name"`
	ClientCertificate     types.String `tfsdk:"client_certificate"`
	ClientKey             types.String `tfsdk:"client_key"`
	ClusterCACertificate  types.String `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.List   `tfsdk:"config_paths"`
	ConfigPath            types.String `tfsdk:"config_path"`
	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`
	Token                 types.String `tfsdk:"token"`
	ProxyURL              types.String `tfsdk:"proxy_url"`
	// Exec                  types.List   `tfsdk:"exec"`
}

type ExecConfigModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Command    types.String `tfsdk:"command"`
	Env        types.Map    `tfsdk:"env"`
	Args       types.List   `tfsdk:"args"`
}

type HelmProvider struct {
	meta *Meta
}

func (p *HelmProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "helm"
}

// ///////////////////////            	START OF SCHEMA CREATION               ///////////////////////////////
// Defines attributes that are avaiable in the provider
func (p *HelmProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schema to define attributes that are available in the provider",
		Attributes: map[string]schema.Attribute{
			"debug": schema.BoolAttribute{
				Description: "Debug indicates whether or not Helm is running in Debug mode.",
				Optional:    true,
			},
			"plugins_path": schema.StringAttribute{
				Description: "The path to the helm plugins directory",
				Optional:    true,
			},
			"registry_config_path": schema.StringAttribute{
				Description: "The path to the registry config file",
				Optional:    true,
			},
			"repository_config_path": schema.StringAttribute{
				Description: "The path to the file containing repository names and URLs",
				Optional:    true,
			},
			"repository_cache": schema.StringAttribute{
				Description: "The path to the file containing cached repository indexes",
				Optional:    true,
			},
			"helm_driver": schema.StringAttribute{
				Description: "The backend storage driver. Values are: configmap, secret, memory, sql",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(strings.ToLower(driver.MemoryDriverName),
						strings.ToLower(driver.ConfigMapsDriverName),
						strings.ToLower(driver.SecretsDriverName),
						strings.ToLower(driver.SQLDriverName)),
				},
			},

			"burst_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Helm burst limit. Increase this if you have a cluster with many CRDs",
			},
			"kubernetes": schema.ListNestedAttribute{
				Optional: true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				Description: "Kubernetes Configuration",
				NestedObject: schema.NestedAttributeObject{
					Attributes: kubernetesResourceSchema(),
				},
			},
			"registries": schema.ListNestedAttribute{
				Optional:    true,
				Description: "RegistryClient configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: registriesResourceSchema(),
				},
			},
			"experiments": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Enable and disable experimental features.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: experimentsSchema(),
				},
			},
		},
	}
}

func experimentsSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"manifest": schema.BoolAttribute{
			Optional:    true,
			Description: "Enable full diff by storing the rendered manifest in the state.",
		},
	}
}

func registriesResourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Required:    true,
			Description: "OCI URL in form of oci://host:port or oci://host",
		},
		"username": schema.StringAttribute{
			Required:    true,
			Description: "The username to use for the OCI HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"password": schema.StringAttribute{
			Required:    true,
			Description: "The password to use for the OCI HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
	}
}

func kubernetesResourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Optional:    true,
			Description: "The hostname (in form of URI) of kubernetes master",
		},
		"username": schema.StringAttribute{
			Optional:    true,
			Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint",
		},
		"password": schema.StringAttribute{
			Optional:    true,
			Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"insecure": schema.BoolAttribute{
			Optional:    true,
			Description: "Whether server should be accessed without verifying the TLS certificate.",
		},
		"tls_server_name": schema.StringAttribute{
			Optional:    true,
			Description: "Server name passed to the server for SNI and is used in the client to check server certificates against.",
		},
		"client_certificate": schema.StringAttribute{
			Optional:    true,
			Description: "PEM-encoded client certificate for TLS authentication.",
		},
		"client_key": schema.StringAttribute{
			Optional:    true,
			Description: "PEM-encoded client certificate key for TLS authentication.",
		},
		"cluster_ca_certificate": schema.StringAttribute{
			Optional:    true,
			Description: "PEM-encoded root certificates bundle for TLS authentication.",
		},
		"config_paths": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
		},
		"config_path": schema.StringAttribute{
			Optional:    true,
			Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
			Validators: []validator.String{
				stringvalidator.ConflictsWith(
					path.Root("kubernetes").AtListIndex(0).AtName("config_paths").Expression(),
				),
			},
		},
		"config_context": schema.StringAttribute{
			Optional:    true,
			Description: "Context to use for Kubernetes config.",
		},
		"config_context_auth_info": schema.StringAttribute{
			Optional: true,
			// TODO REFERENCE THE DEFAULT DOCUMENTATI
			Description: "AuthInfo to use for Kubernetes config context.",
		},
		"config_context_cluster": schema.StringAttribute{
			Optional:    true,
			Description: "Cluster to use for Kubernetes config context.",
		},
		"token": schema.StringAttribute{
			Optional:    true,
			Description: "Token to authenticate a service account.",
		},
		"proxy_url": schema.StringAttribute{
			Optional:    true,
			Description: "URL to the proxy to be used for all API requests.",
		},
		//  "exec": schema.ListNestedAttribute{
		//  	Optional: true,
		//  	Validators: []validator.List{
		//  		listvalidator.SizeAtMost(1),
		//  	},
		//  	Description: "Exec configuration for Kubernetes authentication",
		//  	NestedObject: schema.NestedAttributeObject{
		//  		Attributes: execSchema(),
		//  	},
		//  },
	}
}

func execSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_version": schema.StringAttribute{
			Required:    true,
			Description: "API version for the exec plugin.",
		},
		"command": schema.StringAttribute{
			Required:    true,
			Description: "Command to run for Kubernetes exec plugin",
		},
		"env": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Environment variables for the exec plugin",
		},
		"args": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Arguments for the exec plugin",
		},
	}
}

/////////////////////     					END OF SCHEMA CREATION           ///////////////////////////////

// Setting up the provider, anything we need to get the provider running, probbaly authentication. like the api
func (p *HelmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	pluginsPath := os.Getenv("HELM_PLUGINS_PATH")
	registryConfigPath := os.Getenv("HELM_REGISTRY_CONFIG_PATH")
	repositoryConfigPath := os.Getenv("HELM_REPOSITORY_CONFIG_PATH")
	repositoryCache := os.Getenv("HELM_REPOSITORY_CACHE")
	helmDriver := os.Getenv("HELM_DRIVER")
	burstLimitStr := os.Getenv("HELM_BURST_LIMIT")
	kubeHost := os.Getenv("KUBE_HOST")
	kubeUser := os.Getenv("KUBE_USER")
	kubePassword := os.Getenv("KUBE_PASSWORD")
	kubeInsecureStr := os.Getenv("KUBE_INSECURE")
	kubeTLSServerName := os.Getenv("KUBE_TLS_SERVER_NAME")
	kubeClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA")
	kubeClientKey := os.Getenv("KUBE_CLIENT_KEY_DATA")
	kubeCaCert := os.Getenv("KUBE_CLUSTER_CA_CERT_DATA")
	kubeConfigPaths := os.Getenv("KUBE_CONFIG_PATHS")
	kubeConfigPath := os.Getenv("KUBE_CONFIG_PATH")
	kubeConfigContext := os.Getenv("KUBE_CTX")
	kubeConfigContextAuthInfo := os.Getenv("KUBE_CTX_AUTH_INFO")
	kubeConfigContextCluster := os.Getenv("KUBE_CTX_CLUSTER")
	kubeToken := os.Getenv("KUBE_TOKEN")
	kubeProxy := os.Getenv("KUBE_PROXY")

	// Initialize the HelmProviderModel with values from the config
	var config HelmProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.PluginsPath.IsNull() {
		pluginsPath = config.PluginsPath.ValueString()
	}
	if !config.RegistryConfigPath.IsNull() {
		registryConfigPath = config.RegistryConfigPath.ValueString()
	}
	if !config.RepositoryConfigPath.IsNull() {
		repositoryConfigPath = config.RepositoryConfigPath.ValueString()
	}
	if !config.RepositoryCache.IsNull() {
		repositoryCache = config.RepositoryCache.ValueString()
	}
	if !config.HelmDriver.IsNull() {
		helmDriver = config.HelmDriver.ValueString()
	}
	var burstLimit int64
	if burstLimitStr != "" {
		var err error
		burstLimit, err = strconv.ParseInt(burstLimitStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid burst limit",
				fmt.Sprintf("Invalid burst limit value: %s", burstLimitStr),
			)
			return
		}
	}
	if !config.BurstLimit.IsNull() {
		burstLimit = config.BurstLimit.ValueInt64()
	}
	var kubeInsecure bool
	if kubeInsecureStr != "" {
		var err error
		kubeInsecure, err = strconv.ParseBool(kubeInsecureStr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid insecure value",
				fmt.Sprintf("Invalid insecure value: %s", kubeInsecureStr),
			)
			return
		}
	}

	var kubernetesConfig KubernetesConfigModel
	if !config.Kubernetes.IsNull() && !config.Kubernetes.IsUnknown() {
		var kubernetesConfigs []KubernetesConfigModel
		diags := config.Kubernetes.ElementsAs(ctx, &kubernetesConfigs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(kubernetesConfigs) > 0 {
			kubernetesConfig = kubernetesConfigs[0]
		}
	}

	if !kubernetesConfig.Insecure.IsNull() {
		kubeInsecure = kubernetesConfig.Insecure.ValueBool()
	}
	var kubeConfigPathsList []attr.Value
	if !kubernetesConfig.Host.IsNull() {
		kubeHost = kubernetesConfig.Host.ValueString()
	}
	if !kubernetesConfig.Username.IsNull() {
		kubeUser = kubernetesConfig.Username.ValueString()
	}
	if !kubernetesConfig.Password.IsNull() {
		kubePassword = kubernetesConfig.Password.ValueString()
	}
	if !kubernetesConfig.TLSServerName.IsNull() {
		kubeTLSServerName = kubernetesConfig.TLSServerName.ValueString()
	}
	if !kubernetesConfig.ClientCertificate.IsNull() {
		kubeClientCert = kubernetesConfig.ClientCertificate.ValueString()
	}
	if !kubernetesConfig.ClientKey.IsNull() {
		kubeClientKey = kubernetesConfig.ClientKey.ValueString()
	}
	if !kubernetesConfig.ClusterCACertificate.IsNull() {
		kubeCaCert = kubernetesConfig.ClusterCACertificate.ValueString()
	}
	if kubeConfigPaths != "" {
		for _, path := range filepath.SplitList(kubeConfigPaths) {
			kubeConfigPathsList = append(kubeConfigPathsList, types.StringValue(path))
		}
	}
	if !kubernetesConfig.ConfigPaths.IsNull() {
		var paths []string
		diags = kubernetesConfig.ConfigPaths.ElementsAs(ctx, &paths, false)
		resp.Diagnostics.Append(diags...)
		for _, path := range paths {
			kubeConfigPathsList = append(kubeConfigPathsList, types.StringValue(path))
		}
	}
	if !kubernetesConfig.ConfigPath.IsNull() {
		kubeConfigPath = kubernetesConfig.ConfigPath.ValueString()
	}
	if !kubernetesConfig.ConfigContext.IsNull() {
		kubeConfigContext = kubernetesConfig.ConfigContext.ValueString()
	}
	if !kubernetesConfig.ConfigContextAuthInfo.IsNull() {
		kubeConfigContextAuthInfo = kubernetesConfig.ConfigContextAuthInfo.ValueString()
	}
	if !kubernetesConfig.ConfigContextCluster.IsNull() {
		kubeConfigContextCluster = kubernetesConfig.ConfigContextCluster.ValueString()
	}
	if !kubernetesConfig.Token.IsNull() {
		kubeToken = kubernetesConfig.Token.ValueString()
	}
	if !kubernetesConfig.ProxyURL.IsNull() {
		kubeProxy = kubernetesConfig.ProxyURL.ValueString()
	}
	tflog.Debug(ctx, "Config values after overrides", map[string]interface{}{
		"config": config,
	})
	debug := os.Getenv("HELM_DEBUG") == "true" || config.Debug.ValueBool()
	settings := cli.New()
	settings.Debug = debug
	if pluginsPath != "" {
		settings.PluginsDirectory = pluginsPath
	}
	if registryConfigPath != "" {
		settings.RegistryConfig = registryConfigPath
	}
	if repositoryConfigPath != "" {
		settings.RepositoryConfig = repositoryConfigPath
	}
	if repositoryCache != "" {
		settings.RepositoryCache = repositoryCache
	}
	tflog.Debug(ctx, "Helm settings initialized", map[string]interface{}{
		"settings": settings,
	})
	kubeConfigPathsListValue, diags := types.ListValue(types.StringType, kubeConfigPathsList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var experimentsConfig ExperimentsConfigModel
	if !config.Experiments.IsNull() && !config.Experiments.IsUnknown() {
		var experimentsConfigs []ExperimentsConfigModel
		diags := config.Experiments.ElementsAs(ctx, &experimentsConfigs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(experimentsConfigs) > 0 {
			experimentsConfig = experimentsConfigs[0]
		}
	}

	manifestExperiment := false
	if !experimentsConfig.Manifest.IsNull() {
		manifestExperiment = experimentsConfig.Manifest.ValueBool()
	}

	// Create a list of KubernetesConfigModel
	kubernetesConfigList := []attr.Value{
		types.ObjectValueMust(map[string]attr.Type{
			"host":                     types.StringType,
			"username":                 types.StringType,
			"password":                 types.StringType,
			"insecure":                 types.BoolType,
			"tls_server_name":          types.StringType,
			"client_certificate":       types.StringType,
			"client_key":               types.StringType,
			"cluster_ca_certificate":   types.StringType,
			"config_paths":             types.ListType{ElemType: types.StringType},
			"config_path":              types.StringType,
			"config_context":           types.StringType,
			"config_context_auth_info": types.StringType,
			"config_context_cluster":   types.StringType,
			"token":                    types.StringType,
			"proxy_url":                types.StringType,
		}, map[string]attr.Value{
			"host":                     types.StringValue(kubeHost),
			"username":                 types.StringValue(kubeUser),
			"password":                 types.StringValue(kubePassword),
			"insecure":                 types.BoolValue(kubeInsecure),
			"tls_server_name":          types.StringValue(kubeTLSServerName),
			"client_certificate":       types.StringValue(kubeClientCert),
			"client_key":               types.StringValue(kubeClientKey),
			"cluster_ca_certificate":   types.StringValue(kubeCaCert),
			"config_paths":             kubeConfigPathsListValue,
			"config_path":              types.StringValue(kubeConfigPath),
			"config_context":           types.StringValue(kubeConfigContext),
			"config_context_auth_info": types.StringValue(kubeConfigContextAuthInfo),
			"config_context_cluster":   types.StringValue(kubeConfigContextCluster),
			"token":                    types.StringValue(kubeToken),
			"proxy_url":                types.StringValue(kubeProxy),
		}),
	}
	kubernetesConfigListValue, diags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"host":                     types.StringType,
			"username":                 types.StringType,
			"password":                 types.StringType,
			"insecure":                 types.BoolType,
			"tls_server_name":          types.StringType,
			"client_certificate":       types.StringType,
			"client_key":               types.StringType,
			"cluster_ca_certificate":   types.StringType,
			"config_paths":             types.ListType{ElemType: types.StringType},
			"config_path":              types.StringType,
			"config_context":           types.StringType,
			"config_context_auth_info": types.StringType,
			"config_context_cluster":   types.StringType,
			"token":                    types.StringType,
			"proxy_url":                types.StringType,
		},
	}, kubernetesConfigList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create a list of ExperimentsConfigModel
	experimentsConfigList := []attr.Value{
		types.ObjectValueMust(map[string]attr.Type{
			"manifest": types.BoolType,
		}, map[string]attr.Value{
			"manifest": types.BoolValue(manifestExperiment),
		}),
	}
	experimentsConfigListValue, diags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"manifest": types.BoolType,
		},
	}, experimentsConfigList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta := &Meta{
		Data: &HelmProviderModel{
			Debug:                types.BoolValue(debug),
			PluginsPath:          types.StringValue(pluginsPath),
			RegistryConfigPath:   types.StringValue(registryConfigPath),
			RepositoryConfigPath: types.StringValue(repositoryConfigPath),
			RepositoryCache:      types.StringValue(repositoryCache),
			HelmDriver:           types.StringValue(helmDriver),
			BurstLimit:           types.Int64Value(burstLimit),
			Kubernetes:           kubernetesConfigListValue,
			Experiments:          experimentsConfigListValue,
		},
		Settings:   settings,
		HelmDriver: helmDriver,
		Experiments: map[string]bool{
			"manifest": manifestExperiment,
		},
	}
	registryClient, err := registry.NewClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Registry client initialization failed",
			fmt.Sprintf("Unable to create Helm registry client: %s", err),
		)
		return
	}

	meta.RegistryClient = registryClient
	if !config.Registries.IsUnknown() {
		var registryConfigs []RegistryConfigModel
		diags := config.Registries.ElementsAs(ctx, &registryConfigs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, r := range registryConfigs {
			if r.URL.IsNull() || r.Username.IsNull() || r.Password.IsNull() {
				resp.Diagnostics.AddError(
					"OCI Registry login failed",
					"Registry URL, Username, or Password is null",
				)
				return
			}

			err := OCIRegistryPerformLogin(ctx, meta, meta.RegistryClient, r.URL.ValueString(), r.Username.ValueString(), r.Password.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"OCI Registry login failed",
					err.Error(),
				)
				return
			}
		}
	} else {
		tflog.Debug(ctx, "No registry configurations found")
	}
	resp.DataSourceData = meta
	resp.ResourceData = meta

	tflog.Debug(ctx, "Configure method completed successfully")
}

func (p *HelmProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHelmTemplate,
	}
}

func (p *HelmProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHelmRelease,
	}
}

func OCIRegistryLogin(ctx context.Context, meta *Meta, actionConfig *action.Configuration, registryClient *registry.Client, repository, chartName, username, password string) diag.Diagnostics {
	var diags diag.Diagnostics

	actionConfig.RegistryClient = registryClient

	var ociURL string
	if registry.IsOCI(repository) {
		ociURL = repository
	} else if registry.IsOCI(chartName) {
		ociURL = chartName
	}

	if ociURL == "" {
		return diags
	}

	if username != "" && password != "" {
		err := OCIRegistryPerformLogin(ctx, meta, registryClient, ociURL, username, password)
		if err != nil {
			diags.AddError(
				"OCI Registry Login Failed",
				fmt.Sprintf("Failed to log in to OCI registry %q: %s", ociURL, err.Error()),
			)
		}
	}

	return diags
}

// registryClient = client used to comm with the registry, oci urls, un, and pw used for authentication
func OCIRegistryPerformLogin(ctx context.Context, meta *Meta, registryClient *registry.Client, ociURL, username, password string) error {

	loggedInOCIRegistries := make(map[string]string)
	// getting the oci url, and extracting the host.
	u, err := url.Parse(ociURL)
	if err != nil {
		return fmt.Errorf("could not parse OCI registry URL: %v", err)
	}
	meta.Mutex.Lock()
	defer meta.Mutex.Unlock()
	if _, ok := loggedInOCIRegistries[u.Host]; ok {
		tflog.Info(ctx, fmt.Sprintf("Already logged into OCI registry %q", u.Host))
		return nil
	}
	// Now we perform the login, with the provided username and password by calling the login method
	err = registryClient.Login(u.Host, registry.LoginOptBasicAuth(username, password))
	if err != nil {
		return fmt.Errorf("could not login to OCI registry %q: %v", u.Host, err)
	}
	loggedInOCIRegistries[u.Host] = ""
	tflog.Info(ctx, fmt.Sprintf("Logged into OCI registry %q", u.Host))
	return nil
}

// GetHelmConfiguration retrieves the Helm configuration for a given namespace
func (m *Meta) GetHelmConfiguration(ctx context.Context, namespace string) (*action.Configuration, error) {
	if m == nil {
		tflog.Error(ctx, "Meta is nil")
		return nil, fmt.Errorf("Meta is nil")
	}

	tflog.Info(context.Background(), "[INFO] GetHelmConfiguration start")
	actionConfig := new(action.Configuration)
	kc, err := m.NewKubeConfig(ctx, namespace)
	if err != nil {
		return nil, err
	}
	if err := actionConfig.Init(kc, namespace, m.HelmDriver, func(format string, v ...interface{}) {
		tflog.Info(context.Background(), fmt.Sprintf(format, v...))
	}); err != nil {
		return nil, err
	}
	tflog.Info(context.Background(), "[INFO] GetHelmConfiguration success")
	// returning the initializing action.Configuration object
	return actionConfig, nil
}
