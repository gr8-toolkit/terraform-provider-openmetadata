---
inclusion: always
---

# Project Context

## Module & Addresses

- Go module: `github.com/gr8-toolkit/terraform-provider-openmetadata`
- Go version: `1.25.8` — all builds use `CGO_ENABLED=0`
- Provider registry address: `registry.terraform.io/gr8-toolkit/openmetadata`
- Provider type name: `"openmetadata"` → all resources are `openmetadata_*`

## Architecture

```text
main.go
  └─ entry point; version/commit injected by goreleaser ldflags;
     contains //go:generate directive for tfplugindocs

internal/provider/provider.go
  └─ schema (host + token), Configure() builds + pings client,
     Resources() registers all 9 resources, DataSources() returns empty

internal/client/client.go
  └─ thin net/http wrapper, no external HTTP library;
     30-second timeout, Bearer token auth, tflog structured logging

internal/resources/common.go
  └─ shared schema attribute constructors, JSON helpers, EntityRef struct,
     NormalizeJSONStringModifier plan modifier

internal/resources/<entity>.go   (one file per resource, 9 total)

internal/provider/provider_test.go
  └─ shared test infrastructure (factories, precheck, testRandName, testProviderBlock)

internal/provider/<name>_resource_test.go   (one file per resource)

examples/resources/openmetadata_<name>/resource.tf
  └─ HCL consumed by tfplugindocs to generate docs/resources/<name>.md

docs/
  └─ AUTO-GENERATED — never edit directly; run `make docs` to regenerate
```

## All Registered Resources

| Terraform resource | OM API collection | File |
|---|---|---|
| `openmetadata_classification` | `classifications` | `classification.go` |
| `openmetadata_database_service` | `services/databaseServices` | `database_service.go` |
| `openmetadata_domain` | `domains` | `domain.go` |
| `openmetadata_glossary` | `glossaries` | `glossary.go` |
| `openmetadata_glossary_term` | `glossaryTerms` | `glossary_term.go` |
| `openmetadata_policy` | `policies` | `policy.go` |
| `openmetadata_role` | `roles` | `role.go` |
| `openmetadata_tag` | `tags` | `tag.go` |
| `openmetadata_team` | `teams` | `team.go` |

**No data sources are implemented.** `DataSources()` returns an empty slice.

## Provider Authentication

Two optional schema attributes (`host`, `token`). Resolution order: explicit config value,
then environment variable (`OPENMETADATA_HOST` / `OPENMETADATA_TOKEN`).
`Configure()` errors with `AddError` if either is empty after resolution, then validates
connectivity via `client.Ping()` before injecting `*client.Client` into resources via
`resp.ResourceData`.

## Client API Surface (`internal/client/client.go`)

| Method | HTTP | Notes |
|---|---|---|
| `CreateOrUpdate(ctx, collection, body)` | `PUT /api/v1/{collection}` | Idempotent — used for both Create and Update |
| `GetByName(ctx, collection, fqn, fields)` | `GET .../name/{fqn}?fields=...` | Returns `nil, nil` on 404 |
| `GetByID(ctx, collection, id, fields)` | `GET .../{id}?fields=...` | Returns `nil, nil` on 404 |
| `Delete(ctx, collection, id, hardDelete)` | `DELETE .../{id}?hardDelete=true&recursive=true` | Always hard-deletes in practice |
| `Ping(ctx)` | `GET /api/v1/system/version` | Used by Configure() for connectivity check |

HTTP errors ≥ 400 are returned as `*client.APIError{Code, Message, ResponseHTTPCode}`.

## Direct Dependencies

| Package | Version | Purpose |
|---|---|---|
| `terraform-plugin-framework` | v1.19.0 | Core plugin SDK |
| `terraform-plugin-framework-validators` | v0.19.0 | `stringvalidator.OneOf`, etc. |
| `terraform-plugin-go` | v0.31.0 | Low-level protocol layer |
| `terraform-plugin-log` | v0.11.0 | `tflog` structured logging |
| `terraform-plugin-testing` | v1.16.0 | Acceptance test framework |
| `terraform-plugin-docs` | v0.25.0 | `tfplugindocs` — tools dep only (`tools/tools.go`) |
