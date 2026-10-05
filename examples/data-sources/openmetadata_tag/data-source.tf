# Tags are looked up by their fully qualified name: Classification.TagName
data "openmetadata_tag" "pii_sensitive" {
  name = "PII.Sensitive"
}

output "tag_id" {
  value = data.openmetadata_tag.pii_sensitive.id
}

output "tag_classification" {
  value = data.openmetadata_tag.pii_sensitive.classification
}
