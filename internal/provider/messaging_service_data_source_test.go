// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMessagingServiceDataSource(t *testing.T) {
	name := testRandName("msgsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessagingServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_messaging_service.test", "id",
						"openmetadata_messaging_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_messaging_service.test", "service_type",
						"openmetadata_messaging_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_messaging_service.test", "fully_qualified_name",
						"openmetadata_messaging_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_messaging_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_messaging_service.test", "id"),
				),
			},
		},
	})
}

func TestAccMessagingServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMessagingServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Messaging service not found"),
			},
		},
	})
}

func testAccMessagingServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_messaging_service" "test" {
  name            = %q
  description     = "Messaging service data source acceptance test"
  service_type    = "Kafka"
  connection_json = %q
}

data "openmetadata_messaging_service" "test" {
  name = openmetadata_messaging_service.test.name
}
`, testProviderBlock(), name, messagingServiceConnectionJSON("Kafka"))
}

func testAccMessagingServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_messaging_service" "test" {
  name = "tfacc_messaging_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
