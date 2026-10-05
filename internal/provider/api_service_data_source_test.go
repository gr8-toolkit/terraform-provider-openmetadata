// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiServiceDataSource(t *testing.T) {
	name := testRandName("api")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccApiServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_api_service.test", "id",
						"openmetadata_api_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_api_service.test", "service_type",
						"openmetadata_api_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_api_service.test", "fully_qualified_name",
						"openmetadata_api_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_api_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_api_service.test", "id"),
				),
			},
		},
	})
}

func TestAccApiServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccApiServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("API service not found"),
			},
		},
	})
}

func testAccApiServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_api_service" "test" {
  name         = %q
  description  = "API service data source acceptance test"
  service_type = "Rest"
}

data "openmetadata_api_service" "test" {
  name = openmetadata_api_service.test.name
}
`, testProviderBlock(), name)
}

func testAccApiServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_api_service" "test" {
  name = "tfacc_api_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
