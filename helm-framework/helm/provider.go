package helm

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	//"fmt"
	//"log"
	//"net/url"

	"os"
	//"strconv"
	"strings"

	//"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	//"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	//"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	//"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	//"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	//"github.com/hashicorp/terraform-plugin-framework/provider/validators"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	//"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	//"helm.sh/helm/v3/pkg/action"

	//"helm.sh/helm/v3/pkg/helmpath"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/storage/driver"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/plugin/pkg/client/auth"
)

var _ provider.Provider = &HelmProvider{}

// New instances of our provider. // provider initialization
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HelmProvider{
			//typeName: "helm",
			version: version,
		}
	}
}

type Meta struct {
	Data           *HelmProviderModel
	Settings       *cli.EnvSettings
	RegistryClient *registry.Client
	HelmDriver     string
	// used to lock some operations
	sync.Mutex
	// Experimental feature toggles
	Experiments map[string]bool
}

// Models for our provider helm block
// Change to pascalcase for the fields
type HelmProviderModel struct {
	Debug                types.Bool              `tfsdk:"debug"`
	PluginsPath          types.String            `tfsdk:"plugins_path"`
	RegistryConfigPath   types.String            `tfsdk:"registry_config_path"`
	RepositoryConfigPath types.String            `tfsdk:"repository_config_path"`
	RepositoryCache      types.String            `tfsdk:"repository_cache"`
	HelmDriver           types.String            `tfsdk:"helm_driver"`
	BurstLimit           types.Int64             `tfsdk:"burst_limit"`
	Kubernetes           *KubernetesConfigModel  `tfsdk:"kubernetes"`
	Registry             []RegistryConfigModel   `tfsdk:"registry"`
	Experiments          *ExperimentsConfigModel `tfsdk:"experiments"`
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
	Host                  types.String        `tfsdk:"host"`
	Username              types.String        `tfsdk:"username"`
	Password              types.String        `tfsdk:"password"`
	Insecure              types.Bool          `tfsdk:"insecure"`
	TlsServerName         types.String        `tfsdk:"tls_server_name"`
	ClientCertificate     types.String        `tfsdk:"client_certificate"`
	ClientKey             types.String        `tfsdk:"client_key"`
	ClusterCaCertificate  types.String        `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           basetypes.ListValue `tfsdk:"config_paths"`
	ConfigPath            types.String        `tfsdk:"config_path"`
	ConfigContext         types.String        `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String        `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String        `tfsdk:"config_context_cluster"`
	Token                 types.String        `tfsdk:"token"`
	ProxyUrl              types.String        `tfsdk:"proxy_url"`
	Exec                  *ExecConfigModel    `tfsdk:"exec"`
}

type ExecConfigModel struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Command    types.String `tfsdk:"command"`
	Env        types.Map    `tfsdk:"env"`
	Args       types.List   `tfsdk:"args"`
}

// Represents custom Terraform provider for helm
type HelmProvider struct {
	version string
}

func (p *HelmProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	// name of our provider,it will also be the name of our resources and data sources that we define in the provider
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
			// The registry attr has nested attr's, so I am using this approach for code simplicity
			// TODO CHANGE TO SINGLE NESTED
			"registry": schema.ListNestedAttribute{
				Optional:    true,
				Description: "RegistryClient configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: registryResourceSchema(),
				},
			},
			// The experiemnts attr has nested attr, so I am using this approach for code simplicity
			// TODO CHANGE TO SINGLE NESTED
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

// This func is for the experiments attr, due to it having nested attributes
func experimentsSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"manifest": schema.BoolAttribute{
			Optional:    true,
			Description: "Enable full diff by storing the rendered manifest in the state.",
		},
	}
}

// This func is for the registry attr, due to it having nested attributes
func registryResourceSchema() map[string]schema.Attribute {
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
			//ConflictsWith: []string{"kubernetes.0.config_paths"},
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
		// TODO, SINGLE NESTED BLOCK
		"exec": schema.ListNestedAttribute{
			Optional: true,
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
			},
			Description: "Exec configuration for Kubernetes authentication",
			NestedObject: schema.NestedAttributeObject{
				Attributes: execSchema(),
			},
		},
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
	fmt.Println("Starting Configure method")
	// Fetching environment variables, which will be the fallback values if they are not provided in the config
	debug := os.Getenv("HELM_DEBUG")
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
	kubeTlsServerName := os.Getenv("KUBE_TLS_SERVER_NAME")
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
	fmt.Println("Fetched environment variables")

	// Initializing the HelmProviderModel with values from the config
	var config HelmProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		fmt.Println("Error reading config")
		return
	}
	fmt.Println("Config values read from main.tf:", config)

	// Overriding environment variables if the configuration values are provided
	if !config.Debug.IsNull() {
		debug = fmt.Sprintf("%t", config.Debug.ValueBool())
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
	// Parsing burst limit from string to int64, due to the retrieval being a string
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
	// Parsing insecure boolean value, due to the retrieval being a string
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
	if config.Kubernetes != nil && !config.Kubernetes.Insecure.IsNull() {
		kubeInsecure = config.Kubernetes.Insecure.ValueBool()
	}
	// Overriding Kubernetes config values if they are provided
	var kubeConfigPathsList []attr.Value // Handling config paths list
	if config.Kubernetes != nil {
		if !config.Kubernetes.Host.IsNull() {
			kubeHost = config.Kubernetes.Host.ValueString()
		}
		if !config.Kubernetes.Username.IsNull() {
			kubeUser = config.Kubernetes.Username.ValueString()
		}
		if !config.Kubernetes.Password.IsNull() {
			kubePassword = config.Kubernetes.Password.ValueString()
		}
		if !config.Kubernetes.TlsServerName.IsNull() {
			kubeTlsServerName = config.Kubernetes.TlsServerName.ValueString()
		}
		if !config.Kubernetes.ClientCertificate.IsNull() {
			kubeClientCert = config.Kubernetes.ClientCertificate.ValueString()
		}
		if !config.Kubernetes.ClientKey.IsNull() {
			kubeClientKey = config.Kubernetes.ClientKey.ValueString()
		}
		if !config.Kubernetes.ClusterCaCertificate.IsNull() {
			kubeCaCert = config.Kubernetes.ClusterCaCertificate.ValueString()
		}
		if kubeConfigPaths != "" {
			for _, path := range strings.Split(kubeConfigPaths, ",") {
				kubeConfigPathsList = append(kubeConfigPathsList, types.StringValue(path))
			}
		}
		if !config.Kubernetes.ConfigPaths.IsNull() {
			var paths []string
			diags = config.Kubernetes.ConfigPaths.ElementsAs(ctx, &paths, false)
			resp.Diagnostics.Append(diags...)
			for _, path := range paths {
				kubeConfigPathsList = append(kubeConfigPathsList, types.StringValue(path))
			}
		}
		if !config.Kubernetes.ConfigPath.IsNull() {
			kubeConfigPath = config.Kubernetes.ConfigPath.ValueString()
		}
		if !config.Kubernetes.ConfigContext.IsNull() {
			kubeConfigContext = config.Kubernetes.ConfigContext.ValueString()
		}
		if !config.Kubernetes.ConfigContextAuthInfo.IsNull() {
			kubeConfigContextAuthInfo = config.Kubernetes.ConfigContextAuthInfo.ValueString()
		}
		if !config.Kubernetes.ConfigContextCluster.IsNull() {
			kubeConfigContextCluster = config.Kubernetes.ConfigContextCluster.ValueString()
		}
		if !config.Kubernetes.Token.IsNull() {
			kubeToken = config.Kubernetes.Token.ValueString()
		}
		if !config.Kubernetes.ProxyUrl.IsNull() {
			kubeProxy = config.Kubernetes.ProxyUrl.ValueString()
		}
	}
	fmt.Println("Config values after overrides:", config)
	// Initializing Helm CLI settings
	settings := cli.New()
	settings.Debug = debug == "true"
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
	fmt.Println("Helm settings initialized")
	// Converting kubeConfigPathsList to ListValue
	kubeConfigPathsListValue, diags := types.ListValue(types.StringType, kubeConfigPathsList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		fmt.Println("Error converting kubeConfigPathsList to ListValue")
		return
	}

	// Checking Experiments configuration
	manifestExperiment := false
	if config.Experiments != nil && !config.Experiments.Manifest.IsNull() {
		manifestExperiment = config.Experiments.Manifest.ValueBool()
	}

	// Creating the Meta object, with configuration and settings
	meta := &Meta{
		Data: &HelmProviderModel{
			Debug:                types.BoolValue(debug == "true"),
			PluginsPath:          types.StringValue(pluginsPath),
			RegistryConfigPath:   types.StringValue(registryConfigPath),
			RepositoryConfigPath: types.StringValue(repositoryConfigPath),
			RepositoryCache:      types.StringValue(repositoryCache),
			HelmDriver:           types.StringValue(helmDriver),
			BurstLimit:           types.Int64Value(burstLimit),
			Kubernetes: &KubernetesConfigModel{
				Host:                  types.StringValue(kubeHost),
				Username:              types.StringValue(kubeUser),
				Password:              types.StringValue(kubePassword),
				Insecure:              types.BoolValue(kubeInsecure),
				TlsServerName:         types.StringValue(kubeTlsServerName),
				ClientCertificate:     types.StringValue(kubeClientCert),
				ClientKey:             types.StringValue(kubeClientKey),
				ClusterCaCertificate:  types.StringValue(kubeCaCert),
				ConfigPaths:           kubeConfigPathsListValue,
				ConfigPath:            types.StringValue(kubeConfigPath),
				ConfigContext:         types.StringValue(kubeConfigContext),
				ConfigContextAuthInfo: types.StringValue(kubeConfigContextAuthInfo),
				ConfigContextCluster:  types.StringValue(kubeConfigContextCluster),
				Token:                 types.StringValue(kubeToken),
				ProxyUrl:              types.StringValue(kubeProxy),
			},
			Experiments: &ExperimentsConfigModel{
				Manifest: types.BoolValue(manifestExperiment),
			},
		},
		Settings:   settings,
		HelmDriver: helmDriver,
		Experiments: map[string]bool{
			"manifest": manifestExperiment,
		},
	}

	// Initializing registry client
	registryClient, err := registry.NewClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Registry client initialization failed",
			fmt.Sprintf("Unable to create Helm registry client: %s", err),
		)
		return
	}
	fmt.Println("Registry client initialized")

	meta.RegistryClient = registryClient

	// Performing OCI Registry Login for each registry config
	if config.Registry != nil {
		for _, r := range config.Registry {
			if r.URL.IsNull() || r.Username.IsNull() || r.Password.IsNull() {
				resp.Diagnostics.AddError(
					"OCI Registry login failed",
					"Registry URL, Username, or Password is null",
				)
				return
			}

			err := OCIRegistryPerformLogin(ctx, meta.RegistryClient, r.URL.ValueString(), r.Username.ValueString(), r.Password.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"OCI Registry login failed",
					err.Error(),
				)
				return
			}
			fmt.Println("OCI Registry login successful for", r.URL.ValueString())
		}
	} else {
		fmt.Println("No registry configurations found")
	}

	// Setting the meta object as the data source and resource data
	resp.DataSourceData = meta
	resp.ResourceData = meta

	fmt.Println("Configure method completed")
}

// Defining data sources that will be implemented by the provider
func (p *HelmProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Defining resources that will be implemented by the provider
func (p *HelmProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHelmReleaseResource,
	}
}

var (
	OCILoginMutex         sync.Mutex
	loggedInOCIRegistries = make(map[string]string)
)

// registryClient = client used to comm with the registry, oci urls, un, and pw used for authentication
func OCIRegistryPerformLogin(ctx context.Context, registryClient *registry.Client, ociURL, username, password string) error {
	// getting the oci url, and extracting the host.
	u, err := url.Parse(ociURL)
	if err != nil {
		return fmt.Errorf("could not parse OCI registry URL: %v", err)
	}
	// Making sure the login operations being performed are thread safe
	OCILoginMutex.Lock()
	defer OCILoginMutex.Unlock()

	//Checking for existing login
	// Checking the map for host existence
	if _, ok := loggedInOCIRegistries[u.Host]; ok {
		tflog.Info(ctx, fmt.Sprintf("Already logged into OCI registry %q", u.Host))
		return nil
	}
	// Now we perform the login, with the provided username and password by calling the login method
	err = registryClient.Login(u.Host, registry.LoginOptBasicAuth(username, password))
	if err != nil {
		return fmt.Errorf("could not login to OCI registry %q: %v", u.Host, err)
	}

	//If the login was succesfful, we mark it in our map by inputting the host
	loggedInOCIRegistries[u.Host] = ""
	// Just logging the successful login
	tflog.Info(ctx, fmt.Sprintf("Logged into OCI registry %q", u.Host))
	return nil
}

// GetHelmConfiguration retrieves the Helm configuration for a given namespace
func (m *Meta) GetHelmConfiguration(ctx context.Context, namespace string) (*action.Configuration, error) {
	m.Lock()
	defer m.Unlock()
	tflog.Info(context.Background(), "[INFO] GetHelmConfiguration start")
	actionConfig := new(action.Configuration)
	kc, err := m.newKubeConfig(ctx, namespace)
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
