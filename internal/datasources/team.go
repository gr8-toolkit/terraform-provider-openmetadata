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

var _ datasource.DataSource = &TeamDataSource{}

type TeamDataSource struct {
	client *client.Client
}

type TeamDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
	TeamType    types.String `tfsdk:"team_type"`
	Email       types.String `tfsdk:"email"`
	IsJoinable  types.Bool   `tfsdk:"is_joinable"`
	Parents     types.List   `tfsdk:"parents"`
	Policies    types.List   `tfsdk:"policies"`
	Domains     types.List   `tfsdk:"domains"`
	Owners      types.List   `tfsdk:"owners"`
}

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

func (d *TeamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Team by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "UUID of the team.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the team to look up.",
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "Human-readable display name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the team.",
				Computed:    true,
			},
			"fully_qualified_name": schema.StringAttribute{
				Description: "Fully qualified name of the team.",
				Computed:    true,
			},
			"team_type": schema.StringAttribute{
				Description: "Type of the team (Group, Department, Division, BusinessUnit, Organization).",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "Email address of the team.",
				Computed:    true,
			},
			"is_joinable": schema.BoolAttribute{
				Description: "Whether users can self-join this team.",
				Computed:    true,
			},
			"parents": schema.ListAttribute{
				Description: "Fully qualified names of parent teams.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"policies": schema.ListAttribute{
				Description: "Fully qualified names of policies attached to this team.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"domains": schema.ListAttribute{
				Description: "Fully qualified names of domains this team belongs to.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Description: "Owners of this team.",
				Computed:    true,
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

func (d *TeamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TeamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.client.GetByName(ctx, "teams", state.Name.ValueString(),
		[]string{"owners", "parents", "policies", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Team not found",
			fmt.Sprintf("no team with name %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing team response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.TeamType = resources.StringVal(data, "teamType")
	state.Email = resources.StringVal(data, "email")
	state.IsJoinable = resources.BoolVal(data, "isJoinable")
	state.Parents = resources.StringListVal(data, "parents")
	state.Policies = resources.StringListVal(data, "policies")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
