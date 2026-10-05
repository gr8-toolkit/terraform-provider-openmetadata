resource "openmetadata_metadata_service" "example" {
  name         = "MyMetadataService"
  display_name = "My Metadata Service"
  description  = "Managed by Terraform."
  service_type = "Atlas"

  connection_json = jsonencode({
    config = {
      type = "Atlas"
    }
  })
}
