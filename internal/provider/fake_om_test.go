// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// fakeOM is a minimal in-memory stand-in for the parts of the OpenMetadata
// REST API used by the ingestion pipeline resource and data source. It lets the
// full Terraform plan/apply/import/destroy cycle run without a live OM stack,
// so provider-side bugs such as "inconsistent result after apply" are caught by
// `make test`. It mimics the server behaviour that matters here: UUID
// assignment, FQN construction, default values for omitted fields, and the
// `{code, message}` error body.
//
// It is not a substitute for the acceptance tests, which validate against the
// real server.
type fakeOM struct {
	t      *testing.T
	server *httptest.Server

	mu        sync.Mutex
	seq       int
	services  map[string]map[string]interface{} // "<collection>/<name>" -> entity
	pipelines map[string]map[string]interface{} // fqn -> entity

	puts    []map[string]interface{} // decoded PUT bodies for pipelines, in order
	deletes []string                 // ids of deleted pipelines
}

// newFakeOM starts the fake server and registers its shutdown with t.Cleanup.
// The test is skipped when no Terraform CLI is available, because the plugin
// testing framework drives a real terraform binary.
func newFakeOM(t *testing.T) *fakeOM {
	t.Helper()
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		if _, err := exec.LookPath("terraform"); err != nil {
			t.Skip("terraform CLI not found on PATH; skipping fake-server test")
		}
	}
	f := &fakeOM{
		t:         t,
		services:  map[string]map[string]interface{}{},
		pipelines: map[string]map[string]interface{}{},
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

// providerBlock returns a provider block pointing at the fake server.
func (f *fakeOM) providerBlock() string {
	return fmt.Sprintf(`
provider "openmetadata" {
  host  = %q
  token = "fake-token"
}
`, f.server.URL)
}

func (f *fakeOM) nextID() string {
	f.seq++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
}

// seedService registers an existing service/test suite and returns its UUID.
func (f *fakeOM) seedService(collection, entityType, name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	f.services[collection+"/"+name] = map[string]interface{}{
		"id":                 id,
		"name":               name,
		"fullyQualifiedName": name,
		"type":               entityType, // remembered for building the service reference
	}
	return id
}

func (f *fakeOM) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.puts)
}

func (f *fakeOM) deleteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.deletes)
}

func (f *fakeOM) pipelineCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pipelines)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]interface{}{"code": status, "message": msg})
}

func (f *fakeOM) handle(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer fake-token" {
		writeAPIError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	f.mu.Lock()
	defer f.mu.Unlock()

	switch {
	case r.Method == http.MethodGet && path == "system/version":
		writeJSON(w, http.StatusOK, map[string]string{"version": "fake"})

	case path == "services/ingestionPipelines" && r.Method == http.MethodPut:
		f.putPipeline(w, r)

	case strings.HasPrefix(path, "services/ingestionPipelines/name/") && r.Method == http.MethodGet:
		fqn := strings.TrimPrefix(path, "services/ingestionPipelines/name/")
		if p, ok := f.pipelines[fqn]; ok {
			writeJSON(w, http.StatusOK, p)
			return
		}
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("ingestionPipeline instance for %s not found", fqn))

	case strings.HasPrefix(path, "services/ingestionPipelines/") && r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "services/ingestionPipelines/")
		for fqn, p := range f.pipelines {
			if p["id"] == id {
				delete(f.pipelines, fqn)
				f.deletes = append(f.deletes, id)
				writeJSON(w, http.StatusOK, p)
				return
			}
		}
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("ingestionPipeline instance for %s not found", id))

	case r.Method == http.MethodGet && strings.Contains(path, "/name/"):
		// GET <collection>/name/<name> for services and test suites.
		idx := strings.Index(path, "/name/")
		key := path[:idx] + "/" + path[idx+len("/name/"):]
		if s, ok := f.services[key]; ok {
			writeJSON(w, http.StatusOK, s)
			return
		}
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("entity %s not found", key))

	default:
		writeAPIError(w, http.StatusNotFound, "no such fake route: "+r.Method+" "+path)
	}
}

