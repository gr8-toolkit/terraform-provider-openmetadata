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

var _ resource.Resource = &PipelineServiceResource{}
var _ resource.ResourceWithImportState = &PipelineServiceResource{}

const pipelineServiceCollection = "services/pipelineServices"

type PipelineServiceResource struct{ client *client.Client }

type PipelineServiceResourceModel struct {
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

func NewPipelineServiceResource() resource.Resource { return &PipelineServiceResource{} }

func (r *PipelineServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_service"
}

func (r *PipelineServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Pipeline Service.",
		Attributes: map[string]schema.Attribute{
			"id": IDAttribute(), "name": NameAttribute(), "display_name": DisplayNameAttribute(),
			"description": DescriptionAttribute(false), "fully_qualified_name": FullyQualifiedNameAttribute(),
			"service_type": schema.StringAttribute{
				Description:   "Type of pipeline service (e.g. Airflow, GlueAirflow, DatabricksPipeline, Nifi, SparkPipeline).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connection_json": schema.StringAttribute{
				Description: "Pipeline connection configuration as a JSON string. Sensitive and not read back from the API.",
				Optional:    true, Sensitive: true,
			},
			"owners": OwnersAttribute(), "domains": DomainsAttribute(),
		},
	}
}

func (r *PipelineServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PipelineServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PipelineServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, pipelineServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating pipeline service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PipelineServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetByName(ctx, pipelineServiceCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading pipeline service", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PipelineServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PipelineServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, pipelineServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating pipeline service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PipelineServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PipelineServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Deleting pipeline service", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, pipelineServiceCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting pipeline service", err.Error())
	}
}

func (r *PipelineServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, pipelineServiceCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing pipeline service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Pipeline service not found", fmt.Sprintf("No pipeline service with name %q", req.ID))
		return
	}
	var state PipelineServiceResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PipelineServiceResource) buildBody(ctx context.Context, plan *PipelineServiceResourceModel) (map[string]interface{}, error) {
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

func (r *PipelineServiceResource) readIntoState(raw []byte, state *PipelineServiceResourceModel) {
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
