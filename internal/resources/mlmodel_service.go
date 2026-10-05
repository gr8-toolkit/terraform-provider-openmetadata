// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &MLModelServiceResource{}
var _ resource.ResourceWithImportState = &MLModelServiceResource{}

const mlmodelServiceCollection = "services/mlmodelServices"

type MLModelServiceResource struct{ client *client.Client }

type MLModelServiceResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Description    types.String `tfsdk:"description"`
	ServiceType    types.String `tfsdk:"service_type"`
	ConnectionJSON types.String `tfsdk:"connection_json"`
	Owners         types.List   `tfsdk:"owners"`
	Domains        types.List   `tfsdk:"domains"`
	FQN            types.String `tfsdk:"fully_qualified_name"`
}

func NewMLModelServiceResource() resource.Resource { return &MLModelServiceResource{} }

func (r *MLModelServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mlmodel_service"
}

func (r *MLModelServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata ML Model Service.",
		Attributes: map[string]schema.Attribute{
			"id": IDAttribute(), "name": NameAttribute(), "display_name": DisplayNameAttribute(),
			"description": DescriptionAttribute(false), "fully_qualified_name": FullyQualifiedNameAttribute(),
			"service_type": schema.StringAttribute{
				Description:   "Type of ML model service (e.g. Mlflow, Sagemaker, Vertex, CustomMlModel).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connection_json": schema.StringAttribute{
				Description: "ML model connection configuration as a JSON string. Sensitive and not read back from the API.",
				Optional:    true, Sensitive: true,
			},
			"owners": OwnersAttribute(), "domains": DomainsAttribute(),
		},
	}
}

func (r *MLModelServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *MLModelServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MLModelServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, mlmodelServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ML model service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MLModelServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MLModelServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetByName(ctx, mlmodelServiceCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading ML model service", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MLModelServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MLModelServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, mlmodelServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ML model service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MLModelServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MLModelServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Deleting ML model service", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, mlmodelServiceCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting ML model service", err.Error())
	}
}

func (r *MLModelServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, mlmodelServiceCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing ML model service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("ML model service not found", fmt.Sprintf("No ML model service with name %q", req.ID))
		return
	}
	var state MLModelServiceResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MLModelServiceResource) buildBody(ctx context.Context, plan *MLModelServiceResourceModel) (map[string]interface{}, error) {
	body := map[string]interface{}{"name": plan.Name.ValueString(), "serviceType": plan.ServiceType.ValueString()}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}
	if !plan.ConnectionJSON.IsNull() && !plan.ConnectionJSON.IsUnknown() {
		var conn interface{}
		if err := json.Unmarshal([]byte(plan.ConnectionJSON.ValueString()), &conn); err != nil {
			return nil, fmt.Errorf("invalid connection_json: %w", err)
		}
		body["connection"] = conn
	}
	if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
		body["owners"] = extractOwnerRefs(ctx, plan.Owners)
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		var vals []string
		plan.Domains.ElementsAs(ctx, &vals, false)
		body["domains"] = vals
	}
	return body, nil
}

func (r *MLModelServiceResource) readIntoState(raw []byte, state *MLModelServiceResourceModel) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")
	state.ServiceType = StringVal(data, "serviceType")
	state.FQN = StringVal(data, "fullyQualifiedName")
	state.Domains = StringListVal(data, "domains")
	state.Owners = OwnersListNull()
}
