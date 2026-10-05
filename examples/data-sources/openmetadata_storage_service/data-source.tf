data "openmetadata_storage_service" "example" {
  name = "MyStorageService"
}

output "service_id" {
  value = data.openmetadata_storage_service.example.id
}

output "service_type" {
  value = data.openmetadata_storage_service.example.service_type
}
