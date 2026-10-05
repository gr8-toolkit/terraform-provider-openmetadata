// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPipelineServiceDataSource(t *testing.T) {
	name := testRandName("pipelinesvc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPipelineServiceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_pipeline_service.test", "id",
						"openmetadata_pipeline_service.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_pipeline_service.test", "service_type",
						"openmetadata_pipeline_service.test", "service_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_pipeline_service.test", "fully_qualified_name",
						"openmetadata_pipeline_service.test", "fully_qualified_name"),
					resource.TestCheckNoResourceAttr("data.openmetadata_pipeline_service.test", "connection_json"),
					resource.TestCheckResourceAttrSet("data.openmetadata_pipeline_service.test", "id"),
				),
			},
		},
	})
}

func TestAccPipelineServiceDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPipelineServiceDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile("Pipeline service not found"),
			},
		},
	})
}

func testAccPipelineServiceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_pipeline_service" "test" {
  name            = %q
  description     = "Pipeline service data source acceptance test"
  service_type    = "Airflow"
  connection_json = %q
}

data "openmetadata_pipeline_service" "test" {
  name = openmetadata_pipeline_service.test.name
}
`, testProviderBlock(), name, pipelineServiceConnectionJSON("Airflow"))
}

func testAccPipelineServiceDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_pipeline_service" "test" {
  name = "tfacc_pipeline_svc_does_not_exist_99999"
}
`, testProviderBlock())
}
