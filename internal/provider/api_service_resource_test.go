// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccApiServiceResource exercises Create/Update/Import for openmetadata_api_service.
func TestAccApiServiceResource(t *testing.T) {
	name := testRandName("api")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiServiceConfig(name, "Initial description", "Rest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_api_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_api_service.test", "service_type", "Rest"),
					resource.TestCheckResourceAttr("openmetadata_api_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_api_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_api_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccApiServiceConfig(name, "Updated description", "Rest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_api_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_api_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_api_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccApiServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_api_service" "test" {
  name         = %q
  description  = %q
  service_type = %q
}
`, testProviderBlock(), name, description, serviceType)
}
