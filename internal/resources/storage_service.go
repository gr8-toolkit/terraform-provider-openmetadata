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

var _ resource.Resource = &StorageServiceResource{}
var _ resource.ResourceWithImportState = &StorageServiceResource{}

const storageServiceCollection = "services/storageServices"

type StorageServiceResource struct{ client *client.Client }

type StorageServiceResourceModel struct {
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

func NewStorageServiceResource() resource.Resource { return &StorageServiceResource{} }

func (r *StorageServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_service"
}

func (r *StorageServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenMetadata Storage Service.",
		Attributes: map[string]schema.Attribute{
			"id": IDAttribute(), "name": NameAttribute(), "display_name": DisplayNameAttribute(),
			"description": DescriptionAttribute(false), "fully_qualified_name": FullyQualifiedNameAttribute(),
			"service_type": schema.StringAttribute{
				Description:   "Type of storage service (e.g. S3, GCS, ADLS, CustomStorage).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"connection_json": schema.StringAttribute{
				Description: "Storage connection configuration as a JSON string. Sensitive and not read back from the API.",
				Optional:    true, Sensitive: true,
			},
			"owners": OwnersAttribute(), "domains": DomainsAttribute(),
		},
	}
}

func (r *StorageServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StorageServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, storageServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating storage service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StorageServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := r.client.GetByName(ctx, storageServiceCollection, state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading storage service", err.Error())
		return
	}
	if raw == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StorageServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, err := r.buildBody(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building request body", err.Error())
		return
	}
	raw, err := r.client.CreateOrUpdate(ctx, storageServiceCollection, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating storage service", err.Error())
		return
	}
	r.readIntoState(raw, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *StorageServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Deleting storage service", map[string]interface{}{"name": state.Name.ValueString()})
	if err := r.client.Delete(ctx, storageServiceCollection, state.ID.ValueString(), true); err != nil {
		resp.Diagnostics.AddError("Error deleting storage service", err.Error())
	}
}

func (r *StorageServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	raw, err := r.client.GetByName(ctx, storageServiceCollection, req.ID, []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error importing storage service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Storage service not found", fmt.Sprintf("No storage service with name %q", req.ID))
		return
	}
	var state StorageServiceResourceModel
	r.readIntoState(raw, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StorageServiceResource) buildBody(ctx context.Context, plan *StorageServiceResourceModel) (map[string]interface{}, error) {
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

func (r *StorageServiceResource) readIntoState(raw []byte, state *StorageServiceResourceModel) {
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
