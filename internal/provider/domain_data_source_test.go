// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainDataSource(t *testing.T) {
	name := testRandName("dom")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_domain.test", "id",
						"openmetadata_domain.test", "id"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_domain.test", "name",
						"openmetadata_domain.test", "name"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_domain.test", "description",
						"openmetadata_domain.test", "description"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_domain.test", "domain_type",
						"openmetadata_domain.test", "domain_type"),
					resource.TestCheckResourceAttrPair(
						"data.openmetadata_domain.test", "fully_qualified_name",
						"openmetadata_domain.test", "fully_qualified_name"),
					resource.TestCheckResourceAttrSet("data.openmetadata_domain.test", "id"),
				),
			},
		},
	})
}

// TestAccDomainDataSourceNoDrift verifies that re-reading a domain data source
// after creation produces a consistent plan (no unexpected changes).
func TestAccDomainDataSourceNoDrift(t *testing.T) {
	name := testRandName("dom")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDataSourceConfig(name),
				Check:  resource.TestCheckResourceAttrSet("data.openmetadata_domain.test", "id"),
			},
			// Plan-only: data sources are re-read on every plan; verify no diff.
			{
				Config:             testAccDomainDataSourceConfig(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccDomainDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDomainDataSourceNotFoundConfig(),
				ExpectError: regexp.MustCompile(`Domain not found`),
			},
		},
	})
}

func testAccDomainDataSourceConfig(name string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_domain" "test" {
  name        = %q
  description = "Domain data source acceptance test"
  domain_type = "Source-aligned"
}

data "openmetadata_domain" "test" {
  name = openmetadata_domain.test.name
}
`, testProviderBlock(), name)
}

func testAccDomainDataSourceNotFoundConfig() string {
	return fmt.Sprintf(`
%s

data "openmetadata_domain" "test" {
  name = "tfacc_dom_does_not_exist_99999"
}
`, testProviderBlock())
}
