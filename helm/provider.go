package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"helm.sh/helm/pkg/storage/driver"
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
						"exec": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"api_version": schema.StringAttribute{
										Required:   true,
										Validators: []validator.String{versionValidator{}},
									},
									"command": schema.StringAttribute{
										Required: true,
									},
									"env": schema.MapAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
									"args": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
								},
							},
							Optional: true,
						},
					},
				},
			},
		},

		Attributes: map[string]schema.Attribute{
			"debug": schema.BoolAttribute{
				Optional:    true,
				Description: "Debug indicates whether or not Helm is running in Debug mode.",
			},
			"plugins_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path to the helm plugins directory",
			},
			"registry_config_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path to the registry config file",
			},
			"repository_config_path": schema.StringAttribute{
				Optional:    true,
				Description: "The path to the file containing repository names and URLs",
			},
			"repository_cache": schema.StringAttribute{
				Optional:    true,
				Description: "The path to the file containing cached repository indexes",
			},
			"helm_driver": schema.StringAttribute{
				Optional:    true,
				Description: "The backend storage driver. Values are: configmap, secret, memory, sql",
				Validators: []validator.String{stringvalidator.AtLeastOneOf(path.Expressions{
					path.MatchRoot(strings.ToLower(driver.MemoryDriverName)),
					path.MatchRoot(strings.ToLower(driver.ConfigMapsDriverName)),
					path.MatchRoot(strings.ToLower(driver.SecretsDriverName)),
					path.MatchRoot(strings.ToLower(driver.SQLDriverName))}),
				},
			},
			"burst_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Helm burst limit. Increase this if you have a cluster with many CRDs",
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *helmProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

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

type versionValidator struct{}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v versionValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("validates whether api_version is alphav1")
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v versionValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("validates whether api_version is alphav1")
}

// Validate runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v versionValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if req.ConfigValue.ValueString() == "client.authentication.k8s.io/v1alpha1" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"v1alpha1 of the client authentication API has been removed, use v1beta1 or above",
			"v1alpha1 of the client authentication API is removed in Kubernetes client versions 1.24 and above. You may need to update your exec plugin to use the latest version.")

		return
	}
}
