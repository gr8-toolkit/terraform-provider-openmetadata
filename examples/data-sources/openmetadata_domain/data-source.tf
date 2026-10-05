data "openmetadata_domain" "analytics" {
  name = "Analytics"
}

output "domain_id" {
  value = data.openmetadata_domain.analytics.id
}

output "domain_type" {
  value = data.openmetadata_domain.analytics.domain_type
}

output "domain_fqn" {
  value = data.openmetadata_domain.analytics.fully_qualified_name
}
