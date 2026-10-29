// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataAssetDataSource(t *testing.T) {
	svc := testRandName("assvc")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSeedTable(t, svc)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataAssetDataSourceConfig(testAccSeededTableFQN(svc)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_data_asset.test", "id",
						"openmetadata_data_asset.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_data_asset.test", "description",
						"openmetadata_data_asset.test", "description"),
					resource.TestCheckResourceAttr("data.openmetadata_data_asset.test", "name", "orders"),
					resource.TestCheckResourceAttr("data.openmetadata_data_asset.test", "tags.0", "PII.Sensitive"),
					resource.TestCheckResourceAttrSet("data.openmetadata_data_asset.test", "id"),
				),
			},
		},
	})
}

func TestAccDataAssetDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = "tfacc_missing.db.public.nothing"
}
`, testProviderBlock()),
				ExpectError: regexp.MustCompile(`Data asset not found`),
			},
		},
	})
}

func testAccDataAssetDataSourceConfig(fqn string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = %q
  description          = "Data asset data source acceptance test"
  tags                 = ["PII.Sensitive"]
}

data "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = openmetadata_data_asset.test.fully_qualified_name
}
`, testProviderBlock(), fqn)
}
