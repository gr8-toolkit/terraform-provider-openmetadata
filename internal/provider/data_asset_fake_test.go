// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests exercise openmetadata_data_asset against the in-memory fakeOM.
// The fake stores pre-seeded ("ingested") assets, applies RFC 6902 "add"
// operations from PATCH requests, and, like the real server, returns the
// requested fields only and in an order of its own choosing.

const (
	fakeTableCollection = "tables"
	fakeTopicCollection = "topics"
	fakeAssetAddr       = "openmetadata_data_asset.test"
	fakeAssetDSAddr     = "data.openmetadata_data_asset.test"
	fakeTableFQN        = "fakesvc.fakedb.public.orders"
)

// fakeAssetCollections are the collections the fake serves data assets for.
var fakeAssetCollections = []string{fakeTableCollection, fakeTopicCollection}

// seedAsset registers an existing (ingested) asset and returns its UUID.
func (f *fakeOM) seedAsset(collection, fqn string, extra map[string]interface{}) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	entity := map[string]interface{}{
		"id":                 id,
		"name":               fqn[strings.LastIndex(fqn, ".")+1:],
		"fullyQualifiedName": fqn,
	}
	for k, v := range extra {
		entity[k] = v
	}
	f.assets[collection+"/"+fqn] = entity
	return id
}

func (f *fakeOM) patchCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.patches)
}

func (f *fakeOM) assetCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.assets)
}

// lastPatchValue returns the value of the "add" operation on path in the most
// recent PATCH request, or an error if there was none.
func (f *fakeOM) lastPatchValue(path string) (interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.patches) == 0 {
		return nil, fmt.Errorf("no PATCH request was received")
	}
	for _, op := range f.patches[len(f.patches)-1] {
		m, _ := op.(map[string]interface{})
		if m["path"] == path {
			return m["value"], nil
		}
	}
	return nil, fmt.Errorf("last PATCH has no operation on %s: %v", path, f.patches[len(f.patches)-1])
}

func (f *fakeOM) isAssetRoute(path string) bool {
	for _, c := range fakeAssetCollections {
		if strings.HasPrefix(path, c+"/") {
			return true
		}
	}
	return false
}

// handleAsset serves GET <collection>/name/<fqn> and PATCH <collection>/<id>.
// The caller holds f.mu.
func (f *fakeOM) handleAsset(w http.ResponseWriter, r *http.Request, path string) {
	collection, rest, _ := strings.Cut(path, "/")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(rest, "name/"):
		fqn := strings.TrimPrefix(rest, "name/")
		entity, ok := f.assets[collection+"/"+fqn]
		if !ok {
			writeAPIError(w, http.StatusNotFound, fmt.Sprintf("%s instance for %s not found", collection, fqn))
			return
		}
		writeJSON(w, http.StatusOK, f.projectAsset(entity, r.URL.Query().Get("fields")))

	case r.Method == http.MethodPatch:
		if ct := r.Header.Get("Content-Type"); ct != "application/json-patch+json" {
			writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported content type "+ct)
			return
		}
		var ops []interface{}
		if err := json.NewDecoder(r.Body).Decode(&ops); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid JSON patch: "+err.Error())
			return
		}
		var entity map[string]interface{}
		for key, e := range f.assets {
			if strings.HasPrefix(key, collection+"/") && e["id"] == rest {
				entity = e
			}
		}
		if entity == nil {
			writeAPIError(w, http.StatusNotFound, fmt.Sprintf("%s instance for %s not found", collection, rest))
			return
		}
		f.patches = append(f.patches, ops)
		for _, op := range ops {
			m, _ := op.(map[string]interface{})
			field := strings.TrimPrefix(fmt.Sprint(m["path"]), "/")
			if m["op"] != "add" || field == "" || strings.Contains(field, "/") {
				writeAPIError(w, http.StatusBadRequest, fmt.Sprintf("unsupported patch operation %v", m))
				return
			}
			entity[field] = m["value"]
		}
		writeJSON(w, http.StatusOK, entity)

	default:
		writeAPIError(w, http.StatusNotFound, "no such fake route: "+r.Method+" "+path)
	}
}

