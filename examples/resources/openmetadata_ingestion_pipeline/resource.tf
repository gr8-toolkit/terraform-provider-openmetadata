# The resource only manages the pipeline definition; it does not deploy it to
# the ingestion runner. Deploy from the OpenMetadata UI or API.

resource "openmetadata_database_service" "postgres" {
  name         = "ProductionPostgres"
  service_type = "Postgres"

  connection_json = jsonencode({
    config = {
      type     = "Postgres"
      scheme   = "postgresql+psycopg2"
      username = "readonly_user"
      hostPort = "db.example.com:5432"
      database = "production"
    }
  })
}

# ── Minimal metadata pipeline ─────────────────────────────────────────────────
#
# `service` is resolved by name to the UUID the API expects, so referencing
# `.name` of a managed service also creates the implicit dependency.

resource "openmetadata_ingestion_pipeline" "metadata" {
  name          = "postgres_metadata"
  pipeline_type = "metadata"
  service = {
    type = "databaseService"
    name = openmetadata_database_service.postgres.name
  }

  # The structure of the source config depends on `pipeline_type`. See the JSON
  # schemas for each workflow:
  # https://github.com/open-metadata/OpenMetadata/tree/main/openmetadata-spec/src/main/resources/json/schema/metadataIngestion
  source_config_json = jsonencode({
    config = {
      type = "DatabaseMetadata"
    }
  })
}

# ── Scheduled pipeline with filters, retries and ownership ────────────────────

resource "openmetadata_ingestion_pipeline" "metadata_scheduled" {
  name         = "postgres_metadata_hourly"
  display_name = "Postgres metadata (hourly)"
  description  = "Ingests table and column metadata from the production Postgres every hour."

  pipeline_type = "metadata"
  service = {
    type = "databaseService"
    name = openmetadata_database_service.postgres.name
  }

  source_config_json = jsonencode({
    config = {
      type                = "DatabaseMetadata"
      markDeletedTables   = true
      includeTables       = true
      includeViews        = true
      schemaFilterPattern = { includes = ["^public$", "^analytics$"] }
      tableFilterPattern  = { excludes = ["^tmp_.*"] }
    }
  })

  airflow_config = {
    schedule_interval = "0 * * * *" # cron format; leave unset for manual runs only
    retries           = 3
    retry_delay       = 300 # seconds
    pipeline_catchup  = false
  }

  logger_level           = "INFO" # DEBUG | INFO | WARN | ERROR
  raise_on_error         = true
  enable_streamable_logs = true

  domains = ["Engineering"]

  owners = [
    {
      id   = openmetadata_team.data_engineering.id
      type = "team"
    }
  ]
}

resource "openmetadata_team" "data_engineering" {
  name      = "DataEngineering"
  team_type = "Group"
}

# ── Other workflow types ──────────────────────────────────────────────────────
#
# One resource covers every workflow type. Pick the matching `pipeline_type` and
# `source_config_json.config.type`:
#
#   metadata            DatabaseMetadata
#   usage               DatabaseUsage
#   lineage             DatabaseLineage
#   profiler            Profiler
#   autoClassification  AutoClassification

resource "openmetadata_ingestion_pipeline" "profiler" {
  name          = "postgres_profiler"
  pipeline_type = "profiler"
  service = {
    type = "databaseService"
    name = openmetadata_database_service.postgres.name
  }

  source_config_json = jsonencode({
    config = {
      type               = "Profiler"
      generateSampleData = true
    }
  })

  airflow_config = {
    schedule_interval = "0 2 * * *" # nightly at 02:00
    pipeline_timezone = "UTC"
  }
}

# ── Import ────────────────────────────────────────────────────────────────────
#
# Import by fully qualified name (<service>.<pipeline>):
#
#   terraform import openmetadata_ingestion_pipeline.metadata ProductionPostgres.postgres_metadata
#
# `source_config_json` is not read back from the API (OpenMetadata fills in
# defaults when storing it), so after an import set it in your configuration
# and apply once. Drift in it is not detected.
