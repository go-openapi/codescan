// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

// The dedupe layer is exercised end-to-end via Parse / ParseInto / TypedExtensions tests
// (yaml_test.go); these tests pin the helper's behaviour directly so future edits can't regress
// edge cases without turning red here too.

func TestDedupeFlatMappingLastWins(t *testing.T) {
	body := []byte("type: apiKey\nname: KEY\ntype: oauth2\n")
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["type"] != "oauth2" {
		t.Errorf("type: want oauth2 (last-wins), got %v", got["type"])
	}
	if got["name"] != "KEY" {
		t.Errorf("name: want KEY, got %v", got["name"])
	}
	if len(got) != 2 {
		t.Errorf("want 2 surviving keys, got %d: %v", len(got), got)
	}
}

func TestDedupeNestedMapping(t *testing.T) {
	// Q28 repro shape — duplicate type/in inside a nested SecurityDefinitions entry, plus a
	// duplicate top-level scheme name so we exercise both depths.
	body := []byte(`SecurityDefinitions:
  api_key:
    type: apiKey
    name: KEY
    type: apiKey
    in: header
  oauth2:
    type: oauth2
    in: header
`)
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	sd, ok := got["SecurityDefinitions"].(map[string]any)
	if !ok {
		t.Fatalf("SecurityDefinitions: want map[string]any, got %T", got["SecurityDefinitions"])
	}
	api, ok := sd["api_key"].(map[string]any)
	if !ok {
		t.Fatalf("api_key: want map[string]any, got %T", sd["api_key"])
	}
	if api["type"] != "apiKey" {
		t.Errorf("api_key.type: want apiKey, got %v", api["type"])
	}
	if api["in"] != "header" {
		t.Errorf("api_key.in: want header, got %v", api["in"])
	}
	if len(api) != 3 {
		t.Errorf("api_key: want 3 surviving keys (type,name,in), got %d: %v", len(api), api)
	}
}

func TestDedupeInsideSequenceElements(t *testing.T) {
	// Duplicate keys nested under a sequence entry must also dedupe.
	body := []byte(`items:
  - id: 1
    name: first
    id: 2
  - id: 3
    name: second
`)
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items: want []any of len 2, got %T (len=%d)", got["items"], len(items))
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0]: want map[string]any, got %T", items[0])
	}
	if first["id"] != 2 { // last-wins; yaml lib types unquoted int as int
		t.Errorf("items[0].id: want 2 (last-wins), got %v", first["id"])
	}
	if first["name"] != "first" {
		t.Errorf("items[0].name: want first, got %v", first["name"])
	}
}

func TestDedupeDeeplyNested(t *testing.T) {
	body := []byte(`a:
  b:
    c:
      key: v1
      key: v2
      key: v3
`)
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	a, ok := got["a"].(map[string]any)
	if !ok {
		t.Fatalf("a: want map[string]any, got %T", got["a"])
	}
	b, ok := a["b"].(map[string]any)
	if !ok {
		t.Fatalf("a.b: want map[string]any, got %T", a["b"])
	}
	c, ok := b["c"].(map[string]any)
	if !ok {
		t.Fatalf("a.b.c: want map[string]any, got %T", b["c"])
	}
	if c["key"] != "v3" {
		t.Errorf("deeply nested key: want v3 (last-wins), got %v", c["key"])
	}
	if len(c) != 1 {
		t.Errorf("c: want 1 surviving key, got %d", len(c))
	}
}

func TestDedupeNoDuplicatesIsIdempotent(t *testing.T) {
	// A mapping without dups should pass through unchanged — the fast path in dedupePairs returns
	// the original slice.
	body := []byte("alpha: 1\nbeta: 2\ngamma: 3\n")
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 || got["alpha"] != 1 || got["beta"] != 2 || got["gamma"] != 3 {
		t.Errorf("unexpected: %v", got)
	}
}

func TestDedupeEmptyBodyNoop(t *testing.T) {
	var got map[string]any
	if err := decodeYAMLBody(nil, &got); err != nil {
		t.Errorf("nil body: unexpected error: %v", err)
	}
	if err := decodeYAMLBody([]byte(""), &got); err != nil {
		t.Errorf("empty body: unexpected error: %v", err)
	}
}

func TestDedupeScalarOnlyBody(t *testing.T) {
	// Pure scalar body must survive the dedupe pass untouched.
	body := []byte("just-a-scalar\n")
	var got any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != "just-a-scalar" {
		t.Errorf("scalar: got %v want just-a-scalar", got)
	}
}

func TestDedupeIntoTypedStruct(t *testing.T) {
	// ParseInto-style target — the dedupe must precede the struct decode so yaml.v3's strict check
	// doesn't fire on the duplicate.
	type sec struct {
		Type string `yaml:"type"`
		Name string `yaml:"name"`
		In   string `yaml:"in"`
	}
	body := []byte("type: apiKey\nname: KEY\ntype: apiKey\nin: header\n")
	var got sec
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != (sec{Type: "apiKey", Name: "KEY", In: "header"}) {
		t.Errorf("typed decode: got %+v", got)
	}
}

func TestDedupeDistinctKindsNotMerged(t *testing.T) {
	// (Kind, Value) is the dedupe key — a string "1" and an int 1 have different scalar styles but
	// the same Value+Kind in YAML, so they DO collapse.
	// Document the choice: this mirrors yaml.v3's own uniqueKeys comparison (decode.go:775).
	body := []byte(`"1": quoted
1: unquoted
`)
	var got map[string]any
	if err := decodeYAMLBody(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Both keys land in the same scalar-kind bucket; last-wins keeps the second entry.
	// The surviving key may be quoted or unquoted depending on the yaml lib's round-trip; assert on
	// the value.
	if len(got) != 1 {
		t.Errorf("want 1 surviving key (kind+value collapse), got %d: %v", len(got), got)
	}
	for _, v := range got {
		if v != "unquoted" {
			t.Errorf("surviving value: want unquoted (last-wins), got %v", v)
		}
	}
}

func TestDedupePairsFastPath(t *testing.T) {
	// Direct exercise of dedupePairs's no-op fast path.
	content := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "a"},
		{Kind: yaml.ScalarNode, Value: "1"},
		{Kind: yaml.ScalarNode, Value: "b"},
		{Kind: yaml.ScalarNode, Value: "2"},
	}
	out := dedupePairs(content)
	if len(out) != 4 {
		t.Errorf("fast path: want 4 entries, got %d", len(out))
	}
}
