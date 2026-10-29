// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccSeedTable stands in for an ingestion run: it creates a database
// service with one database, schema and table straight through the API (the
// provider cannot create assets) and removes the service again when the test
// ends. The table's fully qualified name is testAccSeededTableFQN(svc).
func testAccSeedTable(t *testing.T, svc string) {
	t.Helper()
	ctx := context.Background()
	c := client.NewClient(os.Getenv("OPENMETADATA_HOST"), os.Getenv("OPENMETADATA_TOKEN"))

	raw, err := c.CreateOrUpdate(ctx, "services/databaseServices", map[string]interface{}{
		"name":        svc,
		"serviceType": "Mysql",
		"connection": map[string]interface{}{
			"config": map[string]interface{}{
				"type": "Mysql", "scheme": "mysql+pymysql", "username": "test", "hostPort": "localhost:3306",
			},
		},
	})
	if err != nil {
		t.Fatalf("seeding database service: %v", err)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatalf("parsing database service: %v", err)
	}
	t.Cleanup(func() { _ = c.Delete(ctx, "services/databaseServices", created.ID, true) })

	steps := []struct {
		collection string
		body       map[string]interface{}
	}{
		{"databases", map[string]interface{}{"name": "db", "service": svc}},
		{"databaseSchemas", map[string]interface{}{"name": "public", "database": svc + ".db"}},
		{"tables", map[string]interface{}{
			"name":           "orders",
			"databaseSchema": svc + ".db.public",
			"columns":        []map[string]interface{}{{"name": "id", "dataType": "INT"}},
		}},
	}
	for _, s := range steps {
		if _, err := c.CreateOrUpdate(ctx, s.collection, s.body); err != nil {
			t.Fatalf("seeding %s: %v", s.collection, err)
		}
	}
}

// testAccSeededTableFQN is the table testAccSeedTable creates in service svc.
// It is a function of the name so configs can be built before PreCheck runs.
func testAccSeededTableFQN(svc string) string {
	return svc + ".db.public.orders"
}

// TestAccDataAssetResource adopts an existing table, manages its metadata,
// updates it, imports it, and checks that destroying leaves the table alone.
//
// extension_json is not covered: it needs a custom property defined on the
// table type beforehand.
func TestAccDataAssetResource(t *testing.T) {
	svc := testRandName("assvc")
	fqn := testAccSeededTableFQN(svc)
	suffix := testRandName("asset")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccSeedTable(t, svc)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			c := client.NewClient(os.Getenv("OPENMETADATA_HOST"), os.Getenv("OPENMETADATA_TOKEN"))
			raw, err := c.GetByName(context.Background(), "tables", fqn, nil)
			if err != nil {
				return err
			}
			if raw == nil {
				return fmt.Errorf("table %s was deleted by terraform destroy", fqn)
			}
			return nil
		},
		Steps: []resource.TestStep{
			// Step 1: Create + Read
			{
				Config: testAccDataAssetConfig(fqn, suffix, "Initial description", "PII.Sensitive"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "type", "table"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "name", "orders"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "display_name", "Orders"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "description", "Initial description"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "owners.0.type", "team"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "domains.#", "1"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "tags.0", "PII.Sensitive"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "glossary_terms.#", "1"),
					resource.TestCheckResourceAttrSet("openmetadata_data_asset.test", "id"),
				),
			},
			// Step 2: Update
			{
				Config: testAccDataAssetConfig(fqn, suffix, "Updated description", "PII.NonSensitive"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("openmetadata_data_asset.test", "tags.0", "PII.NonSensitive"),
				),
			},
			// Step 3: Import by "<type>:<fqn>"
			{
				ResourceName:      "openmetadata_data_asset.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_data_asset.test"]
					return "table:" + rs.Primary.Attributes["fully_qualified_name"], nil
				},
			},
		},
	})
}

func testAccDataAssetConfig(fqn, suffix, description, tag string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_domain" "test" {
  name        = %q
  description = "Domain for data asset test"
  domain_type = "Source-aligned"
}

resource "openmetadata_team" "test" {
  name      = %q
  team_type = "Group"
}

resource "openmetadata_glossary" "test" {
  name        = %q
  description = "Glossary for data asset test"
}

resource "openmetadata_glossary_term" "test" {
  name        = "Term"
  description = "Term for data asset test"
  glossary    = openmetadata_glossary.test.name
}

resource "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = %q
  display_name         = "Orders"
  description          = %q
  owners               = [{ id = openmetadata_team.test.id, type = "team" }]
  domains              = [openmetadata_domain.test.name]
  tags                 = [%q]
  glossary_terms       = [openmetadata_glossary_term.test.fully_qualified_name]
}
`, testProviderBlock(), suffix+"_d", suffix+"_t", suffix+"_g", fqn, description, tag)
}

// TestAccDataAssetResourceNotFound verifies that adopting a missing asset
// fails instead of creating one.
func TestAccDataAssetResourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

resource "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = "tfacc_missing.db.public.nothing"
  description          = "never applied"
}
`, testProviderBlock()),
				ExpectError: regexp.MustCompile(`Data asset not found`),
			},
		},
	})
}
