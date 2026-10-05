// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClassificationDataSource(t *testing.T) {
	name := testRandName("cls")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClassificationDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_classification.test", "id",
						"openmetadata_classification.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_classification.test", "name",
						"openmetadata_classification.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_classification.test", "description",
						"openmetadata_classification.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_classification.test", "mutually_exclusive",
						"openmetadata_classification.test", "mutually_exclusive"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_classification.test", "fully_qualified_name",
						"openmetadata_classification.test", "fully_qualified_name"),
					resource.TestCheckResourceAttrSet("data.openmetadata_classification.test", "id"),
				),
			},
		},
	})
}

func TestAccClassificationDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccClassificationDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Classification not found`),
			},
		},
	})
}

func testAccClassificationDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_classification" "test" {
  name        = %q
  description = "Classification data source acceptance test"
}

data "openmetadata_classification" "test" {
  name = openmetadata_classification.test.name
}
`, testProviderBlock(), name)
}

func testAccClassificationDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_classification" "test" {
  name = "tfacc_cls_does_not_exist_99999"
}
`, testProviderBlock())
}
