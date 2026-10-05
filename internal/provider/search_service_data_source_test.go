// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSearchServiceDataSource(t *testing.T) {
	name := testRandName("searchvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSearchServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_search_service.test", "id",
						"openmetadata_search_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_search_service.test", "service_type",
						"openmetadata_search_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_search_service.test", "fully_qualified_name",
						"openmetadata_search_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_search_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_search_service.test", "id"),
				),
			},
		},
	})
}

func TestAccSearchServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccSearchServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Search service not found"),
			},
		},
	})
}

func testAccSearchServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_search_service" "test" {
  name            = %q
  description     = "Search service data source acceptance test"
  service_type    = "ElasticSearch"
  connection_json = %q
}

data "openmetadata_search_service" "test" {
  name = openmetadata_search_service.test.name
}
`, testProviderBlock(), name, searchServiceConnectionJSON("ElasticSearch"))
}

func testAccSearchServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_search_service" "test" {
  name = "tfacc_search_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
