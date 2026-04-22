// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"testing"

	oaispec "github.com/go-openapi/spec"
)

// TestUnmarshalOpYAMLRoundTrip verifies the yaml → JSON → UnmarshalJSON
// pipeline the grammar bridge uses for the operation body. The raw
// body here matches what grammar's collectYAMLBody emits for a
// `---` fenced block (contents only, no fences, no `//` markers).
func TestUnmarshalOpYAMLRoundTrip(t *testing.T) {
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
	if err := unmarshalOpYAML(body, op.UnmarshalJSON); err != nil {
		t.Fatalf("unmarshalOpYAML: %v", err)
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

func TestUnmarshalOpYAMLInvalidYAML(t *testing.T) {
	// Unbalanced brackets — yaml.Unmarshal will error.
	body := "parameters: [\n  - name: x"
	op := new(oaispec.Operation)
	if err := unmarshalOpYAML(body, op.UnmarshalJSON); err == nil {
		t.Error("expected error on malformed YAML, got nil")
	}
}

func TestUnmarshalOpYAMLEmptyBody(t *testing.T) {
	// Empty body — yaml.Unmarshal into map[any]any succeeds with
	// zero keys; fmts.YAMLToJSON produces `{}`; op.UnmarshalJSON
	// leaves the op untouched.
	op := new(oaispec.Operation)
	if err := unmarshalOpYAML("", op.UnmarshalJSON); err != nil {
		t.Errorf("empty body should not error: %v", err)
	}
}
