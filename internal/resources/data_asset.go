// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &DataAssetResource{}
var _ resource.ResourceWithImportState = &DataAssetResource{}

// dataAssetCollections maps the `type` attribute to the OM API collection path.
// Every collection here supports the `owners`, `domains`, `tags` and `extension`
// fields and JSON Patch updates.
var dataAssetCollections = map[string]string{
	"api_collection":       "apiCollections",
	"api_endpoint":         "apiEndpoints",
	"chart":                "charts",
	"container":            "containers",
	"dashboard":            "dashboards",
	"dashboard_data_model": "dashboard/datamodels",
	"database":             "databases",
	"database_schema":      "databaseSchemas",
	"ml_model":             "mlmodels",
	"pipeline":             "pipelines",
	"search_index":         "searchIndexes",
	"stored_procedure":     "storedProcedures",
	"table":                "tables",
	"topic":                "topics",
}

// DataAssetFields are the optional entity fields the resource and data source
// ask OM to include in a GET.
var DataAssetFields = []string{"owners", "domains", "tags", "extension"}

// Tag label constants as used by OM's TagLabel.
const (
	tagSourceClassification = "Classification"
	tagSourceGlossary       = "Glossary"
	tagLabelManual          = "Manual"
)

// DataAssetTypes returns the supported values of the `type` attribute, sorted.
func DataAssetTypes() []string {
	names := make([]string, 0, len(dataAssetCollections))
	for t := range dataAssetCollections {
		names = append(names, t)
	}
	sort.Strings(names)
	return names
}

// DataAssetCollection returns the OM API collection path for a data asset type.
func DataAssetCollection(assetType string) (string, bool) {
	c, ok := dataAssetCollections[assetType]
	return c, ok
}

// DataAssetTypeValidator restricts `type` to the supported asset types.
func DataAssetTypeValidator() validator.String {
	return stringvalidator.OneOf(DataAssetTypes()...)
}

// DataAssetTypeDescription describes the `type` attribute, listing the values.
func DataAssetTypeDescription() string {
	return "Type of the data asset: one of `" + strings.Join(DataAssetTypes(), "`, `") + "`."
}

// DataAssetResource adopts a data asset that already exists in OpenMetadata
// (typically created by an ingestion pipeline) and manages its mutable metadata.
type DataAssetResource struct {
	client *client.Client
}

