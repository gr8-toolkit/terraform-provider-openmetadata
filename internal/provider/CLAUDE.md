# Test patterns

Applies to `internal/provider/*_test.go`. All tests are `package provider_test`. Acceptance tests are
gated by `TF_ACC=1`; `make test` runs without it (no live OpenMetadata needed).

CI requires `internal/provider/<base>_resource_test.go` for every `internal/resources/<base>.go`
(except `common.go`). Data sources use `<base>_data_source_test.go`.

## Shared infrastructure (`provider_test.go`)

- `testAccProtoV6ProviderFactories` — in-process provider server; include in every `TestCase`.
- `testAccPreCheck(t)` — aborts early if `OPENMETADATA_HOST` / `OPENMETADATA_TOKEN` are missing.
- `testRandName(prefix)` — collision-safe name `tfacc_{prefix}_{5-digit-random}`; keep the prefix ≤ 6 chars
  to stay within OM name length limits.
- `testProviderBlock()` — `provider "openmetadata" {}` reading host/token from env.

## Required three-step TestCase

Every `TestAcc*` resource test covers Create+Read, Update, Import, in that order:

```go
func TestAccXxxResource(t *testing.T) {
    name := testRandName("xxx")
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Step 1: Create + Read — check every non-computed attribute, plus id and fully_qualified_name
            {
                Config: testAccXxxConfig(name, "Initial description"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("openmetadata_xxx.test", "name", name),
                    resource.TestCheckResourceAttrSet("openmetadata_xxx.test", "id"),
                ),
            },
            // Step 2: Update — change at least one mutable attribute and verify it took effect
            {
                Config: testAccXxxConfig(name, "Updated description"),
                Check:  resource.TestCheckResourceAttr("openmetadata_xxx.test", "description", "Updated description"),
            },
            // Step 3: Import
            {
                ResourceName:            "openmetadata_xxx.test",
                ImportState:             true,
                ImportStateVerify:       true,
                ImportStateVerifyIgnore: []string{}, // write-only fields, see below
                ImportStateIdFunc: func(s *terraform.State) (string, error) {
                    rs := s.RootModule().Resources["openmetadata_xxx.test"]
                    return rs.Primary.Attributes["name"], nil // "fully_qualified_name" for nested entities
                },
            },
        },
    })
}
```

Add extra `TestAcc*` functions (`TestAccXxxResourceWithOwners`, ...) for optional fields and variants;
each uses a fresh `testRandName`.

## `ImportStateVerifyIgnore`

| Field | Resource | Why |
|---|---|---|
| `parents` | `openmetadata_team` | OM adds the implicit root `Organisation` parent |
| `rules` | `openmetadata_policy` | Not read back — JSON key reordering |
| `connection_json` | service resources | Sensitive — API masks credentials |
| `source_config_json` | `openmetadata_ingestion_pipeline` | OM stores it with defaults filled in |

Add any new write-only field to this list in its test.

## Import ID

Top-level entities import by `name`; nested ones (tag, glossary_term, ingestion_pipeline) by
`fully_qualified_name`. Always pull it from state with `ImportStateIdFunc` rather than hardcoding.

## Config helpers

Prepend `testProviderBlock()` and interpolate strings with `%q`. Chain dependent resources in the
same config string:

```go
func testAccTagConfig(classificationName, tagName string) string {
    return fmt.Sprintf(`
%s

resource "openmetadata_classification" "parent" {
  name = %q
}

resource "openmetadata_tag" "test" {
  name           = %q
  classification = openmetadata_classification.parent.name
}
`, testProviderBlock(), classificationName, tagName)
}
```

## Fake server tests (no live OpenMetadata)

`fake_om_test.go` defines `fakeOM`, an in-memory stand-in for the API used by the ingestion pipeline
resource and data source (UUIDs, FQNs, server defaults, `{code, message}` errors). Tests built on it
(see `ingestion_pipeline_fake_test.go`) run the full plan/apply/import/destroy cycle under `make test`
and catch provider-side bugs like "inconsistent result after apply". They skip when there is neither a
`terraform` CLI on `PATH` nor a `TF_ACC_TERRAFORM_PATH`. It does not replace acceptance tests, which
validate against the real server. Keep every helper in these files used — the full-module lint fails
on unused ones.

## Running tests

```bash
make test                                                   # unit + fake-server tests
TF_ACC=1 go test -v -run TestAccXxxResource ./internal/provider/...     # one resource
TF_ACC=1 go test -v -count=1 -timeout 30m ./internal/provider/...       # all acceptance tests
make testacc            # full docker-compose lifecycle
make testacc-external   # against OPENMETADATA_HOST / OPENMETADATA_TOKEN
```
