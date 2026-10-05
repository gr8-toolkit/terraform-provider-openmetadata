resource "openmetadata_dashboard_service" "example" {
  name         = "MyDashboardService"
  display_name = "My Dashboard Service"
  description  = "Managed by Terraform."
  service_type = "Superset"

  connection_json = jsonencode({
    config = {
      type = "Superset"
    }
  })
}
