# Resource development

Applies to `internal/resources/*.go`. Data sources in `internal/datasources/` reuse the helpers in
`common.go` (`resources.StringVal`, `resources.Unmarshal`, ...) and follow the same read patterns.

## File layout

Every resource file follows this order — keep it when adding or modifying resources:

```go
// 1. Compile-time interface checks (always both)
var _ resource.Resource = &XxxResource{}
var _ resource.ResourceWithImportState = &XxxResource{}

// 2. OM API path segment
const xxxCollection = "some/collection"

// 3. Resource struct — only holds the client
type XxxResource struct{ client *client.Client }

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

// 5. Constructor        func NewXxxResource() resource.Resource { return &XxxResource{} }
// 6. Metadata           resp.TypeName = req.ProviderTypeName + "_xxx"
// 7. Schema             use helpers from common.go; never write an attribute inline if a helper exists
// 8. Configure          type-assert ProviderData to *client.Client
// 9. Create             plan → buildBody → CreateOrUpdate → readIntoState → set state
// 10. Read              state → GetByName → nil check → readIntoState → set state
// 11. Update            identical to Create (OM uses an idempotent PUT)
// 12. Delete            state → client.Delete with state.ID
// 13. ImportState       GetByName with req.ID → readIntoState → set state
// 14. buildBody         builds the map[string]interface{} for the PUT
// 15. readIntoState     parses json.RawMessage into the model
```

## Schema helpers (`common.go`) — always use these

| Helper | Attribute | Behaviour |
|---|---|---|
| `IDAttribute()` | `id` | Computed, `UseStateForUnknown` |
| `NameAttribute()` | `name` | Required, `RequiresReplace` |
| `DisplayNameAttribute()` | `display_name` | Optional+computed, `UseStateForUnknown` |
| `DescriptionAttribute(required bool)` | `description` | Required when `true`; optional+computed when `false` |
| `FullyQualifiedNameAttribute()` | `fully_qualified_name` | Computed, `UseStateForUnknown` |
| `DomainsAttribute()` | `domains` | Optional list of strings |
| `OwnersAttribute()` | `owners` | Optional nested list `{id string, type string}` |
| `ExpertsAttribute()` | `experts` | Same shape as owners — only used in `domain.go` |

Use the `NormalizeJSONString()` plan modifier on any attribute holding a JSON string so key
ordering does not cause drift.

## JSON helpers (`common.go`) — use in `readIntoState` and `buildBody`

| Helper | Use for |
|---|---|
| `Unmarshal(raw)` | Parse API response → `map[string]interface{}` |
| `StringVal` / `BoolVal` / `Int64Val(data, key)` | Safe extract → typed value, null if absent |
| `RawStringList(data, key)` | `[]string` from a plain JSON string array (e.g. `synonyms`) |
| `EntityRefNames(data, key)` | `fullyQualifiedName` (or `name`) from an entity-ref array |
| `StringListVal(data, key)` | Entity-ref names → typed `types.List` (null if empty) |
| `StringSliceToList(vals)` | `[]string` → typed `types.List` |
| `OwnersListNull()` | Correctly typed null `owners` list — use it in `readIntoState` |
| `ParseEntityRefs(data, key)` | `[]EntityRef` when you need `{id, type}` pairs |
| `OwnersListFromRefs(refs)` | `[]EntityRef` → typed owners list (null when empty) |
| `extractOwnerRefs(ctx, list)` | Owners `types.List` → `[]EntityRef` for `buildBody` (defined in `classification.go`, package-wide) |

## CRUD pattern

Create and Update are identical — both call `CreateOrUpdate`.

```go
func (r *XxxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan XxxResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
    raw, err := r.client.CreateOrUpdate(ctx, xxxCollection, r.buildBody(ctx, &plan))
    if err != nil {
        resp.Diagnostics.AddError("Error creating xxx", err.Error())
        return
    }
    r.readIntoState(raw, &plan)
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

In `Read`, a `nil` result from `GetByName` means 404: call `resp.State.RemoveResource(ctx)` so
Terraform recreates it. In `Delete`, call `r.client.Delete(ctx, xxxCollection, state.ID.ValueString(), true)`.

## `buildBody`

```go
body := map[string]interface{}{"name": plan.Name.ValueString() /* + always-required fields */}
// Optional scalars: guard with both IsNull() and IsUnknown()
if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
    body["description"] = plan.Description.ValueString()
}
if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
    var vals []string
    plan.Domains.ElementsAs(ctx, &vals, false)
    body["domains"] = vals
}
if !plan.Owners.IsNull() && !plan.Owners.IsUnknown() {
    body["owners"] = extractOwnerRefs(ctx, plan.Owners) // always use extractOwnerRefs
}
```

## `readIntoState`

```go
data, err := Unmarshal(raw)
if err != nil {
    return
}
state.ID = StringVal(data, "id")
state.Name = StringVal(data, "name")
state.DisplayName = StringVal(data, "displayName")
state.Description = StringVal(data, "description")
state.FQN = StringVal(data, "fullyQualifiedName")
state.Domains = StringListVal(data, "domains")
state.Owners = OwnersListNull() // owners are never read back from the API
```

## Write-only fields

Some fields are sent on write but never read back:

| Field | Resource | Reason |
|---|---|---|
| `parents` | `openmetadata_team` | OM always adds the implicit root `Organisation` parent → state drift |
| `rules` | `openmetadata_policy` | OM may reorder JSON keys → non-equal string |
| `connection_json` | service resources | Sensitive — OM masks credential fields in responses |
| `source_config_json` | `openmetadata_ingestion_pipeline` | OM stores it with server-side defaults filled in |

For each write-only field: send it in `buildBody`; skip it in `readIntoState` with a comment saying
why; do not overwrite the plan/state value; list it in `ImportStateVerifyIgnore` in the test.

## ImportState

Import by entity name — or by FQN for nested entities (`glossary_term`, `tag`) and entities that
live under a parent (`ingestion_pipeline` is `<service>.<pipeline>`):

```go
raw, err := r.client.GetByName(ctx, xxxCollection, req.ID, []string{"owners", "domains"})
// err → AddError("Error importing xxx", ...); raw == nil → AddError("Xxx not found", ...)
var state XxxResourceModel
r.readIntoState(raw, &state)
resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
```

`GetByName` takes the short `name` for top-level entities and the FQN for nested ones
(`"MyClassification.MyTag"`, `"MyGlossary.MyTerm"`). Build the FQN from state if `state.FQN` is empty.

## Registering

Add the constructor to `Resources()` in `internal/provider/provider.go`
(`resources.NewXxxResource`), and data sources to `DataSources()` (`omds.NewXxxDataSource`).
