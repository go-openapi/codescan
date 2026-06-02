// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/yaml"
)

func TestParseEmpty(t *testing.T) {
	v, err := yaml.Parse("")
	if err != nil {
		t.Fatalf("empty body: unexpected error: %v", err)
	}
	if v != nil {
		t.Errorf("empty body: want nil, got %v", v)
	}
}

func TestParseFlatMap(t *testing.T) {
	// Note: go.yaml.in/yaml/v3 returns map[string]any for
	// string-keyed maps and auto-types scalars (unquoted "1.0"
	// becomes float64). Quote the value to keep it as a string.
	body := "name: Foo\nversion: \"1.0\"\n"
	v, err := yaml.Parse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want map[string]any, got %T: %v", v, v)
	}
	if m["name"] != "Foo" {
		t.Errorf("name: got %v want Foo", m["name"])
	}
	if m["version"] != "1.0" {
		t.Errorf("version: got %v", m["version"])
	}
}

func TestParseNestedStructure(t *testing.T) {
	// Representative of an operation body's responses mapping.
	// Numeric keys like `200` arrive as int keys; the outer map
	// becomes map[any]any because not all keys are strings.
	body := "responses:\n  200:\n    description: ok\n  404:\n    description: not found\n"
	v, err := yaml.Parse(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	top, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("want top-level map[string]any, got %T", v)
	}
	// The responses map has integer keys (200, 404), so the
	// YAML library returns map[any]any (keys include non-strings).
	resp, ok := top["responses"].(map[any]any)
	if !ok {
		t.Fatalf("responses: want map[any]any (int keys), got %T", top["responses"])
	}
	if len(resp) != 2 {
		t.Errorf("responses: want 2 entries, got %d", len(resp))
	}
}

func TestParseInvalidYAML(t *testing.T) {
	// Bad indentation / stray colon.
	body := "key: [unclosed\n"
	_, err := yaml.Parse(body)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.HasPrefix(err.Error(), "yaml:") {
		t.Errorf("error should be wrapped with 'yaml:' prefix: got %q", err.Error())
	}
}

func TestParseIntoStruct(t *testing.T) {
	type operation struct {
		Method string `yaml:"method"`
		Path   string `yaml:"path"`
	}
	body := "method: GET\npath: /pets\n"
	var op operation
	if err := yaml.ParseInto(body, &op); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if op.Method != http.MethodGet || op.Path != "/pets" {
		t.Errorf("unmarshalled struct: %+v", op)
	}
}

func TestParseIntoEmpty(t *testing.T) {
	// Empty body is a no-op (dst left at zero value).
	type op struct{ Method string }
	var v op
	if err := yaml.ParseInto("", &v); err != nil {
		t.Errorf("empty body: unexpected error: %v", err)
	}
	if v.Method != "" {
		t.Errorf("dst should be untouched, got %+v", v)
	}
}

func TestTypedExtensionsEmpty(t *testing.T) {
	m, err := yaml.TypedExtensions("")
	if err != nil {
		t.Fatalf("empty body: unexpected error: %v", err)
	}
	if m != nil {
		t.Errorf("empty body: want nil map, got %v", m)
	}
}

func TestTypedExtensionsFlatScalars(t *testing.T) {
	body := "x-tag: foo\nx-priority: 5\nx-enabled: true\n"
	m, err := yaml.TypedExtensions(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m["x-tag"] != "foo" {
		t.Errorf("x-tag: got %v want foo", m["x-tag"])
	}
	// JSON normalisation yields float64 for numeric scalars.
	if got, ok := m["x-priority"].(float64); !ok || got != 5 {
		t.Errorf("x-priority: got %v (%T) want float64(5)", m["x-priority"], m["x-priority"])
	}
	if m["x-enabled"] != true {
		t.Errorf("x-enabled: got %v want true", m["x-enabled"])
	}
}

func TestTypedExtensionsNestedYAML(t *testing.T) {
	// The case the schema builder's prior applyExtensionsRawBlock
	// existed for: nested map / list values must arrive as typed
	// map[string]any and []any, not yaml.v3's map[any]any.
	body := "x-config:\n  enabled: true\n  threshold: 0.5\n  tags: [a, b, c]\n"
	m, err := yaml.TypedExtensions(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cfg, ok := m["x-config"].(map[string]any)
	if !ok {
		t.Fatalf("x-config: want map[string]any, got %T", m["x-config"])
	}
	if cfg["enabled"] != true {
		t.Errorf("x-config.enabled: got %v want true", cfg["enabled"])
	}
	if cfg["threshold"] != 0.5 {
		t.Errorf("x-config.threshold: got %v want 0.5", cfg["threshold"])
	}
	tags, ok := cfg["tags"].([]any)
	if !ok || len(tags) != 3 {
		t.Fatalf("x-config.tags: want []any{a,b,c}, got %v (%T)", cfg["tags"], cfg["tags"])
	}
}

func TestTypedExtensionsNoFilter(t *testing.T) {
	// The service does NOT drop non-x-* keys — the caller decides.
	body := "x-good: 1\nnot-an-extension: 2\n"
	m, err := yaml.TypedExtensions(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := m["not-an-extension"]; !ok {
		t.Errorf("non-x-* key should be present (caller filters); got map %v", m)
	}
}

func TestTypedExtensionsInvalidYAML(t *testing.T) {
	body := "x-broken: [unclosed\n"
	_, err := yaml.TypedExtensions(body)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
