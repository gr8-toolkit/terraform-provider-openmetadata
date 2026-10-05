data "openmetadata_policy" "data_access" {
  name = "DataAccessControl"
}

output "policy_id" {
  value = data.openmetadata_policy.data_access.id
}

output "policy_enabled" {
  value = data.openmetadata_policy.data_access.enabled
}

# rules is returned as a JSON string; use jsondecode() to inspect it in HCL
output "policy_rules" {
  value = data.openmetadata_policy.data_access.rules
}
