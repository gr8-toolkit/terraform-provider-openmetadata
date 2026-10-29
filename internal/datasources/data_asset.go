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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &DataAssetDataSource{}

type DataAssetDataSource struct {
	client *client.Client
}

type DataAssetDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Type          types.String `tfsdk:"type"`
	FQN           types.String `tfsdk:"fully_qualified_name"`
	Name          types.String `tfsdk:"name"`
	DisplayName   types.String `tfsdk:"display_name"`
	Description   types.String `tfsdk:"description"`
	Owners        types.List   `tfsdk:"owners"`
	Domains       types.List   `tfsdk:"domains"`
	Tags          types.List   `tfsdk:"tags"`
	GlossaryTerms types.List   `tfsdk:"glossary_terms"`
	ExtensionJSON types.String `tfsdk:"extension_json"`
}

func NewDataAssetDataSource() datasource.DataSource {
	return &DataAssetDataSource{}
}

func (d *DataAssetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_asset"
}

func (d *DataAssetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the current metadata of an existing OpenMetadata data asset (a table, topic, dashboard, " +
			"container, ...) by type and fully qualified name, e.g. an asset created by an ingestion pipeline.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "UUID of the asset."},
			"type": schema.StringAttribute{
				Required:    true,
				Description: resources.DataAssetTypeDescription(),
				Validators:  []validator.String{resources.DataAssetTypeValidator()},
			},
			"fully_qualified_name": schema.StringAttribute{
				Required:    true,
				Description: "Fully qualified name of the asset, e.g. `my_service.my_db.public.orders` for a table.",
			},
			"name":         schema.StringAttribute{Computed: true, Description: "Name of the asset."},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Markdown description of the asset."},
			"owners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Owners of the asset.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, Description: "UUID of the owner entity."},
						"type": schema.StringAttribute{Computed: true, Description: "Type of the owner (user or team)."},
					},
				},
			},
			"domains": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of the domains the asset belongs to.",
				ElementType: types.StringType,
			},
			"tags": schema.ListAttribute{
				Computed: true,
				Description: "Fully qualified names of all classification tags on the asset, whether applied manually, " +
					"by auto classification or derived.",
				ElementType: types.StringType,
			},
			"glossary_terms": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of all glossary terms on the asset.",
				ElementType: types.StringType,
			},
			"extension_json": schema.StringAttribute{
				Computed:    true,
				Description: "Custom property values of the asset as a JSON object; null when none are set.",
			},
		},
	}
}

func (d *DataAssetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DataAssetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataAssetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, ok := resources.DataAssetCollection(state.Type.ValueString())
	if !ok {
		resp.Diagnostics.AddError("Unsupported data asset type", state.Type.ValueString())
		return
	}
	raw, err := d.client.GetByName(ctx, collection, state.FQN.ValueString(), resources.DataAssetFields)
	if err != nil {
		resp.Diagnostics.AddError("Error reading data asset", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Data asset not found",
			fmt.Sprintf("no %s with fully qualified name %q found", state.Type.ValueString(), state.FQN.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing data asset response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.Name = resources.StringVal(data, "name")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.Owners = resources.OwnerRefsToList(resources.ParseEntityRefs(data, "owners"))
	state.Domains = resources.StringSliceToListNonNull(resources.EntityRefNames(data, "domains"))
	tags, terms := resources.AssetTagFQNs(data, false)
	state.Tags = resources.StringSliceToListNonNull(tags)
	state.GlossaryTerms = resources.StringSliceToListNonNull(terms)
	state.ExtensionJSON = types.StringNull()
	if ext := resources.AssetExtensionJSON(data); ext != "" {
		state.ExtensionJSON = types.StringValue(ext)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
