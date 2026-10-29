# Read the current metadata of an ingested table.
data "openmetadata_data_asset" "orders" {
  type                 = "table"
  fully_qualified_name = "warehouse.analytics.public.orders"
}

output "orders_description" {
  value = data.openmetadata_data_asset.orders.description
}

output "orders_owners" {
  value = data.openmetadata_data_asset.orders.owners
}

output "orders_tags" {
  value = data.openmetadata_data_asset.orders.tags
}
