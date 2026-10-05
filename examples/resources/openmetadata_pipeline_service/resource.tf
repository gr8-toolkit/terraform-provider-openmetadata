resource "openmetadata_pipeline_service" "example" {
  name         = "MyPipelineService"
  display_name = "My Pipeline Service"
  description  = "Managed by Terraform."
  service_type = "Airflow"

  connection_json = jsonencode({
    config = {
      type = "Airflow"
    }
  })
}
