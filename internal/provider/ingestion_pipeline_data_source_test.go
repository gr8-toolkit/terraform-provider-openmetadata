// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccIngestionPipelineDataSource(t *testing.T) {
	svcName := testRandName("dbsvc")
	name := testRandName("ingpl")
	const ds = "data.openmetadata_ingestion_pipeline.test"
	const res = "openmetadata_ingestion_pipeline.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionPipelineDataSourceConfig(svcName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "fully_qualified_name", res, "fully_qualified_name"),
					resource.TestCheckResourceAttrPair(ds, "description", res, "description"),
					resource.TestCheckResourceAttrPair(ds, "pipeline_type", res, "pipeline_type"),
					resource.TestCheckResourceAttrPair(ds, "service.type", res, "service.type"),
					resource.TestCheckResourceAttrPair(ds, "service.name", res, "service.name"),
					resource.TestCheckResourceAttrPair(ds, "airflow_config.schedule_interval", res, "airflow_config.schedule_interval"),
					resource.TestCheckResourceAttrPair(ds, "logger_level", res, "logger_level"),
					resource.TestCheckResourceAttrPair(ds, "enabled", res, "enabled"),
					resource.TestCheckResourceAttrPair(ds, "deployed", res, "deployed"),
					// The sourceConfig is write-only and never exposed.
					resource.TestCheckNoResourceAttr(ds, "source_config_json"),
					resource.TestCheckResourceAttrSet(ds, "id"),
				),
			},
		},
	})
}

func TestAccIngestionPipelineDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "openmetadata_ingestion_pipeline" "test" {
  name = "tfacc_missing_service_99999.tfacc_ingpl_does_not_exist_99999"
}
`, testProviderBlock()),
				ExpectError: regexp.MustCompile("Ingestion pipeline not found"),
			},
		},
	})
}

func testAccIngestionPipelineDataSourceConfig(svcName, name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_database_service" "svc" {
  name            = %q
  description     = "Service for the ingestion pipeline data source test"
  service_type    = "Mysql"
  connection_json = %q
}

resource "openmetadata_ingestion_pipeline" "test" {
  name               = %q
  description        = "Ingestion pipeline data source acceptance test"
  pipeline_type      = "metadata"
  service            = { type = "databaseService", name = openmetadata_database_service.svc.name }
  source_config_json = jsonencode({ config = { type = "DatabaseMetadata" } })
  airflow_config     = { schedule_interval = "0 * * * *" }
}

# Pipelines are looked up by FQN (<service>.<pipeline>).
data "openmetadata_ingestion_pipeline" "test" {
  name = openmetadata_ingestion_pipeline.test.fully_qualified_name
}
`, testProviderBlock(), svcName, dbServiceConnectionJSON("Mysql"), name)
}
