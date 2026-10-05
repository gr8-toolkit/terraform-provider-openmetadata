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

var _ datasource.DataSource = &GlossaryDataSource{}

type GlossaryDataSource struct {
	client *client.Client
}

type GlossaryDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
}

func NewGlossaryDataSource() datasource.DataSource {
	return &GlossaryDataSource{}
}

func (d *GlossaryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glossary"
}

func (d *GlossaryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Glossary by name.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "UUID of the glossary."},
			"name":         schema.StringAttribute{Required: true, Description: "Name of the glossary to look up."},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description of the glossary."},
			"fully_qualified_name": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the glossary.",
			},
			"mutually_exclusive": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether direct child terms are mutually exclusive.",
			},
			"domains": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of domains this glossary belongs to.",
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Owners of this glossary.",
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

func (d *GlossaryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GlossaryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state GlossaryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.client.GetByName(ctx, "glossaries", state.Name.ValueString(),
		[]string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading glossary", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Glossary not found",
			fmt.Sprintf("no glossary with name %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing glossary response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.MutuallyExclusive = resources.BoolVal(data, "mutuallyExclusive")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
