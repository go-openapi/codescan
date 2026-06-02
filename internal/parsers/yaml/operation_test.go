// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
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
