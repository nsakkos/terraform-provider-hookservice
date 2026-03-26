package provider

import (
	"context"
	"os"

	"github.com/canonical/terraform-provider-hookservice/internal/client"
	"github.com/canonical/terraform-provider-hookservice/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/oauth2/clientcredentials"
)

var _ provider.Provider = &HookServiceProvider{}

// HookServiceProvider defines the provider implementation.
type HookServiceProvider struct {
	version string
}

// HookServiceProviderModel describes the provider data model.
type HookServiceProviderModel struct {
	Host         types.String `tfsdk:"host"`
	Token        types.String `tfsdk:"token"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	TokenURL     types.String `tfsdk:"token_url"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HookServiceProvider{
			version: version,
		}
	}
}

func (p *HookServiceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hookservice"
	resp.Version = p.version
}

func (p *HookServiceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing groups and access in the Canonical Hook Service.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The Hook Service API host URL (e.g. http://10.0.0.1:8000). " +
					"Can also be set with the HOOK_SERVICE_HOST environment variable.",
				Optional: true,
			},
			"token": schema.StringAttribute{
				Description: "Bearer token for API authentication. " +
					"Can also be set with the HOOK_SERVICE_TOKEN environment variable. " +
					"Mutually exclusive with client_id/client_secret/token_url.",
				Optional:  true,
				Sensitive: true,
			},
			"client_id": schema.StringAttribute{
				Description: "OAuth2 client ID for client credentials authentication. " +
					"Can also be set with the HOOK_SERVICE_CLIENT_ID environment variable. " +
					"Requires client_secret and token_url.",
				Optional: true,
			},
			"client_secret": schema.StringAttribute{
				Description: "OAuth2 client secret for client credentials authentication. " +
					"Can also be set with the HOOK_SERVICE_CLIENT_SECRET environment variable. " +
					"Requires client_id and token_url.",
				Optional:  true,
				Sensitive: true,
			},
			"token_url": schema.StringAttribute{
				Description: "OAuth2 token endpoint URL (e.g. https://<hydra-host>/oauth2/token). " +
					"Can also be set with the HOOK_SERVICE_TOKEN_URL environment variable. " +
					"Requires client_id and client_secret.",
				Optional: true,
			},
		},
	}
}

func (p *HookServiceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HookServiceProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("HOOK_SERVICE_HOST")
	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	token := os.Getenv("HOOK_SERVICE_TOKEN")
	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	clientID := os.Getenv("HOOK_SERVICE_CLIENT_ID")
	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}

	clientSecret := os.Getenv("HOOK_SERVICE_CLIENT_SECRET")
	if !config.ClientSecret.IsNull() {
		clientSecret = config.ClientSecret.ValueString()
	}

	tokenURL := os.Getenv("HOOK_SERVICE_TOKEN_URL")
	if !config.TokenURL.IsNull() {
		tokenURL = config.TokenURL.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError(
			"Missing Hook Service Host",
			"The provider requires a host to be set either in the provider configuration "+
				"block or via the HOOK_SERVICE_HOST environment variable.",
		)
		return
	}

	hasToken := token != ""
	hasClientCreds := clientID != "" || clientSecret != "" || tokenURL != ""

	if hasToken && hasClientCreds {
		resp.Diagnostics.AddError(
			"Conflicting Authentication",
			"Specify either token or client_id/client_secret/token_url, not both.",
		)
		return
	}

	if hasClientCreds {
		if clientID == "" || clientSecret == "" || tokenURL == "" {
			resp.Diagnostics.AddError(
				"Incomplete Client Credentials",
				"All three of client_id, client_secret, and token_url must be set for OAuth2 client credentials authentication.",
			)
			return
		}

		oauthConfig := clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     tokenURL,
		}
		httpClient := oauthConfig.Client(ctx)

		c := client.NewClientWithHTTPClient(host, httpClient)
		resp.DataSourceData = c
		resp.ResourceData = c
		return
	}

	c := client.NewClient(host, token)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *HookServiceProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewGroupResource,
		resources.NewGroupUsersResource,
		resources.NewGroupAppResource,
	}
}

func (p *HookServiceProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		resources.NewGroupsDataSource,
	}
}
