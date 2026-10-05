data "openmetadata_team" "data_engineering" {
  name = "DataEngineering"
}

output "team_id" {
  value = data.openmetadata_team.data_engineering.id
}

output "team_type" {
  value = data.openmetadata_team.data_engineering.team_type
}
