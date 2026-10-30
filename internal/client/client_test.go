// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// sanitized pairs are descriptions sent to OM 2.0.7 and what it stored.
func TestEquivalentTextSanitizedByOM(t *testing.T) {
	for _, c := range [][2]string{
		{"a <env> b", "a  b"},
		{"x < y", "x &lt; y"},
		{"x > y", "x > y"},
		{"a & b", "a & b"},
		{"a &amp; b", "a &amp; b"},
		{"a &lt; b", "a &lt; b"},
		{"<<env>>", "&lt;&gt;"},
		{"om-<env>-consumers", "om--consumers"},
		{"<b>bold</b> & c", "<b>bold</b> &amp; c"},
		{"<div class=\"x\" onclick=\"evil()\">hi</div>", "<div class=\"x\">hi</div>"},
		{"<script>alert(1)</script>after", "after"},
		{"<style>p{}</style>text", "text"},
		{"<a href=\"http://a.b?x=1&y=2\" target=\"_blank\">l</a>", "<a href=\"http://a.b?x&#61;1&amp;y&#61;2\">l</a>"},
		{"<a href=\"javascript:x\">bad</a>", "bad"},
		{"<img src=\"x\" onerror=\"y\">", ""},
		{"<!-- comment -->text", "text"},
		{"&nbsp;x&nbsp;", "&nbsp;x&nbsp;"},
		{"> blockquote", "> blockquote"},
		{"`<code>`", "&#96;<code>&#96;</code>"},
		{"<code>a<b</code>", "<code>a<b></b></code>"},
		{"a <b>unclosed", "a <b>unclosed</b>"},
		{"if a<b and c>d then", "if a<b>d then</b>"},
		{"Members: <#E::table::db.sch.t> link", "Members: <#E::table::db.sch.t> link"},
		{"<env-name> and <ENV>", " and "},
		{"x <3 y", "x &lt;3 y"},
		{"<ul><li>one</li><li>two</li></ul>", "<ul><li>one</li><li>two</li></ul>"},
		{"tab\tsep", "tab\tsep"},
		{"a \"q\" b <x>", "a &#34;q&#34; b "},
		{"it's <x> fine", "it&#39;s  fine"},
		{"1 < 2 && 3 > 2", "1 &lt; 2 &amp;&amp; 3 &gt; 2"},
		{"<p>para</p>\n<p>two</p>", "<p>para</p>\n<p>two</p>"},
		{"<br>", "<br />"},
		{"<br/>", "<br />"},
		{"<hr />", "<hr />"},
		{"a<br>b", "a<br />b"},
		{"<font color=\"red\">r</font>", "r"},
		{"<h1>t</h1>", "<h1>t</h1>"},
		{"<table><tr><td>c</td></tr></table>", "<table><tbody><tr><td>c</td></tr></tbody></table>"},
		{"<iframe src=\"x\"></iframe>", ""},
		{"plain markdown **b** _i_ - list", "plain markdown **b** _i_ - list"},
		{"<span style=\"color:red\">s</span>", "s"},
		{"<details><summary>s</summary>d</details>", "<details><summary>s</summary>d</details>"},
		{"<p/>", "<p></p>"},
		{"a<>b", "a&lt;&gt;b"},
		{"<1>", "&lt;1&gt;"},
		{"< b >", "&lt; b &gt;"},
		{"<b >x</b >", "<b>x</b>"},
		{"</b>x", "x"},
		{"<B>X</B>", "<b>X</b>"},
	} {
		if !EquivalentText(c[0], c[1]) {
			t.Errorf("EquivalentText(%q, %q) = false, want true", c[0], c[1])
		}
	}
}

func TestEquivalentTextDetectsRealChanges(t *testing.T) {
	for _, c := range [][2]string{
		{"old", "new"},
		{"a & b", "a &amp; b"}, // no `<`: OM stores it untouched, so a difference is a real one
		{"om-<env>-consumers", "om-prod-consumers"},
		{"<b>one</b>", "<b>two</b>"},
		{"x", ""},
	} {
		if EquivalentText(c[0], c[1]) {
			t.Errorf("EquivalentText(%q, %q) = true, want false", c[0], c[1])
		}
	}
}

func TestCreateOrUpdateSkipsPatchWhenSanitized(t *testing.T) {
	var patched []map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patched = append(patched, nil)
		}
		_, _ = w.Write([]byte(`{"id":"abc","description":"om--consumers"}`))
	}))
	defer srv.Close()

	raw, err := NewClient(srv.URL, "tok").CreateOrUpdate(context.Background(), "teams",
		map[string]interface{}{"name": "x", "description": "om-<env>-consumers"})
	if err != nil {
		t.Fatal(err)
	}
	if patched != nil {
		t.Fatal("sanitized description must not trigger a patch")
	}
	if string(raw) != `{"id":"abc","description":"om--consumers"}` {
		t.Fatalf("unexpected response %s", raw)
	}
}

func TestCreateOrUpdateReportsIgnoredDisplayName(t *testing.T) {
	var patched []map[string]interface{}
	srv := botServer(t, &patched)
	defer srv.Close()

	_, err := NewClient(srv.URL, "tok").CreateOrUpdate(context.Background(), "roles",
		map[string]interface{}{"name": "x", "description": "old", "displayName": "New"})
	if err == nil || !strings.Contains(err.Error(), "bot tokens cannot edit display names") {
		t.Fatalf("expected display name error, got %v", err)
	}
	if patched != nil {
		t.Fatalf("display name must not be patched, got %v", patched)
	}
}
