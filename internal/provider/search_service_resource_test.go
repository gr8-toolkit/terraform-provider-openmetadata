// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccSearchServiceResource exercises Create/Update/Import for openmetadata_search_service.
func TestAccSearchServiceResource(t *testing.T) {
	name := testRandName("searchvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSearchServiceConfig(name, "Initial description", "ElasticSearch"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_search_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_search_service.test", "service_type", "ElasticSearch"),
					resource.TestCheckResourceAttr("openmetadata_search_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_search_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_search_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccSearchServiceConfig(name, "Updated description", "ElasticSearch"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_search_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_search_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_search_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccSearchServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_search_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, searchServiceConnectionJSON(serviceType))
}

func searchServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"ElasticSearch","hostPort":"localhost:9200"}}`
}
