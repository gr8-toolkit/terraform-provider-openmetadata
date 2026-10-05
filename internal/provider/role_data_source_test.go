// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleDataSource(t *testing.T) {
	policyName := testRandName("pol")
	roleName := testRandName("role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleDataSourceConfig(policyName, roleName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_role.test", "id",
						"openmetadata_role.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_role.test", "name",
						"openmetadata_role.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_role.test", "description",
						"openmetadata_role.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_role.test", "fully_qualified_name",
						"openmetadata_role.test", "fully_qualified_name"),
					resource.TestCheckResourceAttr("data.openmetadata_role.test", "policies.#", "1"),
					resource.TestCheckResourceAttrSet("data.openmetadata_role.test", "id"),
				),
			},
		},
	})
}

func TestAccRoleDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Role not found`),
			},
		},
	})
}

func testAccRoleDataSourceConfig(policyName, roleName string) string {
	rules := `[{"name":"allow-view","effect":"allow","operations":["ViewAll"],"resources":["All"]}]`
	return fmt.Sprintf(`
%s

resource "openmetadata_policy" "test" {
  name        = %q
  description = "Policy for role data source test"
  rules       = %q
}

resource "openmetadata_role" "test" {
  name        = %q
  description = "Role data source acceptance test"
  policies    = [openmetadata_policy.test.name]
}

data "openmetadata_role" "test" {
  name = openmetadata_role.test.name
}
`, testProviderBlock(), policyName, rules, roleName)
}

func testAccRoleDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_role" "test" {
  name = "tfacc_role_does_not_exist_99999"
}
`, testProviderBlock())
}
