// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccTeamResource exercises the full CRUD + import lifecycle of
// openmetadata_team.
func TestAccTeamResource(t *testing.T) {
	name := testRandName("team")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create and Read ──────────────────────────────────────────────
			// Use Department (not Group): the OM API does not allow updating Group
			// teams, but Department teams can be updated normally.
			{
				Config: testAccTeamConfig(name, "Initial team description", "Department"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_team.test", "description", "Initial team description"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "team_type", "Department"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "is_joinable", "true"),
					resource.TestCheckResourceAttrSet("openmetadata_team.test", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_team.test", "fully_qualified_name"),
				),
			},
			// ── Update description ────────────────────────────────────────────
			{
				Config: testAccTeamConfig(name, "Updated team description", "Department"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "description", "Updated team description"),
					resource.TestCheckResourceAttr("openmetadata_team.test", "team_type", "Department"),
				),
			},
			// ── Import ───────────────────────────────────────────────────────
			// Import by name (not UUID). parents is ignored: OM always places
			// teams under "Organisation" and the FQN may differ from short name.
			{
				ResourceName:            "openmetadata_team.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parents"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_team.test"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// TestAccTeamResourceWithEmail verifies the optional email field.
func TestAccTeamResourceWithEmail(t *testing.T) {
	name := testRandName("team")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfigWithEmail(name, "team@example.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.test", "name", name),
					resource.TestCheckResourceAttr("openmetadata_team.test", "email", "team@example.com"),
				),
			},
		},
	})
}

