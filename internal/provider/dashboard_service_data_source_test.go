// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDashboardServiceDataSource(t *testing.T) {
	name := testRandName("dashsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDashboardServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_dashboard_service.test", "id",
						"openmetadata_dashboard_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_dashboard_service.test", "service_type",
						"openmetadata_dashboard_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_dashboard_service.test", "fully_qualified_name",
						"openmetadata_dashboard_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_dashboard_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_dashboard_service.test", "id"),
				),
			},
		},
	})
}

func TestAccDashboardServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDashboardServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Dashboard service not found"),
			},
		},
	})
}

func testAccDashboardServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_dashboard_service" "test" {
  name            = %q
  description     = "Dashboard service data source acceptance test"
  service_type    = "Superset"
  connection_json = %q
}

data "openmetadata_dashboard_service" "test" {
  name = openmetadata_dashboard_service.test.name
}
`, testProviderBlock(), name, dashboardServiceConnectionJSON("Superset"))
}

func testAccDashboardServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_dashboard_service" "test" {
  name = "tfacc_dashboard_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
