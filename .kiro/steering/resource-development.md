---
inclusion: fileMatch
fileMatchPattern: "internal/resources/*.go"
---

# Resource Development Guide

## File Structure

Every file in `internal/resources/` follows this exact layout — maintain it when adding or modifying resources:

```go
// 1. Compile-time interface checks (always both)
var _ resource.Resource = &XxxResource{}
var _ resource.ResourceWithImportState = &XxxResource{}

// 2. OM API path segment
const xxxCollection = "some/collection"

// 3. Resource struct — only holds the client
type XxxResource struct {
    client *client.Client
}

// 4. State model — every field has a tfsdk tag matching the TF attribute name
type XxxResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    DisplayName types.String `tfsdk:"display_name"`
    Description types.String `tfsdk:"description"`
    FQN         types.String `tfsdk:"fully_qualified_name"`
    // ... resource-specific fields ...
    Domains types.List `tfsdk:"domains"`
    Owners  types.List `tfsdk:"owners"`
}

// 5. Constructor
func NewXxxResource() resource.Resource { return &XxxResource{} }

// 6. Metadata — type name is always "openmetadata_xxx"
func (r *XxxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_xxx"
}

// 7. Schema — use helpers from common.go; never write attributes inline if a helper exists
// 8. Configure — type-assert ProviderData to *client.Client
// 9. Create  — get plan → buildBody → CreateOrUpdate → readIntoState → set state
// 10. Read   — get state → GetByName → nil check → readIntoState → set state
// 11. Update — identical to Create (OM uses idempotent PUT)
// 12. Delete — get state → client.Delete with state.ID
// 13. ImportState — GetByName with req.ID → readIntoState → set state
// 14. buildBody — constructs map[string]interface{} for the PUT request
// 15. readIntoState — parses json.RawMessage into the model struct
```

## Schema Helpers (`common.go`) — Always Use These

| Helper | Attribute | Behaviour |
|---|---|---|
| `IDAttribute()` | `"id"` | Computed, `UseStateForUnknown` |
| `NameAttribute()` | `"name"` | Required, `RequiresReplace` |
| `DisplayNameAttribute()` | `"display_name"` | Optional+computed, `UseStateForUnknown` |
| `DescriptionAttribute(required bool)` | `"description"` | Required when `true`; optional+computed when `false` |
| `FullyQualifiedNameAttribute()` | `"fully_qualified_name"` | Computed, `UseStateForUnknown` |
| `DomainsAttribute()` | `"domains"` | Optional list of strings |
| `OwnersAttribute()` | `"owners"` | Optional nested list `{id string, type string}` |
| `ExpertsAttribute()` | `"experts"` | Same shape as owners — used only in `domain.go` |

Use `NormalizeJSONString()` as a plan modifier on any attribute that holds a JSON string
to prevent drift from key-ordering differences.

## JSON Helpers (`common.go`) — Use in `readIntoState` and `buildBody`

| Helper | Use for |
|---|---|
| `Unmarshal(raw json.RawMessage)` | Parse API response → `map[string]interface{}` |
| `StringVal(data, key)` | Safe string extract → `types.String` (null if absent) |
| `BoolVal(data, key)` | Safe bool extract → `types.Bool` (null if absent) |
| `RawStringList(data, key)` | Extract `[]string` from a plain JSON string array (e.g., `synonyms`) |
| `EntityRefNames(data, key)` | Extract `fullyQualifiedName` (or `name`) from entity ref array |
| `StringListVal(data, key)` | Entity ref names → typed `types.List` (null if empty) |
| `StringSliceToList(vals)` | `[]string` → typed `types.List` |
| `OwnersListNull()` | Correctly-typed null list for `owners` — always use this in `readIntoState` |
| `ParseEntityRefs(data, key)` | Extract `[]EntityRef` from JSON when you need `{id, type}` pairs |
| `extractOwnerRefs(ctx, ownersList)` | Convert `types.List` owners → `[]EntityRef` for `buildBody` (defined in `classification.go`, available package-wide) |

## CRUD Pattern

