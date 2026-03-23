package resources

import (
	"context"
	"fmt"
	"sort"

	"github.com/canonical/terraform-provider-hookservice/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &GroupUsersResource{}
	_ resource.ResourceWithConfigure = &GroupUsersResource{}
)

// GroupUsersResource manages the set of users belonging to a Hook Service group.
type GroupUsersResource struct {
	client *client.Client
}

// GroupUsersResourceModel describes the resource data model.
type GroupUsersResourceModel struct {
	GroupID types.String `tfsdk:"group_id"`
	Emails  types.Set    `tfsdk:"emails"`
}

func NewGroupUsersResource() resource.Resource {
	return &GroupUsersResource{}
}

func (r *GroupUsersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_users"
}

func (r *GroupUsersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the users belonging to a group in the Hook Service.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Description: "The ID of the group.",
				Required:    true,
			},
			"emails": schema.SetAttribute{
				Description: "The set of user email addresses to add to the group.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *GroupUsersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupUsersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupUsersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emails, diags := extractEmails(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(emails) > 0 {
		if err := r.client.AddGroupUsers(plan.GroupID.ValueString(), emails); err != nil {
			resp.Diagnostics.AddError("Error adding users to group", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupUsersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupUsersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := r.client.GetGroupUsers(state.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group users", err.Error())
		return
	}

	managedEmails, diags := extractEmails(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remoteSet := make(map[string]bool, len(users))
	for _, u := range users {
		remoteSet[u] = true
	}

	// Keep only managed emails that still exist on the remote.
	var stillPresent []string
	for _, email := range managedEmails {
		if remoteSet[email] {
			stillPresent = append(stillPresent, email)
		}
	}

	sort.Strings(stillPresent)

	newSet, setDiags := types.SetValueFrom(ctx, types.StringType, stillPresent)
	resp.Diagnostics.Append(setDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Emails = newSet
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *GroupUsersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupUsersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupUsersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planEmails, diags := extractEmails(ctx, plan)
	resp.Diagnostics.Append(diags...)
	stateEmails, diags := extractEmails(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planSet := make(map[string]bool, len(planEmails))
	for _, e := range planEmails {
		planSet[e] = true
	}
	stateSet := make(map[string]bool, len(stateEmails))
	for _, e := range stateEmails {
		stateSet[e] = true
	}

	// Add new users.
	var toAdd []string
	for _, e := range planEmails {
		if !stateSet[e] {
			toAdd = append(toAdd, e)
		}
	}
	if len(toAdd) > 0 {
		if err := r.client.AddGroupUsers(plan.GroupID.ValueString(), toAdd); err != nil {
			resp.Diagnostics.AddError("Error adding users to group", err.Error())
			return
		}
	}

	// Remove old users.
	for _, e := range stateEmails {
		if !planSet[e] {
			if err := r.client.RemoveGroupUser(plan.GroupID.ValueString(), e); err != nil {
				resp.Diagnostics.AddError("Error removing user from group", err.Error())
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *GroupUsersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupUsersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	emails, diags := extractEmails(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, email := range emails {
		if err := r.client.RemoveGroupUser(state.GroupID.ValueString(), email); err != nil {
			resp.Diagnostics.AddError("Error removing user from group", err.Error())
			return
		}
	}
}

func extractEmails(ctx context.Context, model GroupUsersResourceModel) ([]string, diag.Diagnostics) {
	var emails []string
	diags := model.Emails.ElementsAs(ctx, &emails, false)
	return emails, diags
}
