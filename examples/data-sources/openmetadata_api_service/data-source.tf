data "openmetadata_api_service" "example" {
  name = "MyApiService"
}

output "service_id" {
  value = data.openmetadata_api_service.example.id
}

output "service_type" {
  value = data.openmetadata_api_service.example.service_type
}
