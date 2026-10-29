# The asset must already exist, e.g. created by an ingestion pipeline that
# ran against the "warehouse" database service. Terraform only manages the
# attributes set below; everything else keeps its ingested value.
resource "openmetadata_data_asset" "orders" {
  type                 = "table"
  fully_qualified_name = "warehouse.analytics.public.orders"

  display_name = "Orders"
  description  = "One row per customer order, refreshed hourly."

  owners = [
    {
      id   = openmetadata_team.data_engineering.id
      type = "team"
    }
  ]

  domains        = [openmetadata_domain.analytics.name]
  tags           = ["PII.Sensitive"]
  glossary_terms = ["Finance.Revenue"]

  # Custom property values; the object replaces all existing values.
  extension_json = jsonencode({
    dataRetentionDays = 365
  })
}
