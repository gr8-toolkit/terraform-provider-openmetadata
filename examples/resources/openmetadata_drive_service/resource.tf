resource "openmetadata_drive_service" "example" {
  name         = "MyDriveService"
  display_name = "My Drive Service"
  description  = "Managed by Terraform."
  service_type = "GoogleDrive"

  connection_json = jsonencode({
    config = {
      type = "GoogleDrive"
    }
  })
}
