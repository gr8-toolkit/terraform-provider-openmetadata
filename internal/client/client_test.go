// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// botServer mimics OM for a bot token: PUT keeps the stored description and
// displayName, PATCH applies them.
func botServer(t *testing.T, patched *[]map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			_, _ = w.Write([]byte(`{"id":"abc","name":"x","description":"old","displayName":"Old"}`))
		case http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(b, patched); err != nil {
				t.Errorf("bad patch body: %v", err)
			}
			if r.Header.Get("Content-Type") != "application/json-patch+json" {
				t.Errorf("unexpected content type %q", r.Header.Get("Content-Type"))
			}
			_, _ = w.Write([]byte(`{"id":"abc","name":"x","description":"new","displayName":"Old"}`))
		}
	}))
}

func TestCreateOrUpdatePatchesIgnoredDescription(t *testing.T) {
	var patched []map[string]interface{}
	srv := botServer(t, &patched)
	defer srv.Close()

	raw, err := NewClient(srv.URL, "tok").CreateOrUpdate(context.Background(), "roles",
		map[string]interface{}{"name": "x", "description": "new", "displayName": "Old"})
	if err != nil {
		t.Fatal(err)
	}
	if len(patched) != 1 || patched[0]["path"] != "/description" || patched[0]["value"] != "new" {
		t.Fatalf("expected a single description patch, got %v", patched)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(raw, &got)
	if got["description"] != "new" {
		t.Fatalf("expected patched entity, got %s", raw)
	}
}

func TestCreateOrUpdateSkipsPatchWhenApplied(t *testing.T) {
	var patched []map[string]interface{}
	srv := botServer(t, &patched)
	defer srv.Close()

	if _, err := NewClient(srv.URL, "tok").CreateOrUpdate(context.Background(), "roles",
		map[string]interface{}{"name": "x", "description": "old"}); err != nil {
		t.Fatal(err)
	}
	if patched != nil {
		t.Fatalf("unexpected patch %v", patched)
	}
}
