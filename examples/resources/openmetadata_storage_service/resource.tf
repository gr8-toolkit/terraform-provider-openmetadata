resource "openmetadata_storage_service" "example" {
  name         = "MyStorageService"
  display_name = "My Storage Service"
  description  = "Managed by Terraform."
  service_type = "S3"

  connection_json = jsonencode({
    config = {
      type = "S3"
    }
  })
}
