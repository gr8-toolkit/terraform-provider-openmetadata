resource "openmetadata_messaging_service" "example" {
  name         = "MyMessagingService"
  display_name = "My Messaging Service"
  description  = "Managed by Terraform."
  service_type = "Kafka"

  connection_json = jsonencode({
    config = {
      type = "Kafka"
    }
  })
}
