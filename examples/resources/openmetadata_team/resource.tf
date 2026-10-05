# ── Minimal team ──────────────────────────────────────────────────────────────

resource "openmetadata_team" "data_engineering" {
  name         = "DataEngineering"
  display_name = "Data Engineering"
  description  = "Owns ETL pipelines, data models, and data infrastructure."
  team_type    = "Group"
  is_joinable  = false
}

# ── Two-level hierarchy: Division → Department ────────────────────────────────
#
# The `parents` attribute accepts either:
#   - A team UUID  (openmetadata_team.foo.id)   — implicit dependency, no extra lookup
#   - A team name  (openmetadata_team.foo.name) — provider resolves to UUID at apply time
#
# Using .id is recommended when both teams are managed in the same configuration
# because it creates an implicit dependency and avoids an extra API call.

resource "openmetadata_team" "platform_division" {
  name         = "Platform"
  display_name = "Platform"
  description  = "Platform division."
  team_type    = "Division"
  is_joinable  = false
}

resource "openmetadata_team" "data_department" {
  name         = "Data"
  display_name = "Data"
  description  = "Data sub-department within Platform."
  team_type    = "Department"
  is_joinable  = false

  # Reference the parent by UUID — Terraform creates Platform first automatically.
  parents = [openmetadata_team.platform_division.id]
}

# ── Three-level hierarchy: Division → Department → Group ──────────────────────

resource "openmetadata_team" "engineering_division" {
  name         = "Engineering"
  display_name = "Engineering"
  description  = "Top-level engineering division."
  team_type    = "Division"
  is_joinable  = false
}

resource "openmetadata_team" "backend_department" {
  name         = "Backend"
  display_name = "Backend"
  description  = "Backend systems department."
  team_type    = "Department"
  is_joinable  = false
  parents      = [openmetadata_team.engineering_division.id]
}

resource "openmetadata_team" "api_team" {
  name         = "API"
  display_name = "API Team"
  description  = "Owns the public API surface."
  team_type    = "Group"
  is_joinable  = true
  parents      = [openmetadata_team.backend_department.id]
}

# ── Team with multiple parents ─────────────────────────────────────────────────
#
# A team can belong to more than one parent. List all parent UUIDs (or names).
# OM places teams without an explicit parent under the root Organisation team.

resource "openmetadata_team" "shared_services" {
  name         = "SharedServices"
  display_name = "Shared Services"
  description  = "Cross-functional team shared across divisions."
  team_type    = "Department"
  is_joinable  = false
  parents = [
    openmetadata_team.platform_division.id,
    openmetadata_team.engineering_division.id,
  ]
}

# ── Re-parenting an existing team ─────────────────────────────────────────────
#
# To move a team to a different parent, update the `parents` attribute and
# run `terraform apply`. The provider pushes the corrected relationship to the
# OpenMetadata API.
#
# Note: `parents` is write-only. Manual changes made in the OM UI are not
# automatically detected by `terraform plan`. Update the config and apply
# to restore the desired hierarchy.
