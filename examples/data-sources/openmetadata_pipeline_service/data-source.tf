data "openmetadata_pipeline_service" "example" {
  name = "MyPipelineService"
}

output "service_id" {
  value = data.openmetadata_pipeline_service.example.id
}

output "service_type" {
  value = data.openmetadata_pipeline_service.example.service_type
}
