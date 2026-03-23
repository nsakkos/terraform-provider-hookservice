package resources

import (
	"context"
	"fmt"

	"github.com/canonical/terraform-provider-hookservice/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &GroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &GroupsDataSource{}
)

// GroupsDataSource reads all groups from the Hook Service.
type GroupsDataSource struct {
	client *client.Client
}

// GroupsDataSourceModel describes the data source data model.
type GroupsDataSourceModel struct {
	Groups []GroupDataSourceModel `tfsdk:"groups"`
}

// GroupDataSourceModel describes a single group.
type GroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

func NewGroupsDataSource() datasource.DataSource {
	return &GroupsDataSource{}
}

func (d *GroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

func (d *GroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads all groups from the Hook Service.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				Description: "List of groups.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The unique identifier of the group.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the group.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A description of the group.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the group.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *GroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *GroupsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	groups, err := d.client.ListGroups()
	if err != nil {
		resp.Diagnostics.AddError("Error reading groups", err.Error())
		return
	}

	state := GroupsDataSourceModel{}
	for _, g := range groups {
		state.Groups = append(state.Groups, GroupDataSourceModel{
			ID:          types.StringValue(g.ID),
			Name:        types.StringValue(g.Name),
			Description: types.StringValue(g.Description),
			Type:        types.StringValue(g.Type),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
