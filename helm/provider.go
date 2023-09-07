package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &helmProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &helmProvider{
			version: version,
		}
	}
}

// hashicupsProvider is the provider implementation.
type helmProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *helmProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "helm"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *helmProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Blocks: map[string]schema.Block{
			"kubernetes": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							Optional:    true,
							Description: "The hostname (in form of URI) of Kubernetes master.",
						},
						"username": schema.StringAttribute{
							Optional:    true,
							Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
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
							ElementType: types.StringType,
							Optional:    true,
							Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
						},
						"config_path": schema.StringAttribute{
							Optional:    true,
							Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
						},
						"config_context": schema.StringAttribute{
							Optional: true,
						},
						"config_context_auth_info": schema.StringAttribute{
							Optional:    true,
							Description: "",
						},
						"config_context_cluster": schema.StringAttribute{
							Optional:    true,
							Description: "",
						},
						"token": schema.StringAttribute{
							Optional:    true,
							Description: "Token to authenticate an service account",
						},
						"proxy_url": schema.StringAttribute{
							Optional:    true,
							Description: "URL to the proxy to be used for all API requests",
						},
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URI for HashiCups API. May also be provided via HASHICUPS_HOST environment variable.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username for HashiCups API. May also be provided via HASHICUPS_USERNAME environment variable.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for HashiCups API. May also be provided via HASHICUPS_PASSWORD environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *helmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	test := path.Path{
		steps: path.PathSteps,
	}

	req.Config.Schema.GetBlocks()
}

// DataSources defines the data sources implemented in the provider.
func (p *helmProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *helmProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
