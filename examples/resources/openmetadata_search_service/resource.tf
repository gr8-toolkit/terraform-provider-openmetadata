resource "openmetadata_search_service" "example" {
  name         = "MySearchService"
  display_name = "My Search Service"
  description  = "Managed by Terraform."
  service_type = "ElasticSearch"

  connection_json = jsonencode({
    config = {
      type = "ElasticSearch"
    }
  })
}