// projectAsset mimics the server: relationship fields are present only when
// requested via ?fields=, domain refs are expanded with their name and FQN,
// and tags come back in reverse order. The caller holds f.mu.
func (f *fakeOM) projectAsset(entity map[string]interface{}, fields string) map[string]interface{} {
	optional := map[string]bool{"owners": true, "domains": true, "tags": true, "extension": true}
	want := map[string]bool{}
	for _, f := range strings.Split(fields, ",") {
		want[f] = true
	}
	out := map[string]interface{}{}
	for k, v := range entity {
		if optional[k] && !want[k] {
			continue
		}
		out[k] = v
	}
	if tags, ok := out["tags"].([]interface{}); ok {
		reversed := make([]interface{}, len(tags))
		for i, t := range tags {
			reversed[len(tags)-1-i] = t
		}
		out["tags"] = reversed
	}
	if domains, ok := out["domains"].([]interface{}); ok {
		expanded := make([]interface{}, 0, len(domains))
		for _, d := range domains {
			ref, _ := d.(map[string]interface{})
			for _, svc := range f.services {
				if svc["id"] == ref["id"] {
					expanded = append(expanded, map[string]interface{}{
						"id": svc["id"], "type": "domain",
						"name": svc["name"], "fullyQualifiedName": svc["fullyQualifiedName"],
					})
				}
			}
		}
		out["domains"] = expanded
	}
	return out
}

func fakeAssetConfig(f *fakeOM, extra string) string {
	return fmt.Sprintf(`
%s

resource "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = %q
%s
}
`, f.providerBlock(), fakeTableFQN, extra)
}

func seedIngestedTable(f *fakeOM) string {
	return f.seedAsset(fakeTableCollection, fakeTableFQN, map[string]interface{}{
		"description": "Ingested description",
		"tags": []interface{}{
			map[string]interface{}{"tagFQN": "Tier.Tier3", "source": "Classification", "labelType": "Automated", "state": "Suggested"},
		},
	})
}

