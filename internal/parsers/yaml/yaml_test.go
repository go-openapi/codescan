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
