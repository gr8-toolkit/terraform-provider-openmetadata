// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccMLModelServiceResource exercises Create/Update/Import for openmetadata_mlmodel_service.
func TestAccMLModelServiceResource(t *testing.T) {
	name := testRandName("mlsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMLModelServiceConfig(name, "Initial description", "Mlflow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_mlmodel_service.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_mlmodel_service.test", "service_type", "Mlflow"),
					resource.TestCheckResourceAttr("openmetadata_mlmodel_service.test", "description", "Initial description"),
					resource.TestCheckResourceAttrSet("openmetadata_mlmodel_service.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_mlmodel_service.test", "fully_qualified_name"),
				),
			},
			{
				Config: testAccMLModelServiceConfig(name, "Updated description", "Mlflow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_mlmodel_service.test", "description", "Updated description"),
				),
			},
			{
				ResourceName:            "openmetadata_mlmodel_service.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connection_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_mlmodel_service.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccMLModelServiceConfig(name, description, serviceType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_mlmodel_service" "test" {
  name            = %q
  description     = %q
  service_type    = %q
  connection_json = %q
}
`, testProviderBlock(), name, description, serviceType, mlmodelServiceConnectionJSON(serviceType))
}

func mlmodelServiceConnectionJSON(serviceType string) string {
	_ = serviceType
	return `{"config":{"type":"Mlflow","trackingUri":"http://localhost:5000","registryUri":"http://localhost:5001"}}`
}
