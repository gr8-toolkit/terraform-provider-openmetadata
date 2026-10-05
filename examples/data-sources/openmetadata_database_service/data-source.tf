data "openmetadata_database_service" "prod_warehouse" {
  name = "ProdWarehouse"
}

output "service_id" {
  value = data.openmetadata_database_service.prod_warehouse.id
}

output "service_type" {
  value = data.openmetadata_database_service.prod_warehouse.service_type
}
