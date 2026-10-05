// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMetadataServiceResource exercises Create/Update/Import for openmetadata_metadata_service.
func TestAccMetadataServiceResource(t *testing.T) {
	name := testRandName("metasvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMetadataServiceConfig(name, "Initial description", "Atlas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_metadata_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_metadata_service.test", "service_type", "Atlas"),
					resource.TestCheckResourceAttr("openmetadata_metadata_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_metadata_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_metadata_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccMetadataServiceConfig(name, "Updated description", "Atlas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_metadata_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_metadata_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_metadata_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccMetadataServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_metadata_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, metadataServiceConnectionJSON(serviceType))
}

func metadataServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"Atlas","hostPort":"http://localhost:21000","username":"admin","password":"admin"}}`
}
