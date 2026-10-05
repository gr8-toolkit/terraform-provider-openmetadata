// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMetadataServiceDataSource(t *testing.T) {
	name := testRandName("metasvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetadataServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_metadata_service.test", "id",
						"openmetadata_metadata_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_metadata_service.test", "service_type",
						"openmetadata_metadata_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_metadata_service.test", "fully_qualified_name",
						"openmetadata_metadata_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_metadata_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_metadata_service.test", "id"),
				),
			},
		},
	})
}

func TestAccMetadataServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMetadataServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Metadata service not found"),
			},
		},
	})
}

func testAccMetadataServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_metadata_service" "test" {
  name            = %q
  description     = "Metadata service data source acceptance test"
  service_type    = "Atlas"
  connection_json = %q
}

data "openmetadata_metadata_service" "test" {
  name = openmetadata_metadata_service.test.name
}
`, testProviderBlock(), name, metadataServiceConnectionJSON("Atlas"))
}

func testAccMetadataServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_metadata_service" "test" {
  name = "tfacc_metadata_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
