// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGlossaryDataSource(t *testing.T) {
	name := testRandName("glos")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGlossaryDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary.test", "id",
						"openmetadata_glossary.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary.test", "name",
						"openmetadata_glossary.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary.test", "description",
						"openmetadata_glossary.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary.test", "mutually_exclusive",
						"openmetadata_glossary.test", "mutually_exclusive"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary.test", "fully_qualified_name",
						"openmetadata_glossary.test", "fully_qualified_name"),
					resource.TestCheckResourceAttrSet("data.openmetadata_glossary.test", "id"),
				),
			},
		},
	})
}

func TestAccGlossaryDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGlossaryDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Glossary not found`),
			},
		},
	})
}

func testAccGlossaryDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_glossary" "test" {
  name        = %q
  description = "Glossary data source acceptance test"
}

data "openmetadata_glossary" "test" {
  name = openmetadata_glossary.test.name
}
`, testProviderBlock(), name)
}

func testAccGlossaryDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_glossary" "test" {
  name = "tfacc_glos_does_not_exist_99999"
}
`, testProviderBlock())
}
