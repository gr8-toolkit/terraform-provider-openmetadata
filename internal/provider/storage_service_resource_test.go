// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccStorageServiceResource exercises Create/Update/Import for openmetadata_storage_service.
func TestAccStorageServiceResource(t *testing.T) {
	name := testRandName("storagvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageServiceConfig(name, "Initial description", "S3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_storage_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_storage_service.test", "service_type", "S3"),
					resource.TestCheckResourceAttr("openmetadata_storage_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_storage_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_storage_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccStorageServiceConfig(name, "Updated description", "S3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_storage_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_storage_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_storage_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccStorageServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_storage_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, storageServiceConnectionJSON(serviceType))
}

func storageServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	// S3 config uses awsConfig directly — the "credentials" wrapper is not
	// a recognised field in OM's S3 connection schema.
	return `{"config":{"type":"S3","awsConfig":{"awsRegion":"us-east-1","awsSecretAccessKey":"test","awsAccessKeyId":"test"}}}`
}
