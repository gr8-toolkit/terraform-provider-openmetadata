resource "openmetadata_api_service" "example" {
  name         = "MyRestAPI"
  display_name = "My REST API"
  description  = "Managed by Terraform."
  service_type = "Rest"

  # connection_json is optional. The required fields vary by service_type and
  # OpenMetadata version — consult your instance's Swagger UI at {host}/swagger.html
  # under PUT /api/v1/services/apiServices for the exact schema.
  #
  # connection_json = jsonencode({
  #   config = {
  #     type = "Rest"
  #     # ... service-type-specific fields ...
  #   }
  # })
}
