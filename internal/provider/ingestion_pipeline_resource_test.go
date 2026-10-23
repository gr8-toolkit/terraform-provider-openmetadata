// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccIngestionPipelineResource exercises Create/Update/Import for
// openmetadata_ingestion_pipeline.
//
// source_config_json is excluded from import verification by design: OM stores
// the config with server-side defaults, so the provider never reads it back.
func TestAccIngestionPipelineResource(t *testing.T) {
	svcName := testRandName("dbsvc")
	name := testRandName("ingpl")
	const addr = "openmetadata_ingestion_pipeline.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create + Read ────────────────────────────────────────────────
			{
				Config: testAccIngestionPipelineConfig(svcName, name, "Initial description", "metadata",
					`{"config":{"type":"DatabaseMetadata"}}`, "0 * * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", name),
					resource.TestCheckResourceAttr(addr, "description", "Initial description"),
					resource.TestCheckResourceAttr(addr, "pipeline_type", "metadata"),
					resource.TestCheckResourceAttr(addr, "service.type", "databaseService"),
					resource.TestCheckResourceAttr(addr, "service.name", svcName),
					resource.TestCheckResourceAttr(addr, "airflow_config.schedule_interval", "0 * * * *"),
					resource.TestCheckResourceAttrSet(addr, "airflow_config.retry_delay"),
					resource.TestCheckResourceAttrSet(addr, "logger_level"),
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "fully_qualified_name", svcName+"."+name),
					resource.TestCheckResourceAttrSet(addr, "enabled"),
					resource.TestCheckResourceAttrSet(addr, "deployed"),
				),
			},
			// ── Update ───────────────────────────────────────────────────────
			{
				Config: testAccIngestionPipelineConfig(svcName, name, "Updated description", "metadata",
					`{"config":{"type":"DatabaseMetadata"}}`, "30 2 * * *"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "description", "Updated description"),
					resource.TestCheckResourceAttr(addr, "airflow_config.schedule_interval", "30 2 * * *"),
				),
			},
			// ── Import by FQN (<service>.<pipeline>) ─────────────────────────
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_config_json"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[addr]
					return rs.Primary.Attributes["fully_qualified_name"], nil
				},
			},
		},
	})
}

// TestAccIngestionPipelineResourceTypes verifies the workflow types that attach
// to a database service are accepted.
func TestAccIngestionPipelineResourceTypes(t *testing.T) {
	cases := []struct {
		pipelineType string
		sourceConfig string
	}{
		{"metadata", `{"config":{"type":"DatabaseMetadata"}}`},
		{"usage", `{"config":{"type":"DatabaseUsage"}}`},
		{"lineage", `{"config":{"type":"DatabaseLineage"}}`},
		{"profiler", `{"config":{"type":"Profiler"}}`},
		{"autoClassification", `{"config":{"type":"AutoClassification"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.pipelineType, func(t *testing.T) {
			svcName := testRandName("dbsvc")
			name := testRandName("ingpl")
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccIngestionPipelineConfig(svcName, name, "Pipeline type test", tc.pipelineType, tc.sourceConfig, ""),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("openmetadata_ingestion_pipeline.test", "pipeline_type", tc.pipelineType),
							resource.TestCheckResourceAttrSet("openmetadata_ingestion_pipeline.test", "id"),
						),
					},
				},
			})
		})
	}
}

// TestAccIngestionPipelineResourceOptions covers the optional runtime settings.
func TestAccIngestionPipelineResourceOptions(t *testing.T) {
	svcName := testRandName("dbsvc")
	name := testRandName("ingpl")
	const addr = "openmetadata_ingestion_pipeline.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccIngestionPipelineOptionsConfig(svcName, name, "DEBUG", 2, 120),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "display_name", "Options pipeline"),
					resource.TestCheckResourceAttr(addr, "logger_level", "DEBUG"),
					resource.TestCheckResourceAttr(addr, "raise_on_error", "false"),
					resource.TestCheckResourceAttr(addr, "airflow_config.retries", "2"),
					resource.TestCheckResourceAttr(addr, "airflow_config.retry_delay", "120"),
					resource.TestCheckResourceAttr(addr, "airflow_config.pipeline_timezone", "Europe/Kyiv"),
				),
			},
			{
				Config: testAccIngestionPipelineOptionsConfig(svcName, name, "WARN", 3, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "logger_level", "WARN"),
					resource.TestCheckResourceAttr(addr, "airflow_config.retries", "3"),
					resource.TestCheckResourceAttr(addr, "airflow_config.retry_delay", "60"),
				),
			},
		},
	})
}

// TestAccIngestionPipelineResourceInvalidAttributes checks plan-time validation
// and the apply-time error for a service that does not exist.
func TestAccIngestionPipelineResourceInvalidAttributes(t *testing.T) {
	cases := map[string]struct {
		pipelineType string
		serviceType  string
		serviceName  string
		sourceConfig string
		expect       string
	}{
		"pipeline type": {"bogus", "databaseService", "svc", `{"config":{}}`, `Invalid Attribute Value Match`},
		"service type":  {"metadata", "bogusService", "svc", `{"config":{}}`, `Invalid Attribute Value Match`},
		"service name":  {"metadata", "databaseService", "tfacc_missing_service_99999", `{"config":{}}`, `not found`},
		"source config": {"metadata", "databaseService", "svc", `["not","an","object"]`, `invalid source_config_json`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = %q
  pipeline_type      = %q
  service            = { type = %q, name = %q }
  source_config_json = %q
}
`, testProviderBlock(), testRandName("ingpl"), tc.pipelineType, tc.serviceType, tc.serviceName, tc.sourceConfig),
						ExpectError: regexp.MustCompile(tc.expect),
					},
				},
			})
		})
	}
}

func testAccIngestionPipelineConfig(svcName, name, description, pipelineType, sourceConfig, schedule string) string {
	airflow := ""
	if schedule != "" {
		airflow = fmt.Sprintf("airflow_config = { schedule_interval = %q }", schedule)
	}
	return fmt.Sprintf(`
%s

resource "openmetadata_database_service" "svc" {
  name            = %q
  description     = "Service for the ingestion pipeline acceptance test"
  service_type    = "Mysql"
  connection_json = %q
}

resource "openmetadata_ingestion_pipeline" "test" {
  name               = %q
  description        = %q
  pipeline_type      = %q
  service            = { type = "databaseService", name = openmetadata_database_service.svc.name }
  source_config_json = %q
  %s
}
`, testProviderBlock(), svcName, dbServiceConnectionJSON("Mysql"), name, description, pipelineType, sourceConfig, airflow)
}

func testAccIngestionPipelineOptionsConfig(svcName, name, loggerLevel string, retries, retryDelay int) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_database_service" "svc" {
  name            = %q
  description     = "Service for the ingestion pipeline acceptance test"
  service_type    = "Mysql"
  connection_json = %q
}

resource "openmetadata_ingestion_pipeline" "test" {
  name               = %q
  display_name       = "Options pipeline"
  pipeline_type      = "metadata"
  service            = { type = "databaseService", name = openmetadata_database_service.svc.name }
  source_config_json = jsonencode({ config = { type = "DatabaseMetadata" } })
  logger_level       = %q
  raise_on_error     = false

  airflow_config = {
    schedule_interval = "0 3 * * *"
    pipeline_timezone = "Europe/Kyiv"
    retries           = %d
    retry_delay       = %d
  }
}
`, testProviderBlock(), svcName, dbServiceConnectionJSON("Mysql"), name, loggerLevel, retries, retryDelay)
}
