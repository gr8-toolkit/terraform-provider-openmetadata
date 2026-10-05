data "openmetadata_dashboard_service" "example" {
  name = "MyDashboardService"
}

output "service_id" {
  value = data.openmetadata_dashboard_service.example.id
}

output "service_type" {
  value = data.openmetadata_dashboard_service.example.service_type
}
