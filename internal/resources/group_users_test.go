package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestGroupUsersSchema_GroupIDRequiresReplace guards against the regression that
// silently emptied groups: without RequiresReplace on group_id, a group rename
// planned group_users as an in-place update, which diffed the email set against
// itself, made no API calls, and left the new group with no members.
func TestGroupUsersSchema_GroupIDRequiresReplace(t *testing.T) {
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	NewGroupUsersResource().(*GroupUsersResource).Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", schemaResp.Diagnostics)
	}

	attr, ok := schemaResp.Schema.Attributes["group_id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("group_id is not a StringAttribute: %T", schemaResp.Schema.Attributes["group_id"])
	}
	if len(attr.PlanModifiers) == 0 {
		t.Fatal("group_id has no plan modifiers; a changed group_id would be planned as an in-place update")
	}

	// Simulate a group rename: group_id is known in state and unknown in the
	// plan because the referenced group resource is being replaced.
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"group_id": tftypes.String}}
	newRaw := func(v interface{}) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"group_id": tftypes.NewValue(tftypes.String, v),
		})
	}

	req := planmodifier.StringRequest{
		Path:        path.Root("group_id"),
		StateValue:  types.StringValue("old-group-id"),
		PlanValue:   types.StringUnknown(),
		ConfigValue: types.StringUnknown(),
		State:       tfsdk.State{Raw: newRaw("old-group-id")},
		Plan:        tfsdk.Plan{Raw: newRaw(tftypes.UnknownValue)},
		Config:      tfsdk.Config{Raw: newRaw(tftypes.UnknownValue)},
	}

	requiresReplace := false
	for _, pm := range attr.PlanModifiers {
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		pm.PlanModifyString(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected plan modifier diagnostics: %v", resp.Diagnostics)
		}
		if resp.RequiresReplace {
			requiresReplace = true
		}
	}

	if !requiresReplace {
		t.Fatal("group_id must force replacement when it changes; otherwise a group rename silently drops all members")
	}
}