// airflowDefaults mirrors the defaults OM applies to an AirflowConfig.
var airflowDefaults = map[string]interface{}{
	"pausePipeline":                  false,
	"concurrency":                    1,
	"pipelineTimezone":               "UTC",
	"retries":                        0,
	"retryDelay":                     300,
	"pipelineCatchup":                false,
	"maxActiveRuns":                  1,
	"workflowDefaultView":            "tree",
	"workflowDefaultViewOrientation": "LR",
}

// putPipeline implements PUT /services/ingestionPipelines. The caller holds f.mu.
func (f *fakeOM) putPipeline(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	// Record a deep copy: the handler below mutates sourceConfig to add defaults.
	var recorded map[string]interface{}
	if b, err := json.Marshal(body); err == nil {
		_ = json.Unmarshal(b, &recorded)
	}
	f.puts = append(f.puts, recorded)

	for _, required := range []string{"name", "service", "pipelineType", "sourceConfig", "airflowConfig"} {
		if body[required] == nil {
			writeAPIError(w, http.StatusBadRequest, "["+required+" must not be null]")
			return
		}
	}

	svcRef, _ := body["service"].(map[string]interface{})
	var svc map[string]interface{}
	for _, s := range f.services {
		if s["id"] == svcRef["id"] && s["type"] == svcRef["type"] {
			svc = s
			break
		}
	}
	if svc == nil {
		writeAPIError(w, http.StatusNotFound, fmt.Sprintf("service instance for %v not found", svcRef["id"]))
		return
	}

	name, _ := body["name"].(string)
	fqn := svc["fullyQualifiedName"].(string) + "." + name
	existing := f.pipelines[fqn]

	entity := map[string]interface{}{
		"name":               name,
		"fullyQualifiedName": fqn,
		"pipelineType":       body["pipelineType"],
		"service": map[string]interface{}{
			"id":                 svc["id"],
			"type":               svc["type"],
			"name":               svc["name"],
			"fullyQualifiedName": svc["fullyQualifiedName"],
		},
		"loggerLevel":          "INFO",
		"raiseOnError":         true,
		"enableStreamableLogs": false,
		"enabled":              true,
		"deployed":             false,
	}
	if existing != nil {
		entity["id"] = existing["id"]
		entity["deployed"] = existing["deployed"]
	} else {
		entity["id"] = f.nextID()
	}

	// sourceConfig: OM stores the config with server-side defaults filled in.
	sc, _ := body["sourceConfig"].(map[string]interface{})
	if cfg, ok := sc["config"].(map[string]interface{}); ok {
		if _, set := cfg["markDeletedTables"]; !set {
			cfg["markDeletedTables"] = true
		}
	}
	entity["sourceConfig"] = sc

	af := map[string]interface{}{}
	for k, v := range airflowDefaults {
		af[k] = v
	}
	if given, ok := body["airflowConfig"].(map[string]interface{}); ok {
		for k, v := range given {
			af[k] = v
		}
	}
	entity["airflowConfig"] = af

	for _, k := range []string{"displayName", "description", "loggerLevel", "raiseOnError", "enableStreamableLogs"} {
		if v, ok := body[k]; ok {
			entity[k] = v
		}
	}
	if owners, ok := body["owners"]; ok {
		entity["owners"] = owners
	}
	if domains, ok := body["domains"].([]interface{}); ok {
		refs := make([]interface{}, 0, len(domains))
		for _, d := range domains {
			refs = append(refs, map[string]interface{}{"id": f.nextID(), "type": "domain", "fullyQualifiedName": d})
		}
		entity["domains"] = refs
	}

	f.pipelines[fqn] = entity
	writeJSON(w, http.StatusOK, entity)
}
