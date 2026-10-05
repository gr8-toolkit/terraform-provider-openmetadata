data "openmetadata_role" "data_steward" {
  name = "DataSteward"
}

output "role_id" {
  value = data.openmetadata_role.data_steward.id
}

output "role_policies" {
  value = data.openmetadata_role.data_steward.policies
}
