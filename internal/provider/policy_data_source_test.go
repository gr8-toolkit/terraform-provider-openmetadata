// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPolicyDataSource(t *testing.T) {
	name := testRandName("pol")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_policy.test", "id",
						"openmetadata_policy.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_policy.test", "name",
						"openmetadata_policy.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_policy.test", "description",
						"openmetadata_policy.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_policy.test", "enabled",
						"openmetadata_policy.test", "enabled"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_policy.test", "fully_qualified_name",
						"openmetadata_policy.test", "fully_qualified_name"),
					// rules is returned by the data source (read from API); verify it is set.
					// Unlike the resource, the data source does not keep rules write-only.
					resource.TestCheckResourceAttrSet("data.openmetadata_policy.test", "rules"),
				),
			},
		},
	})
}

func TestAccPolicyDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Policy not found`),
			},
		},
	})
}

func testAccPolicyDataSourceConfig(name string) string {
	rules := `[{"name":"allow-view","effect":"allow","operations":["ViewAll"],"resources":["All"]}]`
	return fmt.Sprintf(`
%s

resource "openmetadata_policy" "test" {
  name        = %q
  description = "Policy data source acceptance test"
  rules       = %q
}

data "openmetadata_policy" "test" {
  name = openmetadata_policy.test.name
}
`, testProviderBlock(), name, rules)
}

func testAccPolicyDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_policy" "test" {
  name = "tfacc_pol_does_not_exist_99999"
}
`, testProviderBlock())
}
