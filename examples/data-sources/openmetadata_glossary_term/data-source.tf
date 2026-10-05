# Glossary terms are looked up by their fully qualified name: Glossary.TermName
# For nested terms use: Glossary.ParentTerm.ChildTerm
data "openmetadata_glossary_term" "revenue" {
  name = "BusinessGlossary.Revenue"
}

output "term_id" {
  value = data.openmetadata_glossary_term.revenue.id
}

output "term_glossary" {
  value = data.openmetadata_glossary_term.revenue.glossary
}

output "term_synonyms" {
  value = data.openmetadata_glossary_term.revenue.synonyms
}
