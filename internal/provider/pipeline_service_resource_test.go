// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccPipelineServiceResource exercises Create/Update/Import for openmetadata_pipeline_service.
func TestAccPipelineServiceResource(t *testing.T) {
	name := testRandName("pipelinesvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineServiceConfig(name, "Initial description", "Airflow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_pipeline_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_pipeline_service.test", "service_type", "Airflow"),
					resource.TestCheckResourceAttr("openmetadata_pipeline_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_pipeline_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_pipeline_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccPipelineServiceConfig(name, "Updated description", "Airflow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_pipeline_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_pipeline_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_pipeline_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccPipelineServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_pipeline_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, pipelineServiceConnectionJSON(serviceType))
}

func pipelineServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	// Omitting the nested connection field — OM's BackendConnection Jackson
	// deserializer requires a specific class discriminator that varies by version.
	// hostPort alone is sufficient for OM to accept the service registration.
	return `{"config":{"type":"Airflow","hostPort":"http://localhost:8080","numberOfStatus":10}}`
}
