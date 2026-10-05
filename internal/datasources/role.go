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

var _ datasource.DataSource = &RoleDataSource{}

type RoleDataSource struct {
	client *client.Client
}

type RoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
	Policies    types.List   `tfsdk:"policies"`
	Domains     types.List   `tfsdk:"domains"`
}

func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Role by name.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "UUID of the role."},
			"name":         schema.StringAttribute{Required: true, Description: "Name of the role to look up."},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description of the role."},
			"fully_qualified_name": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the role.",
			},
			"policies": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of policies attached to this role.",
				ElementType: types.StringType,
			},
			"domains": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of domains this role belongs to.",
				ElementType: types.StringType,
			},
		},
	}
}

func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state RoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.client.GetByName(ctx, "roles", state.Name.ValueString(),
		[]string{"domains", "policies"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Role not found",
			fmt.Sprintf("no role with name %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing role response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.Policies = resources.StringListVal(data, "policies")
	state.Domains = resources.StringListVal(data, "domains")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
