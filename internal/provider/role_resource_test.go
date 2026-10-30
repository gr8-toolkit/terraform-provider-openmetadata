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

// TestAccRoleResource exercises the full CRUD + import lifecycle of
// openmetadata_role.
//
// The `policies` field references "OrganizationPolicy", which is seeded in
// every fresh OpenMetadata installation. Adjust if your instance uses a
// different default policy name.
//
// Import state verification ignores `policies` because the API returns policy
// references as fully-qualified names, which may differ from the short names
// sent during create (e.g. "OrganizationPolicy" vs "OrganizationPolicy").
// Once the provider normalises FQNs on read, this ignore can be removed.
func TestAccRoleResource(t *testing.T) {
	name := testRandName("role")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			{
				Config: testAccRoleConfig(name, "Initial role description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_role.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Initial role description"),
					resource.TestCheckResourceAttrSet("openmetadata_role.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_role.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccRoleConfig(name, "Updated role description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Updated role description"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). The provider's ImportState uses GetByName.
			{
				ResourceName:            "openmetadata_role.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"policies"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_role.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccRoleResourceBotToken updates the description with the ingestion-bot
// token. OM ignores description changes made by bots through PUT and returns
// the old value, which used to fail with "inconsistent result after apply".
func TestAccRoleResourceBotToken(t *testing.T) {
	name := testRandName("rlbot")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfigWithProvider(testBotProviderBlock(t), name, "Initial role description"),
				Check:  resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Initial role description"),
			},
			{
				Config: testAccRoleConfigWithProvider(testBotProviderBlock(t), name, "Updated role description"),
				Check:  resource.TestCheckResourceAttr("openmetadata_role.test", "description", "Updated role description"),
			},
		},
	})
}

func testAccRoleConfig(name, description string) string {
	return testAccRoleConfigWithProvider(testProviderBlock(), name, description)
}

func testAccRoleConfigWithProvider(providerBlock, name, description string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_role" "test" {
  name        = %q
  description = %q
  # OrganizationPolicy is seeded in every fresh OpenMetadata installation.
  policies = ["OrganizationPolicy"]
}
`, providerBlock, name, description)
}

// TestAccRoleResourceSanitizedDescription covers descriptions that OM rewrites
// while storing them (unknown HTML tags are dropped, `&` is escaped): state must
// keep the configured text instead of failing with "inconsistent result after apply".
func TestAccRoleResourceSanitizedDescription(t *testing.T) {
	name := testRandName("rolsan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig(name, testSanitizedDescription),
				Check:  resource.TestCheckResourceAttr("openmetadata_role.test", "description", testSanitizedDescription),
			},
			{
				Config: testAccRoleConfig(name, testSanitizedDescriptionUpdated),
				Check:  resource.TestCheckResourceAttr("openmetadata_role.test", "description", testSanitizedDescriptionUpdated),
			},
		},
	})
}

// TestAccRoleResourceBotTokenDisplayName: bots cannot edit display names (OM
// ignores it on PUT and denies it on PATCH), which must surface as a clear
// error rather than an inconsistent result.
func TestAccRoleResourceBotTokenDisplayName(t *testing.T) {
	name := testRandName("rldn")
	botProvider := testBotProviderBlock(t)
	config := func(displayName string) string {
		return fmt.Sprintf(`
%s

resource "openmetadata_role" "test" {
  name         = %q
  display_name = %q
  policies     = ["OrganizationPolicy"]
}
`, botProvider, name, displayName)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("First"),
				Check:  resource.TestCheckResourceAttr("openmetadata_role.test", "display_name", "First"),
			},
			{
				Config:      config("Second"),
				ExpectError: regexp.MustCompile(`bot tokens cannot edit\s+display\s+names`),
			},
		},
	})
}
