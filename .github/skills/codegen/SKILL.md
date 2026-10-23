---
name: codegen
description: >-
  Add a new OpenMetadata entity to this Terraform provider: the resource, its data source, tests, examples, README
  entries and regenerated docs, following the patterns already in the codebase. Use when asked to add, generate or
  scaffold a new `openmetadata_*` resource or data source (e.g. a new service type or governance entity).
argument-hint: "<entity name> [OpenMetadata API collection]"
---

# Add a new resource

Generate a resource, a data source, tests, examples and docs for one OpenMetadata entity. Copy the closest existing
implementation and change only what differs. Do not invent a new style: the existing files are the specification.

Read `CLAUDE.md`, `internal/resources/CLAUDE.md` and `internal/provider/CLAUDE.md` first — they define the file
layout, helper functions, write-only field rules and test structure that this skill applies.

## 1. Gather inputs

Determine, asking the user only for what you cannot derive:

- **Entity name** in snake_case → Terraform type `openmetadata_<entity>`, Go names `<Entity>Resource`,
  `<Entity>DataSource`, file `<entity>.go`.
- **OM API collection** — the path under `/api/v1/` (e.g. `services/messagingServices`, `teams`, `glossaryTerms`).
- **Request schema** — the entity's create request in the OpenMetadata JSON schema:
  `openmetadata-spec/src/main/resources/json/schema/api/<area>/create<Entity>.json`, with the stored entity in
  `.../schema/entity/<area>/<entity>.json`. If a local `OpenMetadata/` checkout (gitignored symlink) exists in the
  repo root, read it there; otherwise fetch the files from
  `https://raw.githubusercontent.com/open-metadata/OpenMetadata/main/openmetadata-spec/src/main/resources/json/schema/...`.
  From it take the required fields, enums, and which fields the server defaults or masks.
- Whether the entity supports `owners`, `domains`, `displayName`, `description` (most do).

If the entity already exists in `internal/resources/`, stop and tell the user instead of overwriting it.

## 2. Pick the template

| The new entity is... | Copy from |
|---|---|
| A service type (`services/<x>Services`: has `serviceType` + `connection`) | `messaging_service.go` (resource and data source) |
| A top-level governance entity (name, description, owners, domains) | `classification.go`, `glossary.go` or `role.go` |
| An entity with a parent in its FQN (imported by FQN) | `tag.go`, `glossary_term.go` |
| An entity with a nested object or a reference to another entity | `ingestion_pipeline.go` (nested `service` object, `airflow_config`, name → UUID lookup) |
| An entity with a JSON-string attribute | `policy.go` (`rules`) or `database_service.go` (`connection_json`) |

For a **service type**, only these differ from `messaging_service.go`: the collection constant, the type name, the
`service_type` description, the user-facing strings ("messaging service"), the `connection_json` description, and the
connection JSON used in tests. Everything else is identical — keep it identical.

Read the chosen template files (resource, data source, both tests, both examples) before writing anything.

## 3. Write the code

### Resource — `internal/resources/<entity>.go`

Follow the 15-step layout in `internal/resources/CLAUDE.md`. Specifics:

- Header: `// Copyright (c) OpenMetadata Contributors` and `// SPDX-License-Identifier: Apache-2.0`.
- Collection as an unexported const: `const <entity>Collection = "..."`.
- Use the schema helpers (`IDAttribute`, `NameAttribute`, `DisplayNameAttribute`, `DescriptionAttribute`,
  `FullyQualifiedNameAttribute`, `OwnersAttribute`, `DomainsAttribute`). Fields that cannot change after creation get
  `RequiresReplace` (`name` already has it).
- `buildBody` sends only fields the user set (guard with `IsNull()` and `IsUnknown()`); validate JSON-string
  attributes with `json.Unmarshal` and return an error.
- `readIntoState` reads every server-owned field; `Owners` is always `OwnersListNull()`.
- Request fields the API masks, reorders or defaults (credentials, JSON blobs, parent links) are **write-only**:
  send them, skip them in `readIntoState`, mark them `Sensitive` where relevant, and list them in the test's
  `ImportStateVerifyIgnore`. Say in the schema description that they are not read back.
- `Read`/`ImportState` use `GetByName` with `[]string{"owners", "domains"}` (plus any other `fields` the entity needs);
  use the FQN for nested entities.
- Enums in the JSON schema become `stringvalidator.OneOf(...)` validators, with the values listed in the description.

### Data source — `internal/datasources/<entity>.go`

Same package conventions as `internal/datasources/messaging_service.go`:

- Only `name` is Required; everything else `Computed`. Never expose write-only or sensitive attributes.
- `Read` uses `GetByName`, and on `nil` adds the error `"<Entity> not found"` (the test asserts that text).
- Unlike the resource, the data source **does** populate owners:
  `resources.OwnersListFromRefs(resources.ParseEntityRefs(data, "owners"))`.
- Use the `resources.*` helpers; do not duplicate them.

### Register

In `internal/provider/provider.go`, append `resources.New<Entity>Resource` to `Resources()` and
`omds.New<Entity>DataSource` to `DataSources()`.

## 4. Write the tests (`internal/provider/`)

CI fails if `<entity>_resource_test.go` is missing, and acceptance tests are the only coverage for real API behavior.

- `<entity>_resource_test.go`: `TestAcc<Entity>Resource` with the three required steps (create+read, update, import),
  `testRandName` with a prefix of 6 characters or fewer, `testProviderBlock()` and `%q` in the config helper,
  `ImportStateVerifyIgnore` for write-only fields. Add `TestAcc<Entity>ResourceWithOwners`-style functions for
  optional fields worth covering.
- `<entity>_data_source_test.go`: `TestAcc<Entity>DataSource` (compare against the resource with
  `TestCheckResourceAttrPair`; `TestCheckNoResourceAttr` for write-only fields) and
  `TestAcc<Entity>DataSourceNotFound` (`ExpectError` matching `"<Entity> not found"`).
- If a fake-server test is useful (the entity has server-side defaults or tricky state handling), extend `fakeOM`
  in `fake_om_test.go` and add `<entity>_fake_test.go`, as done for ingestion pipelines. Remove any helper you
  do not end up using — the module-wide lint rejects unused code.

## 5. Examples, README and docs

- `examples/resources/openmetadata_<entity>/resource.tf` and
  `examples/data-sources/openmetadata_<entity>/data-source.tf` — realistic, minimal-but-useful HCL. Run
  `terraform fmt` on them if `terraform` is installed.
- `README.md`: add a row to the matching resource table (Governance or Services) and the name to the data source
  list; add an import line to the import section if the README lists imports.
- `make docs` regenerates `docs/`. Never edit `docs/` by hand. If `docs/resources/<other>.md` changes that you did
  not touch, mention it rather than reverting it.

## 6. Verify

Run, in order, and fix failures before reporting:

```bash
make fmt && make lint && make build && make test
```

Acceptance tests need a live OpenMetadata and take ~20 minutes; do not claim they pass unless you ran them
(`make testacc-external` with `OPENMETADATA_HOST` / `OPENMETADATA_TOKEN`, or
`TF_ACC=1 go test -v -run 'TestAcc<Entity>' ./internal/provider/...`). Otherwise say they were not run.

## 7. Report

List the files created or changed, the template you copied and what you changed from it, any write-only fields and
the reason, and exactly which checks you ran and their results. Do not commit unless asked; when you do, use a
Conventional Commit such as `feat: add openmetadata_<entity> resource and data source`.
