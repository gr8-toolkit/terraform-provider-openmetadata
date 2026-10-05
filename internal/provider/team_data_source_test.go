// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamDataSource(t *testing.T) {
	name := testRandName("team")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Data source returns the same id and name as the resource.
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_team.test", "id",
						"openmetadata_team.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_team.test", "name",
						"openmetadata_team.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_team.test", "team_type",
						"openmetadata_team.test", "team_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_team.test", "description",
						"openmetadata_team.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_team.test", "fully_qualified_name",
						"openmetadata_team.test", "fully_qualified_name"),
					resource.TestCheckResourceAttrSet("data.openmetadata_team.test", "id"),
				),
			},
		},
	})
}

// TestAccTeamDataSourceNotFound verifies the data source returns a clear error
// for a team that does not exist.
func TestAccTeamDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTeamDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Team not found`),
			},
		},
	})
}

func testAccTeamDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "test" {
  name        = %q
  description = "Team data source acceptance test"
  team_type   = "Department"
  is_joinable = false
}

data "openmetadata_team" "test" {
  name = openmetadata_team.test.name
}
`, testProviderBlock(), name)
}

func testAccTeamDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_team" "test" {
  name = "tfacc_team_does_not_exist_99999"
}
`, testProviderBlock())
}