type DataAssetResourceModel struct {
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

func NewDataAssetResource() resource.Resource {
	return &DataAssetResource{}
}

func (r *DataAssetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_asset"
}

func (r *DataAssetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Every managed attribute is optional+computed: an attribute left out of the
	// configuration is not managed and keeps whatever value OpenMetadata holds.
	owners := OwnersAttribute()
	owners.Computed = true
	owners.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
	owners.Description = "Owners of the asset. Set to `[]` to remove all owners; omit to leave the current owners untouched."

	domains := DomainsAttribute()
	domains.Computed = true
	domains.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
	domains.Validators = []validator.List{listvalidator.UniqueValues()}
	domains.Description = "Fully qualified names of the domains the asset belongs to. " +
		"Set to `[]` to remove all domains; omit to leave them untouched."

	resp.Schema = schema.Schema{
		Description: "Manages the metadata of a data asset that already exists in OpenMetadata, such as a table " +
			"or topic created by an ingestion pipeline. The resource never creates the asset: it looks it up by " +
			"`type` and `fully_qualified_name` (failing if it does not exist), then applies the attributes you set " +
			"with a JSON Patch. Attributes you leave out are not managed and keep their current value. " +
			"Destroying the resource only removes it from Terraform state; the asset itself is left untouched.",
		Attributes: map[string]schema.Attribute{
			"id": IDAttribute(),
			"type": schema.StringAttribute{
				Description: DataAssetTypeDescription(),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{DataAssetTypeValidator()},
			},
			"fully_qualified_name": schema.StringAttribute{
				Description: "Fully qualified name of the existing asset, e.g. `my_service.my_db.public.orders` for a table.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the asset (last segment of the fully qualified name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"display_name": DisplayNameAttribute(),
			"description":  DescriptionAttribute(false),
			"owners":       owners,
			"domains":      domains,
			"tags": schema.ListAttribute{
				Description: "Fully qualified names of classification tags (e.g. `PII.Sensitive`) applied manually to the asset. " +
					"Tags applied by auto classification or derived from glossary terms are preserved and not listed here. " +
					"Set to `[]` to remove all manual tags; omit to leave them untouched.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{listvalidator.UniqueValues()},
			},
			"glossary_terms": schema.ListAttribute{
				Description: "Fully qualified names of glossary terms (e.g. `Finance.Revenue`) applied manually to the asset. " +
					"Set to `[]` to remove all manual glossary terms; omit to leave them untouched.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{listvalidator.UniqueValues()},
			},
			"extension_json": schema.StringAttribute{
				Description: "Custom property values of the asset as a JSON object (use `jsonencode`). " +
					"The object replaces all existing custom property values, so list every property you want to keep. " +
					"Values are compared in the format OpenMetadata returns them. Omit to leave custom properties untouched.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					NormalizeJSONString(),
				},
			},
		},
	}
}

func (r *DataAssetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DataAssetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DataAssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DataAssetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DataAssetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, ok := DataAssetCollection(state.Type.ValueString())
	if !ok {
		resp.Diagnostics.AddError("Unsupported data asset type", state.Type.ValueString())
		return
	}
	raw, err := r.client.GetByName(ctx, collection, state.FQN.ValueString(), DataAssetFields)
	if err != nil {
		resp.Diagnostics.AddError("Error reading data asset", err.Error())
		return
	}
	if raw == nil {
		// The asset is gone (e.g. removed by a later ingestion run); drop it from state.
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoState(ctx, raw, &state, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is identical to Create: both reconcile the live asset with the plan.
func (r *DataAssetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DataAssetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete only forgets the asset: its lifecycle belongs to the ingestion pipeline
// that created it, so the entity and its metadata are deliberately left in place.
func (r *DataAssetResource) Delete(ctx context.Context, req resource.DeleteRequest, _ *resource.DeleteResponse) {
	var state DataAssetResourceModel
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		return
	}
	tflog.Info(ctx, "Removing data asset from state; the asset itself is not deleted", map[string]interface{}{
		"type": state.Type.ValueString(),
		"fqn":  state.FQN.ValueString(),
	})
}

// ImportState imports an asset by "<type>:<fully qualified name>", e.g.
// "table:my_service.my_db.public.orders".
func (r *DataAssetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	assetType, fqn, ok := strings.Cut(req.ID, ":")
	collection, known := DataAssetCollection(assetType)
	if !ok || fqn == "" || !known {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected \"<type>:<fully_qualified_name>\" where type is one of %s, got %q.",
				strings.Join(DataAssetTypes(), ", "), req.ID))
		return
	}

	raw, err := r.client.GetByName(ctx, collection, fqn, DataAssetFields)
	if err != nil {
		resp.Diagnostics.AddError("Error importing data asset", err.Error())
		return
	}
	if raw == nil {
		resp.Diagnostics.AddError("Data asset not found", fmt.Sprintf("No %s with fully qualified name %q", assetType, fqn))
		return
	}

	state := DataAssetResourceModel{
		Type:          types.StringValue(assetType),
		FQN:           types.StringValue(fqn),
		Owners:        OwnersListNull(),
		Domains:       types.ListNull(types.StringType),
		Tags:          types.ListNull(types.StringType),
		GlossaryTerms: types.ListNull(types.StringType),
		ExtensionJSON: types.StringNull(),
	}
	r.readIntoState(ctx, raw, &state, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// apply looks up the existing asset, patches whatever differs from the plan and
// refreshes plan with the resulting server state.
func (r *DataAssetResource) apply(ctx context.Context, plan *DataAssetResourceModel, diags *diag.Diagnostics) {
	assetType, fqn := plan.Type.ValueString(), plan.FQN.ValueString()
	collection, ok := DataAssetCollection(assetType)
	if !ok {
		diags.AddError("Unsupported data asset type", assetType)
		return
	}

	raw, err := r.client.GetByName(ctx, collection, fqn, DataAssetFields)
	if err != nil {
		diags.AddError("Error reading data asset", err.Error())
		return
	}
	if raw == nil {
		diags.AddError("Data asset not found",
			fmt.Sprintf("No %s with fully qualified name %q exists. Data assets are created by ingestion: "+
				"run the ingestion pipeline first, then apply this configuration.", assetType, fqn))
		return
	}
	current, err := Unmarshal(raw)
	if err != nil {
		diags.AddError("Error parsing data asset response", err.Error())
		return
	}

	ops := r.buildPatch(ctx, current, plan, diags)
	if diags.HasError() {
		return
	}
	if len(ops) > 0 {
		id, _ := current["id"].(string)
		if _, err := r.client.Patch(ctx, collection, id, ops); err != nil {
			diags.AddError("Error updating data asset", err.Error())
			return
		}
		// Re-read rather than trust the PATCH response, which may omit relationship fields.
		raw, err = r.client.GetByName(ctx, collection, fqn, DataAssetFields)
		if err != nil {
			diags.AddError("Error reading data asset", err.Error())
			return
		}
		if raw == nil {
			diags.AddError("Data asset not found", fmt.Sprintf("%s %q disappeared while it was being updated", assetType, fqn))
			return
		}
	}

	r.readIntoState(ctx, raw, plan, true)
}

// buildPatch returns the JSON Patch operations that bring the current asset in
// line with every attribute the plan sets. Unset (null/unknown) attributes are
// not managed and produce no operation.
//
// All operations use "add": per RFC 6902 it replaces an existing member, and
// unlike "replace" it also works when the member is absent on the entity.
func (r *DataAssetResource) buildPatch(ctx context.Context, current map[string]interface{}, plan *DataAssetResourceModel, diags *diag.Diagnostics) []map[string]interface{} {
	var ops []map[string]interface{}
	add := func(path string, value interface{}) {
		ops = append(ops, map[string]interface{}{"op": "add", "path": path, "value": value})
	}
	set := func(v interface {
		IsNull() bool
		IsUnknown() bool
	}) bool {
		return !v.IsNull() && !v.IsUnknown()
	}
	str := func(key string) string {
		s, _ := current[key].(string)
		return s
	}

	if set(plan.DisplayName) && plan.DisplayName.ValueString() != str("displayName") {
		add("/displayName", plan.DisplayName.ValueString())
	}
	if set(plan.Description) && plan.Description.ValueString() != str("description") {
		add("/description", plan.Description.ValueString())
	}

	if set(plan.Owners) {
		want := extractOwnerRefs(ctx, plan.Owners)
		if !sameStrings(refKeys(want), refKeys(ParseEntityRefs(current, "owners"))) {
			if want == nil {
				want = []EntityRef{}
			}
			add("/owners", want)
		}
	}

	if set(plan.Domains) {
		want := listStrings(ctx, plan.Domains)
		if !sameStrings(want, EntityRefNames(current, "domains")) {
			refs := make([]EntityRef, 0, len(want))
			for _, fqn := range want {
				raw, err := r.client.GetByName(ctx, "domains", fqn, nil)
				if err != nil {
					diags.AddAttributeError(path.Root("domains"), "Error looking up domain", err.Error())
					return nil
				}
				if raw == nil {
					diags.AddAttributeError(path.Root("domains"), "Domain not found",
						fmt.Sprintf("No domain with fully qualified name %q", fqn))
					return nil
				}
				data, err := Unmarshal(raw)
				if err != nil {
					diags.AddError("Error parsing domain response", err.Error())
					return nil
				}
				id, _ := data["id"].(string)
				refs = append(refs, EntityRef{ID: id, Type: "domain"})
			}
			add("/domains", refs)
		}
	}

	desired := map[string][]string{}
	if set(plan.Tags) {
		desired[tagSourceClassification] = listStrings(ctx, plan.Tags)
	}
	if set(plan.GlossaryTerms) {
		desired[tagSourceGlossary] = listStrings(ctx, plan.GlossaryTerms)
	}
	if tags, changed := buildTagsPatch(current, desired); changed {
		add("/tags", tags)
	}

	if set(plan.ExtensionJSON) {
		var want map[string]interface{}
		if err := json.Unmarshal([]byte(plan.ExtensionJSON.ValueString()), &want); err != nil || want == nil {
			diags.AddAttributeError(path.Root("extension_json"), "Invalid extension_json",
				"extension_json must be a JSON object, e.g. jsonencode({ myProperty = \"value\" }).")
			return nil
		}
		have, _ := current["extension"].(map[string]interface{})
		if len(have) == 0 {
			have = map[string]interface{}{}
		}
		if !reflect.DeepEqual(want, have) {
			add("/extension", want)
		}
	}

	return ops
}

// buildTagsPatch computes the new `tags` array for the sources in desired
// (Classification and/or Glossary). Only manually applied labels of a managed
// source are replaced; automated, derived and propagated labels, and labels of
// unmanaged sources, are carried over unchanged. It reports whether anything
// differs from the current manual labels.
func buildTagsPatch(current map[string]interface{}, desired map[string][]string) ([]interface{}, bool) {
	if len(desired) == 0 {
		return nil, false
	}
	items, _ := current["tags"].([]interface{})
	kept := make([]interface{}, 0, len(items)+1)
	haveManual := map[string][]string{}
	for _, item := range items {
		label, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		source, _ := label["source"].(string)
		fqn, _ := label["tagFQN"].(string)
		labelType, _ := label["labelType"].(string)
		want, managed := desired[source]
		switch {
		case !managed:
			kept = append(kept, item)
		case labelType == tagLabelManual:
			haveManual[source] = append(haveManual[source], fqn) // replaced below
		case !containsString(want, fqn):
			kept = append(kept, item)
			// else: a non-manual label the user now asks for explicitly becomes manual.
		}
	}

	changed := false
	for source, want := range desired {
		if !sameStrings(want, haveManual[source]) {
			changed = true
		}
	}
	if !changed {
		return nil, false
	}

	sources := make([]string, 0, len(desired))
	for source := range desired {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		for _, fqn := range desired[source] {
			kept = append(kept, map[string]interface{}{
				"tagFQN":    fqn,
				"source":    source,
				"labelType": tagLabelManual,
				"state":     "Confirmed",
			})
		}
	}
	return kept, true
}

// readIntoState copies the server state into the model. Type and
// fully_qualified_name are inputs and are left alone. Lists keep the order of
// their prior value when the members are the same, so reordering on the server
// does not surface as a change. When keepPlannedExtension is true and the prior
// extension_json is set, it is kept as the user wrote it.
func (r *DataAssetResource) readIntoState(ctx context.Context, raw []byte, state *DataAssetResourceModel, keepPlannedExtension bool) {
	data, err := Unmarshal(raw)
	if err != nil {
		return
	}
	state.ID = StringVal(data, "id")
	state.Name = StringVal(data, "name")
	state.DisplayName = StringVal(data, "displayName")
	state.Description = StringVal(data, "description")

	priorOwners := extractOwnerRefs(ctx, state.Owners)
	state.Owners = OwnerRefsToList(orderedRefs(priorOwners, ParseEntityRefs(data, "owners")))
	state.Domains = StringSliceToListNonNull(orderedStrings(listStrings(ctx, state.Domains), EntityRefNames(data, "domains")))

	tags, terms := AssetTagFQNs(data, true)
	state.Tags = StringSliceToListNonNull(orderedStrings(listStrings(ctx, state.Tags), tags))
	state.GlossaryTerms = StringSliceToListNonNull(orderedStrings(listStrings(ctx, state.GlossaryTerms), terms))

	api := AssetExtensionJSON(data)
	prior := state.ExtensionJSON
	switch {
	case prior.IsNull() || prior.IsUnknown():
		state.ExtensionJSON = extensionValue(api)
	case keepPlannedExtension:
		// The PATCH succeeded; keep what the user wrote rather than OM's rendering of it.
	case jsonEqual(prior.ValueString(), api):
		// Same content (OM omits an empty extension): keep the prior spelling.
	default:
		state.ExtensionJSON = extensionValue(api)
	}
}

// AssetTagFQNs splits an asset's `tags` array into classification tag FQNs and
// glossary term FQNs. With manualOnly, labels that were not applied manually
// (automated, derived, propagated) are skipped.
func AssetTagFQNs(data map[string]interface{}, manualOnly bool) (tags, glossaryTerms []string) {
	items, _ := data["tags"].([]interface{})
	for _, item := range items {
		label, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if labelType, _ := label["labelType"].(string); manualOnly && labelType != tagLabelManual {
			continue
		}
		fqn, _ := label["tagFQN"].(string)
		switch source, _ := label["source"].(string); source {
		case tagSourceGlossary:
			glossaryTerms = append(glossaryTerms, fqn)
		case tagSourceClassification:
			tags = append(tags, fqn)
		}
	}
	return tags, glossaryTerms
}

// AssetExtensionJSON returns the asset's custom property values as canonical
// (key-sorted) JSON, or "" when it has none.
func AssetExtensionJSON(data map[string]interface{}) string {
	ext, _ := data["extension"].(map[string]interface{})
	if len(ext) == 0 {
		return ""
	}
	b, err := json.Marshal(ext)
	if err != nil {
		return ""
	}
	return string(b)
}

func extensionValue(extensionJSON string) types.String {
	if extensionJSON == "" {
		return types.StringNull()
	}
	return types.StringValue(extensionJSON)
}

// jsonEqual compares two JSON object strings; "" counts as an empty object.
func jsonEqual(a, b string) bool {
	parse := func(s string) (map[string]interface{}, bool) {
		m := map[string]interface{}{}
		if s == "" {
			return m, true
		}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return nil, false
		}
		if m == nil {
			m = map[string]interface{}{}
		}
		return m, true
	}
	ma, okA := parse(a)
	mb, okB := parse(b)
	return okA && okB && reflect.DeepEqual(ma, mb)
}

// --- list helpers ---

// StringSliceToListNonNull converts a []string into a typed list; unlike
// StringSliceToList it yields an empty list, not null, for no elements.
func StringSliceToListNonNull(vals []string) types.List {
	if vals == nil {
		vals = []string{}
	}
	list, _ := types.ListValueFrom(context.Background(), types.StringType, vals)
	return list
}

// OwnerRefsToList converts entity refs into a typed owners list; unlike
// OwnersListFromRefs it yields an empty list, not null, for no refs.
func OwnerRefsToList(refs []EntityRef) types.List {
	if len(refs) == 0 {
		empty, _ := types.ListValue(OwnersListNull().ElementType(context.Background()), nil)
		return empty
	}
	return OwnersListFromRefs(refs)
}

func listStrings(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var vals []string
	list.ElementsAs(ctx, &vals, false)
	return vals
}

func refKeys(refs []EntityRef) []string {
	keys := make([]string, len(refs))
	for i, ref := range refs {
		keys[i] = ref.Type + ":" + ref.ID
	}
	return keys
}

func containsString(vals []string, s string) bool {
	for _, v := range vals {
		if v == s {
			return true
		}
	}
	return false
}

// sameStrings reports whether a and b hold the same members, ignoring order.
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as, bs := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	return reflect.DeepEqual(as, bs)
}

// orderedStrings returns prior when it has the same members as actual, so a
// server-side reordering is not reported as a change; otherwise actual.
func orderedStrings(prior, actual []string) []string {
	if sameStrings(prior, actual) {
		return prior
	}
	return actual
}

func orderedRefs(prior, actual []EntityRef) []EntityRef {
	if sameStrings(refKeys(prior), refKeys(actual)) {
		return prior
	}
	return actual
}