Create and Update are **identical** — both call `CreateOrUpdate` (OM's idempotent PUT):

```go
func (r *XxxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan XxxResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
    body := r.buildBody(ctx, &plan)
    raw, err := r.client.CreateOrUpdate(ctx, xxxCollection, body)
    if err != nil {
        resp.Diagnostics.AddError("Error creating xxx", err.Error())
        return
    }
    r.readIntoState(raw, &plan)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *XxxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state XxxResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }
    raw, err := r.client.GetByName(ctx, xxxCollection, state.Name.ValueString(), []string{"owners", "domains"})
    if err != nil {
        resp.Diagnostics.AddError("Error reading xxx", err.Error())
        return
    }
    if raw == nil {
        resp.State.RemoveResource(ctx) // 404 → signal Terraform to recreate
        return
    }
    r.readIntoState(raw, &state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *XxxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state XxxResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }
    if err := r.client.Delete(ctx, xxxCollection, state.ID.ValueString(), true); err != nil {
        resp.Diagnostics.AddError("Error deleting xxx", err.Error())
    }
}
```

## `buildBody` Pattern

```go
func (r *XxxResource) buildBody(ctx context.Context, plan *XxxResourceModel) map[string]interface{} {
    body := map[string]interface{}{
        "name": plan.Name.ValueString(),
        // include other always-required fields here
    }
    // Optional scalar fields — guard with IsNull() && IsUnknown()
    if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
        body["displayName"] = plan.DisplayName.ValueString()
    }
    if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
        body["description"] = plan.Description.ValueString()
    }
    // List fields
    if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
        var vals []string
        plan.Domains.ElementsAs(ctx, &vals, false)
        body["domains"] = vals
    }
    // Owners — always use extractOwnerRefs
    if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
        body["owners"] = extractOwnerRefs(ctx, plan.Owners)
    }
    return body
}
```

## `readIntoState` Pattern

```go
func (r *XxxResource) readIntoState(raw []byte, state *XxxResourceModel) {
    data, err := Unmarshal(raw)
    if err != nil {
        return
    }
    state.ID          = StringVal(data, "id")
    state.Name        = StringVal(data, "name")
    state.DisplayName = StringVal(data, "displayName")
    state.Description = StringVal(data, "description")
    state.FQN         = StringVal(data, "fullyQualifiedName")
    state.Domains     = StringListVal(data, "domains")
    state.Owners      = OwnersListNull() // owners are never read back from the API
}
```

## Write-Only Fields

Some fields are sent to the API on write but **never read back** in `readIntoState`:

| Field | Resource | Reason |
|---|---|---|
| `parents` | `openmetadata_team` | OM always adds the implicit root `Organisation` parent, causing state drift |
| `rules` | `openmetadata_policy` | OM may reorder JSON keys, producing a non-equal string |
| `connection_json` | `openmetadata_database_service` | Sensitive — OM masks credential fields in responses |

For each write-only field:

1. Send it in `buildBody` as normal.
2. Skip it entirely in `readIntoState` with a comment explaining why.
3. Preserve the plan/state value by not overwriting it in `readIntoState`.
4. List it in `ImportStateVerifyIgnore` in the test file.

## ImportState Pattern

Import by entity name (or FQN for nested entities like `glossary_term` and `tag`):

```go
func (r *XxxResource) ImportState(
    ctx context.Context,
    req resource.ImportStateRequest,
    resp *resource.ImportStateResponse,
) {
    raw, err := r.client.GetByName(ctx, xxxCollection, req.ID, []string{"owners", "domains"})
    if err != nil {
        resp.Diagnostics.AddError("Error importing xxx", err.Error())
        return
    }
    if raw == nil {
        resp.Diagnostics.AddError("Xxx not found", fmt.Sprintf("No xxx with name %q", req.ID))
        return
    }
    var state XxxResourceModel
    r.readIntoState(raw, &state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

## FQN vs Short Name for `GetByName`

- **Top-level entities** (team, classification, glossary, policy, role, domain, database_service): look up by short `name`.
- **Nested entities** (tag, glossary_term): look up by FQN —
  e.g., `"MyClassification.MyTag"` or `"MyGlossary.MyTerm"`.
  Build the FQN from stored state fields if `state.FQN` is empty.

## Registering a New Resource

Add the constructor to `internal/provider/provider.go` → `Resources()`:

```go
func (p *OpenMetadataProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // ... existing ...
        resources.NewXxxResource, // add here
    }
}
```

## Checklist for a New Resource

1. Create `internal/resources/<entity>.go` following the file structure above.
2. Register in `internal/provider/provider.go` → `Resources()`.
3. Create `internal/provider/<entity>_resource_test.go` with full `TestAcc*` coverage (CI fails without it).
4. Create `examples/resources/openmetadata_<entity>/resource.tf` with a realistic HCL example.
5. Run `make docs` to regenerate `docs/resources/<entity>.md`.
6. Run `make fmt && make lint && make build && make test` — all must pass.
