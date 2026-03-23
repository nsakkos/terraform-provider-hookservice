package resources

import (
	"context"
	"fmt"

	"github.com/canonical/terraform-provider-hookservice/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &GroupAppResource{}
	_ resource.ResourceWithConfigure = &GroupAppResource{}
)

// GroupAppResource manages access for a group to an application.
type GroupAppResource struct {
	client *client.Client
}

// GroupAppResourceModel describes the resource data model.
type GroupAppResourceModel struct {
	GroupID  types.String `tfsdk:"group_id"`
	ClientID types.String `tfsdk:"client_id"`
}

func NewGroupAppResource() resource.Resource {
	return &GroupAppResource{}
}

func (r *GroupAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_app"
}

func (r *GroupAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Grants a group access to an application in the Hook Service.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Description: "The ID of the group.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OAuth client ID of the application to grant access to.",
				Required:    true,
			},
		},
	}
}

func (r *GroupAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *GroupAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AddGroupApp(plan.GroupID.ValueString(), plan.ClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error granting app access to group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apps, err := r.client.GetGroupApps(state.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group apps", err.Error())
		return
	}

	// Check if the managed client_id is still present.
	found := false
	for _, cid := range apps {
		if cid == state.ClientID.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *GroupAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove old app access.
	if err := r.client.RemoveGroupApp(state.GroupID.ValueString(), state.ClientID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error removing old app access", err.Error())
		return
	}

	// Add new app access.
	if err := r.client.AddGroupApp(plan.GroupID.ValueString(), plan.ClientID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error granting new app access", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RemoveGroupApp(state.GroupID.ValueString(), state.ClientID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error removing app access from group", err.Error())
		return
	}
}
