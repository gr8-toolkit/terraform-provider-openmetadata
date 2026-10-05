resource "openmetadata_mlmodel_service" "example" {
  name         = "MyMLModelService"
  display_name = "My MLModel Service"
  description  = "Managed by Terraform."
  service_type = "Mlflow"

  connection_json = jsonencode({
    config = {
      type = "Mlflow"
    }
  })
}
