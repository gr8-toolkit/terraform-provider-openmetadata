// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name openmetadata

package main

import (
	"context"
	"log"

	"github.com/gr8-toolkit/terraform-provider-openmetadata/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set by goreleaser via ldflags.
var version string = "dev"

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/gr8-toolkit/openmetadata",
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
