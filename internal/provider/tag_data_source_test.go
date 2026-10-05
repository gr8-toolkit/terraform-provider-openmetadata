// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTagDataSource(t *testing.T) {
	clsName := testRandName("cls")
	tagName := testRandName("tag")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagDataSourceConfig(clsName, tagName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_tag.test", "id",
						"openmetadata_tag.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_tag.test", "fully_qualified_name",
						"openmetadata_tag.test", "fully_qualified_name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_tag.test", "description",
						"openmetadata_tag.test", "description"),
					// The data source reads classification from the API entity ref;
					// the resource stores it as the plain name string.
					resource.TestCheckResourceAttr(
						"data.openmetadata_tag.test", "classification", clsName),
					resource.TestCheckResourceAttrSet("data.openmetadata_tag.test", "id"),
				),
			},
		},
	})
}

func TestAccTagDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTagDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Tag not found`),
			},
		},
	})
}

func testAccTagDataSourceConfig(clsName, tagName string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_classification" "test" {
  name        = %q
  description = "Parent classification for tag data source test"
}

resource "openmetadata_tag" "test" {
  name           = %q
  description    = "Tag data source acceptance test"
  classification = openmetadata_classification.test.name
}

# Tags are looked up by FQN (Classification.TagName).
data "openmetadata_tag" "test" {
  name = openmetadata_tag.test.fully_qualified_name
}
`, testProviderBlock(), clsName, tagName)
}

func testAccTagDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_tag" "test" {
  name = "tfacc_cls_99999.tfacc_tag_does_not_exist_99999"
}
`, testProviderBlock())
}
