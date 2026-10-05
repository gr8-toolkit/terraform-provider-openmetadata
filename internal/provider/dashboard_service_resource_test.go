// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccDashboardServiceResource exercises Create/Update/Import for openmetadata_dashboard_service.
func TestAccDashboardServiceResource(t *testing.T) {
	name := testRandName("dashsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardServiceConfig(name, "Initial description", "Superset"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_dashboard_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_dashboard_service.test", "service_type", "Superset"),
					resource.TestCheckResourceAttr("openmetadata_dashboard_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_dashboard_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_dashboard_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccDashboardServiceConfig(name, "Updated description", "Superset"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_dashboard_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_dashboard_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_dashboard_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccDashboardServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_dashboard_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, dashboardServiceConnectionJSON(serviceType))
}

func dashboardServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"Superset","hostPort":"http://localhost:8088","connection":{"provider":"db","username":"admin","password":"test"}}}`
}
