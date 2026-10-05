---
inclusion: fileMatch
fileMatchPattern: "internal/provider/*_resource_test.go"
---

# Acceptance Test Patterns

## Shared Infrastructure (`internal/provider/provider_test.go`)

```go
// In-process provider server — include in every TestCase, no external binary needed
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "openmetadata": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// Abort early if required env vars are missing
func testAccPreCheck(t *testing.T)
// Checks: OPENMETADATA_HOST, OPENMETADATA_TOKEN

// Returns a collision-safe entity name: "tfacc_{prefix}_{5-digit-random}"
// prefix must be ≤ 6 chars to stay within OM name length limits
func testRandName(prefix string) string

// Returns `provider "openmetadata" {}` — reads OPENMETADATA_HOST/TOKEN from env
func testProviderBlock() string
```

All tests are in `package provider_test` and are gated by `TF_ACC=1`.

## Required Three-Step TestCase Structure

Every `TestAcc*` function must cover these three steps in order:

```go
func TestAccXxxResource(t *testing.T) {
    name := testRandName("xxx") // prefix ≤ 6 chars

    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{

            // ── Step 1: Create + Read ─────────────────────────────────────
            {
                Config: testAccXxxConfig(name, "Initial description"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("openmetadata_xxx.test", "name", name),
                    resource.TestCheckResourceAttr("openmetadata_xxx.test", "description", "Initial description"),
                    resource.TestCheckResourceAttrSet("openmetadata_xxx.test", "id"),
                    resource.TestCheckResourceAttrSet("openmetadata_xxx.test", "fully_qualified_name"),
                    // check every non-computed attribute
                ),
            },

            // ── Step 2: Update ────────────────────────────────────────────
            // Change at least one mutable attribute; verify it took effect
            {
                Config: testAccXxxConfig(name, "Updated description"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("openmetadata_xxx.test", "description", "Updated description"),
                ),
            },

            // ── Step 3: Import ────────────────────────────────────────────
            {
                ResourceName:      "openmetadata_xxx.test",
                ImportState:       true,
                ImportStateVerify: true,
                // List write-only fields here (see table below)
                ImportStateVerifyIgnore: []string{},
                ImportStateIdFunc: func(s *terraform.State) (string, error) {
                    rs := s.RootModule().Resources["openmetadata_xxx.test"]
                    return rs.Primary.Attributes["name"], nil
                    // Use "fully_qualified_name" for nested entities (tag, glossary_term)
                },
            },
        },
    })
}
```

## `ImportStateVerifyIgnore` — Which Fields and Why

| Field | Resource | Why it's ignored |
|---|---|---|
| `"parents"` | `openmetadata_team` | OM adds the implicit root `Organisation` parent; can't round-trip to user-supplied names |
| `"rules"` | `openmetadata_policy` | Not read back from API — JSON key reordering causes inequality |
| `"connection_json"` | `openmetadata_database_service` | Sensitive — API masks credential fields; not read back |

Add any new write-only fields for new resources to this list in their test.

## Config Helper Pattern

Every config helper must prepend `testProviderBlock()` and use `%q` for string interpolation:

```go
func testAccXxxConfig(name, description string) string {
    return fmt.Sprintf(`
%s

resource "openmetadata_xxx" "test" {
  name        = %q
  description = %q
}
`, testProviderBlock(), name, description)
}
```

For resources that depend on other resources, chain them in the same config string:

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

## Import ID — Short Name vs FQN

| Entity type | Import ID to use | Example |
|---|---|---|
| Top-level (team, classification, glossary, policy, role, domain, database_service) | `name` attribute | `"my-team"` |
| Nested (tag, glossary_term) | `fully_qualified_name` attribute | `"MyClassification.my-tag"` |

Use `ImportStateIdFunc` to pull the right attribute from state rather than hardcoding:

```go
ImportStateIdFunc: func(s *terraform.State) (string, error) {
    rs := s.RootModule().Resources["openmetadata_xxx.test"]
    return rs.Primary.Attributes["fully_qualified_name"], nil
},
```

## Multiple Test Functions Per File

Add extra `TestAcc*` functions for optional fields or edge-case configurations:

```go
// Base test: covers the standard lifecycle
func TestAccXxxResource(t *testing.T) { ... }

// Supplemental test: covers an optional field or variant behaviour
func TestAccXxxResourceWithOwners(t *testing.T) { ... }
```

Each function is independent — use a fresh `testRandName` call in each.

## Running Tests

```bash
# Full lifecycle with docker-compose (~20 min, 8 GB RAM)
make testacc

# Against an already-running instance
export OPENMETADATA_HOST=http://localhost:8585
export OPENMETADATA_TOKEN=<jwt>
make testacc-external

# Single resource during development
TF_ACC=1 go test -v -run TestAccXxxResource ./internal/provider/...

# All acceptance tests, single pass, with timeout
TF_ACC=1 go test -v -count=1 -timeout 30m ./internal/provider/...
```

Unit tests (no live OM instance, no `TF_ACC`):

```bash
make test
# equivalent to: go test -v -count=1 ./...
```
