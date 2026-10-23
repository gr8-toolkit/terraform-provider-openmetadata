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

// These tests run the real provider through a full Terraform lifecycle against
// the in-memory fakeOM server, so they need no live OpenMetadata instance.
// They are skipped when no terraform CLI is available.

const (
	fakeDBCollection = "services/databaseServices"
	fakeResAddr      = "openmetadata_ingestion_pipeline.test"
	fakeDSAddr       = "data.openmetadata_ingestion_pipeline.test"
)

// fakeIngestionConfig renders a pipeline resource; extra holds additional
// attribute lines placed inside the resource block.
func fakeIngestionConfig(f *fakeOM, svcName, name, pipelineType, extra string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = %q
  pipeline_type      = %q
  service            = { type = "databaseService", name = %q }
  source_config_json = jsonencode({ config = { type = "DatabaseMetadata" } })
%s
}
`, f.providerBlock(), name, pipelineType, svcName, extra)
}

func checkPutBody(f *fakeOM, check func(body map[string]interface{}) error) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		f.mu.Lock()
		if len(f.puts) == 0 {
			f.mu.Unlock()
			return fmt.Errorf("no pipeline PUT request was received")
		}
		body := f.puts[len(f.puts)-1]
		f.mu.Unlock()
		return check(body)
	}
}

func TestFakeIngestionPipelineLifecycle(t *testing.T) {
	f := newFakeOM(t)
	svcName, name := "fakesvc", "fakepipe"
	svcID := f.seedService(fakeDBCollection, "databaseService", svcName)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if n := f.pipelineCount(); n != 0 {
				return fmt.Errorf("%d pipeline(s) left behind after destroy", n)
			}
			return nil
		},
		Steps: []resource.TestStep{
			// ── Create: unset fields must come back as the server defaults ────
			{
				Config: fakeIngestionConfig(f, svcName, name, "metadata", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeResAddr, "name", name),
					resource.TestCheckResourceAttr(fakeResAddr, "pipeline_type", "metadata"),
					resource.TestCheckResourceAttr(fakeResAddr, "fully_qualified_name", svcName+"."+name),
					resource.TestCheckResourceAttr(fakeResAddr, "service.type", "databaseService"),
					resource.TestCheckResourceAttr(fakeResAddr, "service.name", svcName),
					resource.TestCheckResourceAttrSet(fakeResAddr, "id"),
					resource.TestCheckResourceAttr(fakeResAddr, "logger_level", "INFO"),
					resource.TestCheckResourceAttr(fakeResAddr, "raise_on_error", "true"),
					resource.TestCheckResourceAttr(fakeResAddr, "enable_streamable_logs", "false"),
					resource.TestCheckResourceAttr(fakeResAddr, "enabled", "true"),
					resource.TestCheckResourceAttr(fakeResAddr, "deployed", "false"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.retries", "0"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.retry_delay", "300"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.concurrency", "1"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.max_active_runs", "1"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.pipeline_timezone", "UTC"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.pause_pipeline", "false"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.pipeline_catchup", "false"),
					resource.TestCheckNoResourceAttr(fakeResAddr, "airflow_config.schedule_interval"),
					resource.TestCheckResourceAttr(fakeResAddr, "source_config_json", `{"config":{"type":"DatabaseMetadata"}}`),
					// The request carried the resolved service UUID and the parsed sourceConfig.
					checkPutBody(f, func(b map[string]interface{}) error {
						svc, _ := b["service"].(map[string]interface{})
						if svc["id"] != svcID || svc["type"] != "databaseService" {
							return fmt.Errorf("service ref = %v, want id %s type databaseService", svc, svcID)
						}
						sc, _ := b["sourceConfig"].(map[string]interface{})
						cfg, _ := sc["config"].(map[string]interface{})
						if cfg["type"] != "DatabaseMetadata" {
							return fmt.Errorf("sourceConfig = %v", sc)
						}
						if af, ok := b["airflowConfig"].(map[string]interface{}); !ok || len(af) != 0 {
							return fmt.Errorf("airflowConfig = %v, want empty object so the server applies defaults", b["airflowConfig"])
						}
						for _, k := range []string{"description", "displayName", "loggerLevel", "owners", "domains"} {
							if _, present := b[k]; present {
								return fmt.Errorf("unset attribute %q must not be sent", k)
							}
						}
						return nil
					}),
				),
			},
			// ── Update in place ───────────────────────────────────────────────
			{
				Config: fakeIngestionConfig(f, svcName, name, "metadata", `
  description  = "Updated"
  display_name = "Fake pipe"
  logger_level = "DEBUG"
  airflow_config = {
    schedule_interval = "0 4 * * *"
    retries           = 3
  }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeResAddr, "description", "Updated"),
					resource.TestCheckResourceAttr(fakeResAddr, "display_name", "Fake pipe"),
					resource.TestCheckResourceAttr(fakeResAddr, "logger_level", "DEBUG"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.schedule_interval", "0 4 * * *"),
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.retries", "3"),
					// Unspecified nested fields keep the server defaults.
					resource.TestCheckResourceAttr(fakeResAddr, "airflow_config.retry_delay", "300"),
					checkPutBody(f, func(b map[string]interface{}) error {
						af, _ := b["airflowConfig"].(map[string]interface{})
						if af["scheduleInterval"] != "0 4 * * *" || af["retries"] != float64(3) {
							return fmt.Errorf("airflowConfig = %v", af)
						}
						if b["loggerLevel"] != "DEBUG" || b["description"] != "Updated" {
							return fmt.Errorf("body = %v", b)
						}
						return nil
					}),
				),
			},
			// ── Re-applying the same config is a no-op ────────────────────────
			{
				Config: fakeIngestionConfig(f, svcName, name, "metadata", `
  description  = "Updated"
  display_name = "Fake pipe"
  logger_level = "DEBUG"
  airflow_config = {
    schedule_interval = "0 4 * * *"
    retries           = 3
  }`),
				PlanOnly: true,
			},
			// ── Import by FQN ─────────────────────────────────────────────────
			{
				ResourceName:            fakeResAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"source_config_json"},
				ImportStateId:           svcName + "." + name,
			},
		},
	})
}

