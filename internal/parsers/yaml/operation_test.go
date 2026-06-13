// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/yaml"
	oaispec "github.com/go-openapi/spec"
)

// TestUnmarshalBody_RoundTrip checks the YAML → JSON →
// UnmarshalJSON pipeline used by the swagger:operation grammar bridge.
// The raw body here matches what grammar's TokenOpaqueYaml emits for
// a `---` fenced block (contents only, no fences, no `//` markers).
func TestUnmarshalBody_RoundTrip(t *testing.T) {
	body := `parameters:
  - name: limit
    in: query
    type: integer
    format: int32
responses:
  "200":
    description: OK
`
	op := new(oaispec.Operation)
	if err := yaml.UnmarshalBody(body, op.UnmarshalJSON); err != nil {
		t.Fatalf("UnmarshalBody: %v", err)
	}

	if len(op.Parameters) != 1 {
		t.Fatalf("parameters: got %d, want 1", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Name != "limit" || p.In != "query" || p.Type != "integer" || p.Format != "int32" {
		t.Errorf("parameter fields: %+v", p)
	}
	if op.Responses == nil || op.Responses.StatusCodeResponses[200].Description != "OK" {
		t.Errorf("responses: %+v", op.Responses)
	}
}

func TestUnmarshalBody_InvalidYAML(t *testing.T) {
	// Unbalanced brackets — yaml.Unmarshal will error.
	body := "parameters: [\n  - name: x"
	op := new(oaispec.Operation)
	if err := yaml.UnmarshalBody(body, op.UnmarshalJSON); err == nil {
		t.Error("expected error on malformed YAML, got nil")
	}
}

func TestUnmarshalBody_EmptyBody(t *testing.T) {
	// Empty body short-circuits before unmarshal — caller's target
	// stays untouched.
	op := new(oaispec.Operation)
	if err := yaml.UnmarshalBody("", op.UnmarshalJSON); err != nil {
		t.Errorf("empty body should not error: %v", err)
	}
}

// TestUnmarshalBody_TabIndent verifies the dedent step
// handles tab-indented godoc-style bodies (the go119 fixture style).
func TestUnmarshalBody_TabIndent(t *testing.T) {
	body := "\tparameters:\n\t  - name: limit\n\t    in: query\n\t    type: integer\n"
	op := new(oaispec.Operation)
	if err := yaml.UnmarshalBody(body, op.UnmarshalJSON); err != nil {
		t.Fatalf("UnmarshalBody: %v", err)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "limit" {
		t.Fatalf("parameters: %+v", op.Parameters)
	}
}

// TestUnmarshalListBody_Tags checks the sequence-shaped pipeline used
// by the meta `Tags:` bridge: a YAML list of tag objects (with a
// nested externalDocs mapping and a vendor extension) round-trips into
// []spec.Tag. Tab-indented like a real godoc comment body.
func TestUnmarshalListBody_Tags(t *testing.T) {
	body := "\t- name: pet\n" +
		"\t  description: Everything about your Pets\n" +
		"\t  externalDocs:\n" +
		"\t    description: Find out more\n" +
		"\t    url: http://swagger.io\n" +
		"\t- name: store\n" +
		"\t  x-display-name: Store\n"
	var tags []oaispec.Tag
	err := yaml.UnmarshalListBody(body, func(data []byte) error {
		return json.Unmarshal(data, &tags)
	})
	if err != nil {
		t.Fatalf("UnmarshalListBody: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags: got %d, want 2", len(tags))
	}
	if tags[0].Name != "pet" || tags[0].Description != "Everything about your Pets" {
		t.Errorf("tag[0]: %+v", tags[0].TagProps)
	}
	if tags[0].ExternalDocs == nil || tags[0].ExternalDocs.URL != "http://swagger.io" {
		t.Errorf("tag[0].externalDocs: %+v", tags[0].ExternalDocs)
	}
	if tags[1].Name != "store" || tags[1].Extensions["x-display-name"] != "Store" {
		t.Errorf("tag[1]: %+v / %+v", tags[1].TagProps, tags[1].Extensions)
	}
}

func TestUnmarshalListBody_EmptyBody(t *testing.T) {
	var tags []oaispec.Tag
	if err := yaml.UnmarshalListBody("", func([]byte) error { return nil }); err != nil {
		t.Errorf("empty body should not error: %v", err)
	}
	if tags != nil {
		t.Errorf("empty body should leave target untouched, got %+v", tags)
	}
}