// TestAccTeamResourceWithParents verifies that a team can be placed under a
// parent team using the parents field. parents is preserved in state from the
// plan (not re-read from the API) so no drift occurs on subsequent plans.
// Import ignores parents because the API does not expose them in a form we can
// round-trip back to the user-supplied names.
func TestAccTeamResourceWithParents(t *testing.T) {
	parentName := testRandName("tp")
	childName := testRandName("tc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create parent and child ────────────────────────────────────────
			{
				Config: testAccTeamConfigWithParents(parentName, childName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.parent", "name", parentName),
					resource.TestCheckResourceAttr("openmetadata_team.child", "name", childName),
					resource.TestCheckResourceAttr("openmetadata_team.child", "team_type", "Department"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "parents.#", "1"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "parents.0", parentName),
					resource.TestCheckResourceAttrSet("openmetadata_team.child", "id"),
				),
			},
			// ── Import child ────────────────────────────────────────────────
			{
				ResourceName:            "openmetadata_team.child",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parents"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_team.child"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccTeamConfig(name, description, teamType string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "test" {
  name        = %q
  description = %q
  team_type   = %q
}
`, testProviderBlock(), name, description, teamType)
}

func testAccTeamConfigWithEmail(name, email string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "test" {
  name  = %q
  email = %q
}
`, testProviderBlock(), name, email)
}

func testAccTeamConfigWithParents(parentName, childName string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "parent" {
  name      = %q
  team_type = "Department"
}

resource "openmetadata_team" "child" {
  name      = %q
  team_type = "Department"
  parents   = [openmetadata_team.parent.name]
}
`, testProviderBlock(), parentName, childName)
}

// TestAccTeamResourceWithParentByID verifies that parents can be specified
// using the parent team's UUID (openmetadata_team.foo.id) rather than its name.
// This is the natural Terraform idiom when both teams are managed in the same
// config and you want an implicit dependency.
func TestAccTeamResourceWithParentByID(t *testing.T) {
	parentName := testRandName("tp")
	childName := testRandName("tc")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Create parent (Division) and child (Department) ───────────────
			{
				Config: testAccTeamConfigWithParentByID(parentName, childName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.parent", "name", parentName),
					resource.TestCheckResourceAttr("openmetadata_team.child", "name", childName),
					resource.TestCheckResourceAttr("openmetadata_team.child", "team_type", "Department"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "is_joinable", "false"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "parents.#", "1"),
					resource.TestCheckResourceAttrSet("openmetadata_team.child", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_team.child", "fully_qualified_name"),
				),
			},
			// ── Import child — parents is ignored (write-only) ────────────────
			{
				ResourceName:            "openmetadata_team.child",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parents"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_team.child"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

func testAccTeamConfigWithParentByID(parentName, childName string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "parent" {
  name        = %q
  display_name = "Parent Division"
  team_type   = "Division"
  is_joinable = false
}

resource "openmetadata_team" "child" {
  name        = %q
  display_name = "Child Department"
  description = "Sub-team under the parent division."
  team_type   = "Department"
  is_joinable = false
  parents     = [openmetadata_team.parent.id]
}
`, testProviderBlock(), parentName, childName)
}

// TestAccTeamResourceParentReparent tests that a team's parent relationship can
// be updated via config — the primary remediation path when team hierarchy has
// been changed outside Terraform (manually in the OM UI or by another process).
//
// Because the parents field is write-only (not read back from the API), manual
// drift is not auto-detected on a plain `terraform plan`. The correct workflow
// is: discover the drift → update the config to the desired parent → apply.
// This test exercises that full cycle:
//
//  1. Create two Division teams and one Department team under Division A.
//  2. Re-parent the Department to Division B (simulating the corrective apply).
//  3. Import verifies the child team's state is consistent post-reparent.
func TestAccTeamResourceParentReparent(t *testing.T) {
	parentAName := testRandName("diva")
	parentBName := testRandName("divb")
	childName := testRandName("dept")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ── Step 1: Initial hierarchy — Department under Division A ───────
			{
				Config: testAccTeamReparentConfig(parentAName, parentBName, childName, "parent_a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openmetadata_team.parent_a", "id"),
					resource.TestCheckResourceAttrSet("openmetadata_team.parent_b", "id"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "name", childName),
					resource.TestCheckResourceAttr("openmetadata_team.child", "team_type", "Department"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "is_joinable", "false"),
					resource.TestCheckResourceAttr("openmetadata_team.child", "parents.#", "1"),
					// State holds the plan value (write-only); assert it equals parent_a's UUID.
					resource.TestCheckResourceAttrPair(
						"openmetadata_team.child", "parents.0",
						"openmetadata_team.parent_a", "id",
					),
				),
			},
			// ── Step 2: Re-parent Department to Division B ───────────────────
			// Mirrors the corrective apply after discovering a manual change:
			// someone moved the team to Division B in the OM UI, the operator
			// updates the config accordingly, and `terraform apply` pushes the
			// correct relationship back to the API.
			{
				Config: testAccTeamReparentConfig(parentAName, parentBName, childName, "parent_b"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openmetadata_team.child", "parents.#", "1"),
					// State must now reflect Division B's UUID, not Division A's.
					resource.TestCheckResourceAttrPair(
						"openmetadata_team.child", "parents.0",
						"openmetadata_team.parent_b", "id",
					),
					resource.TestCheckResourceAttrSet("openmetadata_team.child", "fully_qualified_name"),
				),
			},
			// ── Step 3: Import — verify state is consistent after reparent ───
			// parents is excluded: the API always adds Organisation as an implicit
			// ancestor, making the round-trip value unpredictable.
			{
				ResourceName:            "openmetadata_team.child",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"parents"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["openmetadata_team.child"]
					return rs.Primary.Attributes["name"], nil
				},
			},
		},
	})
}

// testAccTeamReparentConfig builds a config with two Division teams and one
// Department team. activeParentRef selects which Division is the active parent
// ("parent_a" or "parent_b") so the same helper drives both test steps.
func testAccTeamReparentConfig(parentAName, parentBName, childName, activeParentRef string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_team" "parent_a" {
  name         = %q
  display_name = "Division A"
  team_type    = "Division"
  is_joinable  = false
}

resource "openmetadata_team" "parent_b" {
  name         = %q
  display_name = "Division B"
  team_type    = "Division"
  is_joinable  = false
}

resource "openmetadata_team" "child" {
  name         = %q
  display_name = "Department"
  team_type    = "Department"
  is_joinable  = false
  parents      = [openmetadata_team.%s.id]
}
`, testProviderBlock(), parentAName, parentBName, childName, activeParentRef)
}
