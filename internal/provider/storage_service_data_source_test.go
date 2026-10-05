// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStorageServiceDataSource(t *testing.T) {
	name := testRandName("storagvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_storage_service.test", "id",
						"openmetadata_storage_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_storage_service.test", "service_type",
						"openmetadata_storage_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_storage_service.test", "fully_qualified_name",
						"openmetadata_storage_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_storage_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_storage_service.test", "id"),
				),
			},
		},
	})
}

func TestAccStorageServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccStorageServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Storage service not found"),
			},
		},
	})
}

func testAccStorageServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_storage_service" "test" {
  name            = %q
  description     = "Storage service data source acceptance test"
  service_type    = "S3"
  connection_json = %q
}

data "openmetadata_storage_service" "test" {
  name = openmetadata_storage_service.test.name
}
`, testProviderBlock(), name, storageServiceConnectionJSON("S3"))
}

func testAccStorageServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_storage_service" "test" {
  name = "tfacc_storage_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
