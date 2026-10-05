// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDriveServiceDataSource(t *testing.T) {
	name := testRandName("drivesvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDriveServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_drive_service.test", "id",
						"openmetadata_drive_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_drive_service.test", "service_type",
						"openmetadata_drive_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_drive_service.test", "fully_qualified_name",
						"openmetadata_drive_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_drive_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_drive_service.test", "id"),
				),
			},
		},
	})
}

func TestAccDriveServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDriveServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Drive service not found"),
			},
		},
	})
}

func testAccDriveServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_drive_service" "test" {
  name            = %q
  description     = "Drive service data source acceptance test"
  service_type    = "GoogleDrive"
  connection_json = %q
}

data "openmetadata_drive_service" "test" {
  name = openmetadata_drive_service.test.name
}
`, testProviderBlock(), name, driveServiceConnectionJSON("GoogleDrive"))
}

func testAccDriveServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_drive_service" "test" {
  name = "tfacc_drive_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
