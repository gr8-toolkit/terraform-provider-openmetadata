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

var _ datasource.DataSource = &TagDataSource{}

type TagDataSource struct {
	client *client.Client
}

// TagDataSourceModel uses name as the full FQN (e.g. "MyClassification.MyTag").
type TagDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
	Classification    types.String `tfsdk:"classification"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
}

func NewTagDataSource() datasource.DataSource {
	return &TagDataSource{}
}

func (d *TagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Tag by its fully qualified name (Classification.TagName).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "UUID of the tag."},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Fully qualified name of the tag, e.g. \"PII.Sensitive\".",
			},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description of the tag."},
			"fully_qualified_name": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the tag.",
			},
			"classification": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the parent classification.",
			},
			"mutually_exclusive": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this tag is mutually exclusive within its classification.",
			},
			"domains": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of domains this tag belongs to.",
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Owners of this tag.",
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

func (d *TagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Tags are looked up by FQN (e.g. "PII.Sensitive").
	raw, err := d.client.GetByName(ctx, "tags", state.Name.ValueString(),
		[]string{"owners", "domains"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading tag", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Tag not found",
			fmt.Sprintf("no tag with FQN %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing tag response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.MutuallyExclusive = resources.BoolVal(data, "mutuallyExclusive")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))

	// Extract classification name from the classification entity ref.
	if cls, ok := data["classification"].(map[string]interface{}); ok {
		if fqn, ok := cls["fullyQualifiedName"].(string); ok {
			state.Classification = types.StringValue(fqn)
		} else if name, ok := cls["name"].(string); ok {
			state.Classification = types.StringValue(name)
		} else {
			state.Classification = types.StringNull()
		}
	} else {
		state.Classification = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
