data "openmetadata_classification" "pii" {
  name = "PII"
}

output "classification_id" {
  value = data.openmetadata_classification.pii.id
}

output "mutually_exclusive" {
  value = data.openmetadata_classification.pii.mutually_exclusive
}
