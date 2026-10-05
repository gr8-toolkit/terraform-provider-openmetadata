// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccDriveServiceResource exercises Create/Update/Import for openmetadata_drive_service.
func TestAccDriveServiceResource(t *testing.T) {
	name := testRandName("drivesvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDriveServiceConfig(name, "Initial description", "GoogleDrive"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_drive_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_drive_service.test", "service_type", "GoogleDrive"),
					resource.TestCheckResourceAttr("openmetadata_drive_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_drive_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_drive_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccDriveServiceConfig(name, "Updated description", "GoogleDrive"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_drive_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_drive_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_drive_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccDriveServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_drive_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, driveServiceConnectionJSON(serviceType))
}

func driveServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"GoogleDrive","credentials":{"gcpConfig":{"type":"gcp_adc","projectId":"test-project"}}}}`
}
