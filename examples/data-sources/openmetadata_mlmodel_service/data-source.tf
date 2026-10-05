data "openmetadata_mlmodel_service" "example" {
  name = "MyMLModelService"
}

output "service_id" {
  value = data.openmetadata_mlmodel_service.example.id
}

output "service_type" {
  value = data.openmetadata_mlmodel_service.example.service_type
}
