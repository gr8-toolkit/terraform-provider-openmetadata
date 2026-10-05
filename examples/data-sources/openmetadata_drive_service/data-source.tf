data "openmetadata_drive_service" "example" {
  name = "MyDriveService"
}

output "service_id" {
  value = data.openmetadata_drive_service.example.id
}

output "service_type" {
  value = data.openmetadata_drive_service.example.service_type
}