// TestFakeIngestionPipelineSourceConfigFormatting verifies that a source config
// written with different whitespace and key order than the server returns does
// not cause a perpetual diff or an apply error.
func TestFakeIngestionPipelineSourceConfigFormatting(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")

	cfg := fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name          = "fakepipe"
  pipeline_type = "metadata"
  service       = { type = "databaseService", name = "fakesvc" }
  source_config_json = <<-EOT
    {
      "config": {
        "schemaFilterPattern": { "includes": ["public"] },
        "type": "DatabaseMetadata"
      }
    }
  EOT
}
`, f.providerBlock())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true},
		},
	})
}

func TestFakeIngestionPipelineOwnersAndDomains(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")
	const ownerID = "11111111-2222-4333-8444-555555555555"

	cfg := fakeIngestionConfig(f, "fakesvc", "fakepipe", "metadata", fmt.Sprintf(`
  owners  = [{ id = %q, type = "user" }]
  domains = ["Marketing"]`, ownerID))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeResAddr, "owners.#", "1"),
					resource.TestCheckResourceAttr(fakeResAddr, "owners.0.id", ownerID),
					resource.TestCheckResourceAttr(fakeResAddr, "owners.0.type", "user"),
					resource.TestCheckResourceAttr(fakeResAddr, "domains.#", "1"),
					resource.TestCheckResourceAttr(fakeResAddr, "domains.0", "Marketing"),
					checkPutBody(f, func(b map[string]interface{}) error {
						owners, _ := b["owners"].([]interface{})
						if len(owners) != 1 {
							return fmt.Errorf("owners = %v", b["owners"])
						}
						o, _ := owners[0].(map[string]interface{})
						if o["id"] != ownerID || o["type"] != "user" {
							return fmt.Errorf("owner = %v", o)
						}
						domains, _ := b["domains"].([]interface{})
						if len(domains) != 1 || domains[0] != "Marketing" {
							return fmt.Errorf("domains = %v", b["domains"])
						}
						return nil
					}),
				),
			},
			// Owners are preserved from config, not read back: no diff on refresh.
			{Config: cfg, PlanOnly: true},
		},
	})
}

// TestFakeIngestionPipelineServiceTypes verifies every supported service type
// is resolved through the right API collection.
func TestFakeIngestionPipelineServiceTypes(t *testing.T) {
	cases := []struct {
		serviceType  string
		collection   string
		pipelineType string
	}{
		{"databaseService", "services/databaseServices", "metadata"},
		{"messagingService", "services/messagingServices", "metadata"},
		{"dashboardService", "services/dashboardServices", "metadata"},
		{"pipelineService", "services/pipelineServices", "metadata"},
		{"mlmodelService", "services/mlmodelServices", "metadata"},
		{"storageService", "services/storageServices", "metadata"},
		{"searchService", "services/searchServices", "metadata"},
		{"metadataService", "services/metadataServices", "metadata"},
		{"apiService", "services/apiServices", "metadata"},
		{"driveService", "services/driveServices", "metadata"},
		{"testSuite", "dataQuality/testSuites", "TestSuite"},
	}
	for _, tc := range cases {
		t.Run(tc.serviceType, func(t *testing.T) {
			f := newFakeOM(t)
			id := f.seedService(tc.collection, tc.serviceType, "fakesvc")
			cfg := fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = "fakepipe"
  pipeline_type      = %q
  service            = { type = %q, name = "fakesvc" }
  source_config_json = jsonencode({ config = { type = "Any" } })
}
`, f.providerBlock(), tc.pipelineType, tc.serviceType)

			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: cfg,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(fakeResAddr, "service.type", tc.serviceType),
							resource.TestCheckResourceAttr(fakeResAddr, "pipeline_type", tc.pipelineType),
							checkPutBody(f, func(b map[string]interface{}) error {
								svc, _ := b["service"].(map[string]interface{})
								if svc["id"] != id || svc["type"] != tc.serviceType {
									return fmt.Errorf("service ref = %v, want id %s type %s", svc, id, tc.serviceType)
								}
								return nil
							}),
						),
					},
				},
			})
		})
	}
}

