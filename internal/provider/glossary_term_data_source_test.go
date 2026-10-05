// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGlossaryTermDataSource(t *testing.T) {
	glossaryName := testRandName("glos")
	termName := testRandName("term")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGlossaryTermDataSourceConfig(glossaryName, termName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary_term.test", "id",
						"openmetadata_glossary_term.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary_term.test", "description",
						"openmetadata_glossary_term.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_glossary_term.test", "fully_qualified_name",
						"openmetadata_glossary_term.test", "fully_qualified_name"),
					// glossary is returned from the API as an entity ref FQN.
					resource.TestCheckResourceAttr(
						"data.openmetadata_glossary_term.test", "glossary", glossaryName),
					resource.TestCheckResourceAttrSet("data.openmetadata_glossary_term.test", "id"),
				),
			},
		},
	})
}

func TestAccGlossaryTermDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGlossaryTermDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Glossary term not found`),
			},
		},
	})
}

func testAccGlossaryTermDataSourceConfig(glossaryName, termName string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_glossary" "test" {
  name        = %q
  description = "Parent glossary for term data source test"
}

resource "openmetadata_glossary_term" "test" {
  name        = %q
  description = "Glossary term data source acceptance test"
  glossary    = openmetadata_glossary.test.name
}

# Glossary terms are looked up by FQN (Glossary.TermName).
data "openmetadata_glossary_term" "test" {
  name = openmetadata_glossary_term.test.fully_qualified_name
}
`, testProviderBlock(), glossaryName, termName)
}

func testAccGlossaryTermDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_glossary_term" "test" {
  name = "tfacc_glos_99999.tfacc_term_does_not_exist_99999"
}
`, testProviderBlock())
}
