// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMessagingServiceResource exercises Create/Update/Import for openmetadata_messaging_service.
func TestAccMessagingServiceResource(t *testing.T) {
	name := testRandName("msgsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMessagingServiceConfig(name, "Initial description", "Kafka"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_messaging_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_messaging_service.test", "service_type", "Kafka"),
					resource.TestCheckResourceAttr("openmetadata_messaging_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_messaging_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_messaging_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccMessagingServiceConfig(name, "Updated description", "Kafka"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_messaging_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_messaging_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_messaging_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccMessagingServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_messaging_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, messagingServiceConnectionJSON(serviceType))
}

func messagingServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"Kafka","bootstrapServers":"localhost:9092"}}`
}
