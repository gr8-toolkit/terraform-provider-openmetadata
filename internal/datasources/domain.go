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

var _ datasource.DataSource = &DomainDataSource{}

type DomainDataSource struct {
	client *client.Client
}

type DomainDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	FQN         types.String `tfsdk:"fully_qualified_name"`
	DomainType  types.String `tfsdk:"domain_type"`
	Parent      types.String `tfsdk:"parent"`
	Owners      types.List   `tfsdk:"owners"`
	Experts     types.List   `tfsdk:"experts"`
}

func NewDomainDataSource() datasource.DataSource {
	return &DomainDataSource{}
}

func (d *DomainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DomainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Domain by name.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "UUID of the domain."},
			"name":         schema.StringAttribute{Required: true, Description: "Name of the domain to look up."},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description of the domain."},
			"fully_qualified_name": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the domain.",
			},
			"domain_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the domain (Source-aligned, Consumer-aligned, Aggregate).",
			},
			"parent": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the parent domain, if any.",
			},
			"owners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Owners of this domain.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, Description: "UUID of the owner entity."},
						"type": schema.StringAttribute{Computed: true, Description: "Type of the owner (user or team)."},
					},
				},
			},
			"experts": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Subject-matter experts for this domain.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, Description: "UUID of the expert user."},
						"type": schema.StringAttribute{Computed: true, Description: "Type of the expert entity (user)."},
					},
				},
			},
		},
	}
}

func (d *DomainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DomainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.client.GetByName(ctx, "domains", state.Name.ValueString(),
		[]string{"owners", "experts", "parent"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Domain not found",
			fmt.Sprintf("no domain with name %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing domain response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.DomainType = resources.StringVal(data, "domainType")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))
	state.Experts = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "experts"))

	// Extract optional parent domain FQN.
	if p, ok := data["parent"].(map[string]interface{}); ok {
		if fqn, ok := p["fullyQualifiedName"].(string); ok {
			state.Parent = types.StringValue(fqn)
		} else if name, ok := p["name"].(string); ok {
			state.Parent = types.StringValue(name)
		} else {
			state.Parent = types.StringNull()
		}
	} else {
		state.Parent = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