// TestFakeIngestionPipelineReplace verifies that changing an immutable
// attribute destroys and recreates the pipeline rather than updating it.
func TestFakeIngestionPipelineReplace(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")
	f.seedService(fakeDBCollection, "databaseService", "othersvc")

	var firstID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fakeIngestionConfig(f, "fakesvc", "fakepipe", "metadata", ""),
				Check: func(s *terraform.State) error {
					firstID = s.RootModule().Resources[fakeResAddr].Primary.Attributes["id"]
					return nil
				},
			},
			{ // pipeline_type change
				Config: fakeIngestionConfig(f, "fakesvc", "fakepipe", "usage", ""),
				Check: func(s *terraform.State) error {
					id := s.RootModule().Resources[fakeResAddr].Primary.Attributes["id"]
					if id == firstID {
						return fmt.Errorf("pipeline_type change should replace the pipeline, id %s unchanged", id)
					}
					if n := f.deleteCount(); n != 1 {
						return fmt.Errorf("delete count = %d, want 1", n)
					}
					return nil
				},
			},
			{ // service change
				Config: fakeIngestionConfig(f, "othersvc", "fakepipe", "usage", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeResAddr, "service.name", "othersvc"),
					resource.TestCheckResourceAttr(fakeResAddr, "fully_qualified_name", "othersvc.fakepipe"),
					func(_ *terraform.State) error {
						if n := f.deleteCount(); n != 2 {
							return fmt.Errorf("delete count = %d, want 2", n)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestFakeIngestionPipelineDeletedOutOfBand verifies a pipeline removed on the
// server is planned for re-creation instead of failing the refresh.
func TestFakeIngestionPipelineDeletedOutOfBand(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")
	cfg := fakeIngestionConfig(f, "fakesvc", "fakepipe", "metadata", "")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig: func() {
					f.mu.Lock()
					delete(f.pipelines, "fakesvc.fakepipe")
					f.mu.Unlock()
				},
				Config: cfg,
				Check: func(_ *terraform.State) error {
					if n := f.putCount(); n != 2 {
						return fmt.Errorf("PUT count = %d, want 2 (create + re-create)", n)
					}
					return nil
				},
			},
		},
	})
}

