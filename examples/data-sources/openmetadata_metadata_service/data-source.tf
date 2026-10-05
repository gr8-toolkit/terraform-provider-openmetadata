data "openmetadata_metadata_service" "example" {
  name = "MyMetadataService"
}

output "service_id" {
  value = data.openmetadata_metadata_service.example.id
}

output "service_type" {
  value = data.openmetadata_metadata_service.example.service_type
}
