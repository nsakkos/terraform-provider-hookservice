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
)

var _ provider.Provider = &HookServiceProvider{}

// HookServiceProvider defines the provider implementation.
type HookServiceProvider struct {
	version string
}

// HookServiceProviderModel describes the provider data model.
type HookServiceProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
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
					"Can also be set with the HOOK_SERVICE_TOKEN environment variable.",
				Optional:  true,
				Sensitive: true,
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

	if host == "" {
		resp.Diagnostics.AddError(
			"Missing Hook Service Host",
			"The provider requires a host to be set either in the provider configuration "+
				"block or via the HOOK_SERVICE_HOST environment variable.",
		)
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