func TestFakeIngestionPipelineErrors(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fakeIngestionConfig(f, "fakesvc", "fakepipe", "bogus", ""),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				Config: fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = "fakepipe"
  pipeline_type      = "metadata"
  service            = { type = "bogusService", name = "fakesvc" }
  source_config_json = "{}"
}
`, f.providerBlock()),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				Config:      fakeIngestionConfig(f, "missingsvc", "fakepipe", "metadata", ""),
				ExpectError: regexp.MustCompile(`databaseService "missingsvc" not found`),
			},
			{
				Config: fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = "fakepipe"
  pipeline_type      = "metadata"
  service            = { type = "databaseService", name = "fakesvc" }
  source_config_json = "[1, 2]"
}
`, f.providerBlock()),
				ExpectError: regexp.MustCompile(`invalid source_config_json`),
			},
			{
				Config: fmt.Sprintf(`
%s

resource "openmetadata_ingestion_pipeline" "test" {
  name               = "fakepipe"
  pipeline_type      = "metadata"
  service            = { type = "databaseService", name = "fakesvc" }
  source_config_json = "not json"
}
`, f.providerBlock()),
				ExpectError: regexp.MustCompile(`invalid source_config_json`),
			},
			{
				Config:      fakeIngestionConfig(f, "fakesvc", "fakepipe", "metadata", `logger_level = "TRACE"`),
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				ResourceName:  fakeResAddr,
				ImportState:   true,
				ImportStateId: "fakesvc.does_not_exist",
				Config:        fakeIngestionConfig(f, "fakesvc", "fakepipe", "metadata", ""),
				ExpectError:   regexp.MustCompile(`Ingestion pipeline not found`),
			},
		},
	})

	// None of the invalid configurations may have reached the pipeline endpoint.
	if n := f.putCount(); n != 0 {
		t.Errorf("PUT count = %d, want 0: invalid input must be rejected before any write", n)
	}
}

func TestFakeIngestionPipelineDataSource(t *testing.T) {
	f := newFakeOM(t)
	f.seedService(fakeDBCollection, "databaseService", "fakesvc")
	const ownerID = "11111111-2222-4333-8444-555555555555"

	cfg := fakeIngestionConfig(f, "fakesvc", "fakepipe", "usage", fmt.Sprintf(`
  description    = "For the data source"
  airflow_config = { schedule_interval = "0 5 * * *" }
  owners         = [{ id = %q, type = "team" }]
  domains        = ["Finance"]`, ownerID)) + `
data "openmetadata_ingestion_pipeline" "test" {
  name = openmetadata_ingestion_pipeline.test.fully_qualified_name
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(fakeDSAddr, "id", fakeResAddr, "id"),
					resource.TestCheckResourceAttr(fakeDSAddr, "fully_qualified_name", "fakesvc.fakepipe"),
					resource.TestCheckResourceAttr(fakeDSAddr, "description", "For the data source"),
					resource.TestCheckResourceAttr(fakeDSAddr, "pipeline_type", "usage"),
					resource.TestCheckResourceAttr(fakeDSAddr, "service.type", "databaseService"),
					resource.TestCheckResourceAttr(fakeDSAddr, "service.name", "fakesvc"),
					resource.TestCheckResourceAttr(fakeDSAddr, "airflow_config.schedule_interval", "0 5 * * *"),
					resource.TestCheckResourceAttr(fakeDSAddr, "airflow_config.retry_delay", "300"),
					resource.TestCheckResourceAttr(fakeDSAddr, "logger_level", "INFO"),
					resource.TestCheckResourceAttr(fakeDSAddr, "raise_on_error", "true"),
					resource.TestCheckResourceAttr(fakeDSAddr, "enabled", "true"),
					resource.TestCheckResourceAttr(fakeDSAddr, "deployed", "false"),
					resource.TestCheckResourceAttr(fakeDSAddr, "domains.0", "Finance"),
					// The data source, unlike the resource, reads owners from the API.
					resource.TestCheckResourceAttr(fakeDSAddr, "owners.0.id", ownerID),
					resource.TestCheckResourceAttr(fakeDSAddr, "owners.0.type", "team"),
					resource.TestCheckNoResourceAttr(fakeDSAddr, "source_config_json"),
				),
			},
		},
	})
}

func TestFakeIngestionPipelineDataSourceNotFound(t *testing.T) {
	f := newFakeOM(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: f.providerBlock() + `
data "openmetadata_ingestion_pipeline" "test" {
  name = "nosvc.nopipe"
}
`,
				ExpectError: regexp.MustCompile(`Ingestion pipeline not found`),
			},
		},
	})
}
