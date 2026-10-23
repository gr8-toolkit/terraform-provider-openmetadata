# Look up an ingestion pipeline by its fully qualified name (<service>.<pipeline>).
# The sourceConfig is not returned.
data "openmetadata_ingestion_pipeline" "postgres_metadata" {
  name = "ProductionPostgres.postgres_metadata"
}

output "pipeline_id" {
  value = data.openmetadata_ingestion_pipeline.postgres_metadata.id
}

output "pipeline_type" {
  value = data.openmetadata_ingestion_pipeline.postgres_metadata.pipeline_type
}

output "pipeline_schedule" {
  value = data.openmetadata_ingestion_pipeline.postgres_metadata.airflow_config.schedule_interval
}

output "pipeline_deployed" {
  value = data.openmetadata_ingestion_pipeline.postgres_metadata.deployed
}
