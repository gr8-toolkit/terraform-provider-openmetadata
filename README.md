# Terraform Provider for OpenMetadata

[![CI](https://github.com/gr8-toolkit/terraform-provider-openmetadata/actions/workflows/ci.yml/badge.svg)](https://github.com/gr8-toolkit/terraform-provider-openmetadata/actions/workflows/ci.yml)
[![Acceptance Tests](https://github.com/gr8-toolkit/terraform-provider-openmetadata/actions/workflows/acceptance.yml/badge.svg)](https://github.com/gr8-toolkit/terraform-provider-openmetadata/actions/workflows/acceptance.yml)
[![prek](https://img.shields.io/badge/hooks-prek-blue)](https://github.com/j178/prek)
[![Terraform Registry](https://img.shields.io/badge/Terraform_Registry-gr8--toolkit%2Fopenmetadata-7B42BC?logo=terraform)](https://registry.terraform.io/providers/gr8-toolkit/openmetadata/latest)
[![OpenTofu Registry](https://img.shields.io/badge/OpenTofu_Registry-gr8--toolkit%2Fopenmetadata-FFDA18?logo=opentofu&logoColor=000)](https://search.opentofu.org/provider/gr8-toolkit/openmetadata/latest)

A Terraform/OpenTofu provider for managing [OpenMetadata](https://open-metadata.org/) resources
declaratively — teams, classifications, tags, glossaries, policies, roles, domains, and all
service connection types.

OpenMetadata is an open-source metadata platform. This provider lets you version-control and
automate your OpenMetadata configuration the same way you manage the rest of your infrastructure.

## Requirements

| Tool | Minimum version |
|------|-----------------|
| [Terraform](https://developer.hashicorp.com/terraform/downloads) or [OpenTofu](https://opentofu.org/docs/intro/install/) | 1.0 |
| [Go](https://golang.org/doc/install) | 1.25 (only for building from source) |
| [Docker + Compose v2](https://docs.docker.com/compose/) | any recent (only for running acceptance tests) |

## Using the Provider

```hcl
terraform {
  required_providers {
    openmetadata = {
      source  = "gr8-toolkit/openmetadata"
      version = "~> 0.1"
    }
  }
}

provider "openmetadata" {
  host  = "https://openmetadata.example.com"
  token = var.openmetadata_token
}
```

Provider configuration can also come from environment variables:

| Attribute | Environment variable    | Default    |
|-----------|-------------------------|------------|
| `host`    | `OPENMETADATA_HOST`     | (required) |
| `token`   | `OPENMETADATA_TOKEN`    | (required) |

## Resources

### Governance

| Resource | Description |
|----------|-------------|
| `openmetadata_team` | Teams (Group, Department, Division, BusinessUnit) |
| `openmetadata_classification` | Tag classifications (categories) |
| `openmetadata_tag` | Tags within a classification |
| `openmetadata_glossary` | Business glossaries |
| `openmetadata_glossary_term` | Terms within a glossary |
| `openmetadata_policy` | Access control policies |
| `openmetadata_role` | Roles referencing policies |
| `openmetadata_domain` | Domains (Source-aligned, Consumer-aligned, Aggregate) |

### Services

| Resource | Description |
|----------|-------------|
| `openmetadata_database_service` | Database service connections (Snowflake, BigQuery, Postgres, …) |
| `openmetadata_api_service` | REST / Webhook API service connections |
| `openmetadata_messaging_service` | Messaging service connections (Kafka, Kinesis, PubSub, …) |
| `openmetadata_dashboard_service` | Dashboard service connections (Looker, Tableau, Superset, …) |
| `openmetadata_pipeline_service` | Pipeline service connections (Airflow, dbt, Glue, …) |
| `openmetadata_mlmodel_service` | ML model service connections (MLflow, SageMaker, …) |
| `openmetadata_storage_service` | Storage service connections (S3, GCS, Azure Blob, …) |
| `openmetadata_search_service` | Search service connections (Elasticsearch, OpenSearch, …) |
| `openmetadata_metadata_service` | Metadata service connections (Atlas, Alation, …) |
| `openmetadata_drive_service` | Drive service connections (Google Drive, …) |

## Data Sources

Every resource above has a corresponding data source for read-only lookups:

```text
openmetadata_team              openmetadata_classification    openmetadata_tag
openmetadata_glossary          openmetadata_glossary_term     openmetadata_policy
openmetadata_role              openmetadata_domain            openmetadata_database_service
openmetadata_api_service       openmetadata_messaging_service openmetadata_dashboard_service
openmetadata_pipeline_service  openmetadata_mlmodel_service   openmetadata_storage_service
openmetadata_search_service    openmetadata_metadata_service  openmetadata_drive_service
```

## Examples

```hcl
# Governance — classification, tag, glossary, team, domain

resource "openmetadata_classification" "pii" {
  name        = "PII"
  description = "Personally Identifiable Information"
}

resource "openmetadata_tag" "email" {
  name           = "Email"
  classification = openmetadata_classification.pii.name
  description    = "Email address fields"
}

resource "openmetadata_glossary" "finance" {
  name        = "Finance"
  description = "Finance business terms"
}

resource "openmetadata_glossary_term" "revenue" {
  name        = "Revenue"
  glossary    = openmetadata_glossary.finance.name
  description = "Total recognised revenue"
}

resource "openmetadata_team" "data_engineering" {
  name         = "DataEngineering"
  display_name = "Data Engineering"
  team_type    = "Group"
}

resource "openmetadata_domain" "analytics" {
  name         = "Analytics"
  display_name = "Analytics"
  domain_type  = "Consumer-aligned"
}

# Access control — policy and role

resource "openmetadata_policy" "read_only" {
  name        = "ReadOnlyPolicy"
  description = "Allow read access to all metadata"
  rules = jsonencode([
    {
      name      = "AllowRead"
      effect    = "allow"
      resources = [".*"]
      operations = ["ViewAll"]
    }
  ])
}

resource "openmetadata_role" "viewer" {
  name     = "ViewerRole"
  policies = [openmetadata_policy.read_only.name]
}

# Service connections

resource "openmetadata_database_service" "warehouse" {
  name         = "DataWarehouse"
  service_type = "Snowflake"
  connection_json = jsonencode({
    config = {
      type     = "Snowflake"
      username = "svc_terraform"
      password = var.snowflake_password
      account  = "my-account"
      database = "ANALYTICS"
      warehouse = "COMPUTE_WH"
    }
  })
}

resource "openmetadata_messaging_service" "events" {
  name         = "KafkaEvents"
  service_type = "Kafka"
  connection_json = jsonencode({
    config = {
      type            = "Kafka"
      bootstrapServers = "kafka.internal:9092"
    }
  })
}

resource "openmetadata_pipeline_service" "orchestration" {
  name         = "AirflowProd"
  service_type = "Airflow"
}
```

## Import

All resources support `terraform import`. Top-level resources use the entity `name`;
nested resources (tag, glossary_term) use the `fully_qualified_name`.

```bash
# Governance
terraform import openmetadata_team.example           "DataEngineering"
terraform import openmetadata_classification.example "PII"
terraform import openmetadata_tag.example            "PII.Email"
terraform import openmetadata_glossary.example       "Finance"
terraform import openmetadata_glossary_term.example  "Finance.Revenue"
terraform import openmetadata_policy.example         "ReadOnlyPolicy"
terraform import openmetadata_role.example           "ViewerRole"
terraform import openmetadata_domain.example         "Analytics"

# Services
terraform import openmetadata_database_service.example   "DataWarehouse"
terraform import openmetadata_api_service.example        "PaymentsAPI"
terraform import openmetadata_messaging_service.example  "KafkaEvents"
terraform import openmetadata_dashboard_service.example  "SupersetProd"
terraform import openmetadata_pipeline_service.example   "AirflowProd"
terraform import openmetadata_mlmodel_service.example    "MLflowProd"
terraform import openmetadata_storage_service.example    "S3DataLake"
terraform import openmetadata_search_service.example     "ElasticsearchProd"
terraform import openmetadata_metadata_service.example   "AtlasProd"
terraform import openmetadata_drive_service.example      "GDriveProd"
```

## Building from Source

```bash
git clone https://github.com/gr8-toolkit/terraform-provider-openmetadata.git
cd terraform-provider-openmetadata

make build    # Compile provider binary
make install  # Build + install to ~/.terraform.d/plugins/
```

## Development

```bash
make build   # Compile
make fmt     # go fmt ./...
make lint    # golangci-lint run ./...
make test    # Unit tests (no live OpenMetadata instance needed)
make docs    # Regenerate docs/ via tfplugindocs
make deps    # go mod tidy
```

Gate before every commit:

```bash
make fmt && make lint && make build && make test
```

### Running Acceptance Tests

Acceptance tests require a live OpenMetadata instance. The full stack is managed
automatically via Docker Compose (~20 min, 8 GB RAM):

```bash
# Default version matrix (1.12.4, 2.0.3)
make testacc

# Specific version
make testacc OM_VERSION=2.0.3

# All versions in docker/test/versions (CI matrix)
make testacc-all

# Against an external instance you already have running
export OPENMETADATA_HOST=http://localhost:8585
export OPENMETADATA_TOKEN=<jwt-admin-token>
make testacc-external

# Single resource during development
TF_ACC=1 go test -v -run TestAccDatabaseServiceResource ./internal/provider/...
```

CI runs the full acceptance suite against these OpenMetadata versions on every push:

| Version |
|---------|
| 1.12.4  |
| 2.0.3   |

## Documentation

Docs under `docs/` are auto-generated by
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) — do not edit them directly.
Regenerate after any schema or example change:

```bash
make docs
```

Per-resource examples live in `examples/resources/openmetadata_<name>/resource.tf`.

## Repository Layout

```text
internal/
  client/       # Thin net/http wrapper — Bearer auth, CRUD, 30 s timeout
  datasources/  # Read-only data sources (one file per entity, 18 total)
  provider/     # Provider config + acceptance tests (one test file per resource)
  resources/    # Managed resources (one file per entity, 18 total)
examples/       # HCL usage examples consumed by tfplugindocs
docs/           # Auto-generated documentation — do not edit
docker/test/    # Docker Compose stack for acceptance tests
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
To report a vulnerability, see [SECURITY.md](SECURITY.md) — do not open a public issue.

## License

[Apache License 2.0](LICENSE)
