// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseServiceDataSource(t *testing.T) {
	name := testRandName("dbsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatabaseServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_database_service.test", "id",
						"openmetadata_database_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_database_service.test", "name",
						"openmetadata_database_service.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_database_service.test", "description",
						"openmetadata_database_service.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_database_service.test", "service_type",
						"openmetadata_database_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_database_service.test", "fully_qualified_name",
						"openmetadata_database_service.test", "fully_qualified_name"),
					// connection_json is intentionally absent from the data source schema.
					resource.TestCheckNoResourceAttr(
						"data.openmetadata_database_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_database_service.test", "id"),
				),
			},
		},
	})
}

func TestAccDatabaseServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDatabaseServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Database service not found`),
			},
		},
	})
}

func testAccDatabaseServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_database_service" "test" {
  name            = %q
  description     = "Database service data source acceptance test"
  service_type    = "Mysql"
  connection_json = %q
}

data "openmetadata_database_service" "test" {
  name = openmetadata_database_service.test.name
}
`, testProviderBlock(), name, dbServiceConnectionJSON("Mysql"))
}

func testAccDatabaseServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_database_service" "test" {
  name = "tfacc_dbsvc_does_not_exist_99999"
}
`, testProviderBlock())
}
