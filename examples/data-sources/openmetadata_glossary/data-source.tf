data "openmetadata_glossary" "business" {
  name = "BusinessGlossary"
}

output "glossary_id" {
  value = data.openmetadata_glossary.business.id
}

output "glossary_fqn" {
  value = data.openmetadata_glossary.business.fully_qualified_name
}
