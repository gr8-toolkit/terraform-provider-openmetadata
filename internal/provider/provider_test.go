// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

// Package provider_test contains acceptance tests for the OpenMetadata Terraform provider.
//
// # Acceptance tests
//
// Acceptance tests require a live OpenMetadata instance. Set:
//
//	OPENMETADATA_HOST  – e.g. http://localhost:8585
//	OPENMETADATA_TOKEN – JWT token for an admin user
//
// Gate tests with TF_ACC=1 so they are skipped in the regular unit-test pass:
//
//	TF_ACC=1 go test -v -count=1 -timeout 30m ./internal/provider/...
//
// Use the Makefile target for the full local lifecycle (starts docker-compose,
// acquires a token, runs tests, and tears down):
//
//	make testacc
package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/client"
	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/provider"
)

// testAccProtoV6ProviderFactories is shared by every acceptance test in this
// package. It wires the in-process provider server so no external binary is
// needed at test time.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"openmetadata": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccPreCheck aborts the test if the required environment variables are absent.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, env := range []string{"OPENMETADATA_HOST", "OPENMETADATA_TOKEN"} {
		if os.Getenv(env) == "" {
			t.Fatalf("acceptance tests require the %s environment variable to be set", env)
		}
	}
}

// testRandName returns a short unique name safe for use as an OpenMetadata
// entity name (alphanumeric + underscores, ≤ 63 chars).
// prefix must be ≤ 6 characters.
func testRandName(prefix string) string {
	return fmt.Sprintf("tfacc_%s_%05d", prefix, rand.Intn(99999)) //nolint:gosec
}

// testProviderBlock returns the minimal provider HCL block.
// The provider reads OPENMETADATA_HOST and OPENMETADATA_TOKEN from the environment.
func testProviderBlock() string {
	return `
provider "openmetadata" {}
`
}

// testBotProviderBlock returns a provider block that authenticates with the
// ingestion-bot JWT instead of OPENMETADATA_TOKEN. OM does not let bots change
// an existing description or displayName through PUT, so resources must cope
// with that to work for bot tokens. The bot token is fetched with the admin
// token from the environment.
func testBotProviderBlock(t *testing.T) string {
	t.Helper()
	// Runs while the TestCase is being built, i.e. before resource.Test applies
	// its own TF_ACC gate, so it must skip on its own.
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}
	testAccPreCheck(t)
	ctx := context.Background()
	c := client.NewClient(os.Getenv("OPENMETADATA_HOST"), os.Getenv("OPENMETADATA_TOKEN"))

	rawBot, err := c.GetByName(ctx, "bots", "ingestion-bot", nil)
	if err != nil || rawBot == nil {
		t.Fatalf("fetching ingestion-bot: %v", err)
	}
	var bot struct {
		BotUser struct {
			ID string `json:"id"`
		} `json:"botUser"`
	}
	if err := json.Unmarshal(rawBot, &bot); err != nil || bot.BotUser.ID == "" {
		t.Fatalf("ingestion-bot has no botUser: %v (%s)", err, rawBot)
	}

	rawAuth, err := c.GetByID(ctx, "users/auth-mechanism", bot.BotUser.ID, nil)
	if err != nil || rawAuth == nil {
		t.Fatalf("fetching ingestion-bot auth mechanism: %v", err)
	}
	var auth struct {
		Config struct {
			JWTToken string `json:"JWTToken"`
		} `json:"config"`
	}
	if err := json.Unmarshal(rawAuth, &auth); err != nil || auth.Config.JWTToken == "" {
		t.Fatalf("ingestion-bot has no JWT token: %v", err)
	}

	return fmt.Sprintf(`
provider "openmetadata" {
  host  = %q
  token = %q
}
`, os.Getenv("OPENMETADATA_HOST"), auth.Config.JWTToken)
}
