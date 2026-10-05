data "openmetadata_messaging_service" "example" {
  name = "MyMessagingService"
}

output "service_id" {
  value = data.openmetadata_messaging_service.example.id
}

output "service_type" {
  value = data.openmetadata_messaging_service.example.service_type
}
