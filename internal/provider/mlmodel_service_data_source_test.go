// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMLModelServiceDataSource(t *testing.T) {
	name := testRandName("mlsvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMLModelServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_mlmodel_service.test", "id",
						"openmetadata_mlmodel_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_mlmodel_service.test", "service_type",
						"openmetadata_mlmodel_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_mlmodel_service.test", "fully_qualified_name",
						"openmetadata_mlmodel_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_mlmodel_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_mlmodel_service.test", "id"),
				),
			},
		},
	})
}

func TestAccMLModelServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccMLModelServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("ML Model service not found"),
			},
		},
	})
}

func testAccMLModelServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_mlmodel_service" "test" {
  name            = %q
  description     = "MLModel service data source acceptance test"
  service_type    = "Mlflow"
  connection_json = %q
}

data "openmetadata_mlmodel_service" "test" {
  name = openmetadata_mlmodel_service.test.name
}
`, testProviderBlock(), name, mlmodelServiceConnectionJSON("Mlflow"))
}

func testAccMLModelServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_mlmodel_service" "test" {
  name = "tfacc_mlmodel_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