func TestFakeDataAssetLifecycle(t *testing.T) {
	f := newFakeOM(t)
	seedIngestedTable(f)
	domainID := f.seedService("domains", "domain", "Analytics")
	const ownerID = "11111111-1111-4111-8111-111111111111"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(_ *terraform.State) error {
			if n := f.assetCount(); n != 1 {
				return fmt.Errorf("%d asset(s) left after destroy, want the ingested one untouched", n)
			}
			return nil
		},
		Steps: []resource.TestStep{
			// ── Adopt without managing anything: reads, writes nothing ────────
			{
				Config: fakeAssetConfig(f, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeAssetAddr, "name", "orders"),
					resource.TestCheckResourceAttrSet(fakeAssetAddr, "id"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "description", "Ingested description"),
					resource.TestCheckNoResourceAttr(fakeAssetAddr, "display_name"),
					resource.TestCheckNoResourceAttr(fakeAssetAddr, "extension_json"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "owners.#", "0"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "domains.#", "0"),
					// The automated Tier tag is not a manual tag.
					resource.TestCheckResourceAttr(fakeAssetAddr, "tags.#", "0"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "glossary_terms.#", "0"),
					func(_ *terraform.State) error {
						if n := f.patchCount(); n != 0 {
							return fmt.Errorf("%d PATCH request(s) sent although nothing was configured", n)
						}
						return nil
					},
				),
			},
			// ── Manage every attribute ────────────────────────────────────────
			{
				Config: fakeAssetConfig(f, fmt.Sprintf(`
  display_name   = "Orders"
  description    = "Curated description"
  owners         = [{ id = %q, type = "user" }]
  domains        = ["Analytics"]
  tags           = ["PII.Sensitive", "PII.Email"]
  glossary_terms = ["Finance.Revenue"]
  extension_json = jsonencode({ retentionDays = 365, steward = "data-team" })
`, ownerID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeAssetAddr, "display_name", "Orders"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "description", "Curated description"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "owners.#", "1"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "owners.0.id", ownerID),
					resource.TestCheckResourceAttr(fakeAssetAddr, "owners.0.type", "user"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "domains.#", "1"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "domains.0", "Analytics"),
					// Order as written, even though the fake returns tags reversed.
					resource.TestCheckResourceAttr(fakeAssetAddr, "tags.0", "PII.Sensitive"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "tags.1", "PII.Email"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "glossary_terms.0", "Finance.Revenue"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "extension_json", `{"retentionDays":365,"steward":"data-team"}`),
					func(_ *terraform.State) error {
						tags, err := f.lastPatchValue("/tags")
						if err != nil {
							return err
						}
						b, _ := json.Marshal(tags)
						got := string(b)
						// The automated label survives; manual ones are Confirmed.
						for _, want := range []string{
							`"tagFQN":"Tier.Tier3"`, `"labelType":"Automated"`,
							`"tagFQN":"PII.Sensitive"`, `"tagFQN":"PII.Email"`,
							`"source":"Glossary"`, `"labelType":"Manual"`, `"state":"Confirmed"`,
						} {
							if !strings.Contains(got, want) {
								return fmt.Errorf("tags patch %s is missing %s", got, want)
							}
						}
						domains, err := f.lastPatchValue("/domains")
						if err != nil {
							return err
						}
						b, _ = json.Marshal(domains)
						if want := fmt.Sprintf(`[{"id":%q,"type":"domain"}]`, domainID); string(b) != want {
							return fmt.Errorf("domains patch = %s, want %s", b, want)
						}
						return nil
					},
				),
			},
			// ── Re-applying the same configuration is a no-op ─────────────────
			{
				Config: fakeAssetConfig(f, fmt.Sprintf(`
  display_name   = "Orders"
  description    = "Curated description"
  owners         = [{ id = %q, type = "user" }]
  domains        = ["Analytics"]
  tags           = ["PII.Sensitive", "PII.Email"]
  glossary_terms = ["Finance.Revenue"]
  extension_json = jsonencode({ steward = "data-team", retentionDays = 365 })
`, ownerID)),
				PlanOnly: true,
			},
			// ── Empty lists clear; omitted attributes stay untouched ──────────
			{
				Config: fakeAssetConfig(f, `
  tags    = []
  domains = []
  owners  = []
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fakeAssetAddr, "tags.#", "0"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "domains.#", "0"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "owners.#", "0"),
					// Not in the configuration any more, so left as it was.
					resource.TestCheckResourceAttr(fakeAssetAddr, "description", "Curated description"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "glossary_terms.0", "Finance.Revenue"),
					resource.TestCheckResourceAttr(fakeAssetAddr, "extension_json", `{"retentionDays":365,"steward":"data-team"}`),
					func(_ *terraform.State) error {
						tags, err := f.lastPatchValue("/tags")
						if err != nil {
							return err
						}
						b, _ := json.Marshal(tags)
						got := string(b)
						if !strings.Contains(got, "Tier.Tier3") || !strings.Contains(got, "Finance.Revenue") ||
							strings.Contains(got, "PII.") {
							return fmt.Errorf("tags patch = %s, want only the automated tag and the glossary term", got)
						}
						return nil
					},
				),
			},
			// ── Import by "<type>:<fqn>" ──────────────────────────────────────
			{
				ResourceName:      fakeAssetAddr,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "table:" + fakeTableFQN,
			},
		},
	})
}

func TestFakeDataAssetDataSource(t *testing.T) {
	f := newFakeOM(t)
	f.seedAsset(fakeTopicCollection, "fakekafka.events", map[string]interface{}{
		"description": "Event stream",
		"tags": []interface{}{
			map[string]interface{}{"tagFQN": "Tier.Tier1", "source": "Classification", "labelType": "Automated", "state": "Suggested"},
			map[string]interface{}{"tagFQN": "Finance.Revenue", "source": "Glossary", "labelType": "Manual", "state": "Confirmed"},
		},
		"owners":    []interface{}{map[string]interface{}{"id": "22222222-2222-4222-8222-222222222222", "type": "team"}},
		"extension": map[string]interface{}{"b": 1, "a": "x"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "openmetadata_data_asset" "test" {
  type                 = "topic"
  fully_qualified_name = "fakekafka.events"
}
`, f.providerBlock()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fakeAssetDSAddr, "id"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "name", "events"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "description", "Event stream"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "owners.0.type", "team"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "domains.#", "0"),
					// Unlike the resource, the data source reports labels of every kind.
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "tags.#", "1"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "tags.0", "Tier.Tier1"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "glossary_terms.0", "Finance.Revenue"),
					resource.TestCheckResourceAttr(fakeAssetDSAddr, "extension_json", `{"a":"x","b":1}`),
				),
			},
		},
	})
}

func TestFakeDataAssetNotFound(t *testing.T) {
	f := newFakeOM(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fakeAssetConfig(f, `description = "x"`),
				ExpectError: regexp.MustCompile(`Data asset not found`),
			},
			{
				Config: fmt.Sprintf(`
%s

data "openmetadata_data_asset" "test" {
  type                 = "table"
  fully_qualified_name = "nope.nope.nope.nope"
}
`, f.providerBlock()),
				ExpectError: regexp.MustCompile(`Data asset not found`),
			},
		},
	})
}

// TestFakeDataAssetUnknownDomain checks that a domain that does not exist is
// reported instead of silently dropped.
func TestFakeDataAssetUnknownDomain(t *testing.T) {
	f := newFakeOM(t)
	seedIngestedTable(f)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      fakeAssetConfig(f, `domains = ["NoSuchDomain"]`),
				ExpectError: regexp.MustCompile(`Domain not found`),
			},
		},
	})
}
