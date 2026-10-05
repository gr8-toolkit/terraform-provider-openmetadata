// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package datasources

import (
	"context"
	"fmt"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DriveServiceDataSource{}

type DriveServiceDataSource struct{ client *client.Client }

type DriveServiceDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
	ServiceType types.String `tfsdk:"service_type"`
	Domains     types.List   `tfsdk:"domains"`
	Owners      types.List   `tfsdk:"owners"`
}

func NewDriveServiceDataSource() datasource.DataSource { return &DriveServiceDataSource{} }

func (d *DriveServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_drive_service"
}

func (d *DriveServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Drive Service by name. Connection credentials are not returned.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, Description: "UUID of the service."},
			"name":                 schema.StringAttribute{Required: true, Description: "Name of the service to look up."},
			"display_name":         schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":          schema.StringAttribute{Computed: true, Description: "Description of the service."},
			"fully_qualified_name": schema.StringAttribute{Computed: true, Description: "Fully qualified name."},
			"service_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of drive service (e.g. GoogleDrive, OneDrive, S3, CustomDrive).",
			},
			"domains": schema.ListAttribute{
				Computed: true, Description: "Domains this service belongs to.",
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Computed: true, Description: "Owners of this service.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, Description: "UUID of the owner entity."},
						"type": schema.StringAttribute{Computed: true, Description: "Type of the owner (user or team)."},
					},
				},
			},
		},
	}
}

func (d *DriveServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *DriveServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DriveServiceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := d.client.GetByName(ctx, "services/driveServices", state.Name.ValueString(), []string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading drive service", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Drive service not found",
			fmt.Sprintf("no drive service with name %q found", state.Name.ValueString()))
		return
	}
	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing drive service response", err.Error())
		return
	}
	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.ServiceType = resources.StringVal(data, "serviceType")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
