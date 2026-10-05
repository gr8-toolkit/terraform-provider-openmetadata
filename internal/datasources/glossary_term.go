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

var _ datasource.DataSource = &GlossaryTermDataSource{}

type GlossaryTermDataSource struct {
	client *client.Client
}

// GlossaryTermDataSourceModel uses name as the full FQN (e.g. "Glossary.Term" or
// "Glossary.Parent.Child").
type GlossaryTermDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	FQN               types.String `tfsdk:"fully_qualified_name"`
	Glossary          types.String `tfsdk:"glossary"`
	Parent            types.String `tfsdk:"parent"`
	Synonyms          types.List   `tfsdk:"synonyms"`
	RelatedTerms      types.List   `tfsdk:"related_terms"`
	MutuallyExclusive types.Bool   `tfsdk:"mutually_exclusive"`
	Domains           types.List   `tfsdk:"domains"`
	Owners            types.List   `tfsdk:"owners"`
}

func NewGlossaryTermDataSource() datasource.DataSource {
	return &GlossaryTermDataSource{}
}

func (d *GlossaryTermDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_glossary_term"
}

func (d *GlossaryTermDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an OpenMetadata Glossary Term by its fully qualified name " +
			"(e.g. \"BusinessGlossary.Revenue\" or \"BusinessGlossary.Finance.Revenue\").",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "UUID of the glossary term."},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Fully qualified name of the term, e.g. \"Glossary.TermName\" " +
					"or \"Glossary.Parent.TermName\".",
			},
			"display_name": schema.StringAttribute{Computed: true, Description: "Human-readable display name."},
			"description":  schema.StringAttribute{Computed: true, Description: "Description of the term."},
			"fully_qualified_name": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the term.",
			},
			"glossary": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the parent glossary.",
			},
			"parent": schema.StringAttribute{
				Computed:    true,
				Description: "Fully qualified name of the parent term, if any.",
			},
			"synonyms": schema.ListAttribute{
				Computed:    true,
				Description: "Alternate names for this term.",
				ElementType: types.StringType,
			},
			"related_terms": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of related glossary terms.",
				ElementType: types.StringType,
			},
			"mutually_exclusive": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether child terms are mutually exclusive.",
			},
			"domains": schema.ListAttribute{
				Computed:    true,
				Description: "Fully qualified names of domains this term belongs to.",
				ElementType: types.StringType,
			},
			"owners": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Owners of this term.",
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

func (d *GlossaryTermDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GlossaryTermDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state GlossaryTermDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := d.client.GetByName(ctx, "glossaryTerms", state.Name.ValueString(),
		[]string{"owners", "domains", "relatedTerms"})
	if err != nil {
		resp.Diagnostics.AddError("Error reading glossary term", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Glossary term not found",
			fmt.Sprintf("no glossary term with FQN %q found", state.Name.ValueString()))
		return
	}

	data, err := resources.Unmarshal(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing glossary term response", err.Error())
		return
	}

	state.ID = resources.StringVal(data, "id")
	state.DisplayName = resources.StringVal(data, "displayName")
	state.Description = resources.StringVal(data, "description")
	state.FQN = resources.StringVal(data, "fullyQualifiedName")
	state.MutuallyExclusive = resources.BoolVal(data, "mutuallyExclusive")
	state.Synonyms = resources.StringSliceToList(resources.RawStringList(data, "synonyms"))
	state.RelatedTerms = resources.StringListVal(data, "relatedTerms")
	state.Domains = resources.StringListVal(data, "domains")
	state.Owners = resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))

	// Extract the parent glossary FQN from the nested glossary entity ref.
	if g, ok := data["glossary"].(map[string]interface{}); ok {
		if fqn, ok := g["fullyQualifiedName"].(string); ok {
			state.Glossary = types.StringValue(fqn)
		} else if name, ok := g["name"].(string); ok {
			state.Glossary = types.StringValue(name)
		} else {
			state.Glossary = types.StringNull()
		}
	} else {
		state.Glossary = types.StringNull()
	}

	// Extract optional parent term FQN.
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
