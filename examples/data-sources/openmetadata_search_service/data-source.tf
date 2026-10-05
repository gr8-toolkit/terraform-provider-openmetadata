data "openmetadata_search_service" "example" {
  name = "MySearchService"
}

output "service_id" {
  value = data.openmetadata_search_service.example.id
}

output "service_type" {
  value = data.openmetadata_search_service.example.service_type
}
