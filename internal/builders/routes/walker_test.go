// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// parseRouteBody parses the given body as a swagger:route block using
// grammar's ParseAs hook — the lexer prepends a `swagger:route`
// annotation header so the keywords' CtxRoute context applies.
//
//nolint:ireturn // grammar.Block is the package's polymorphic return.
func parseRouteBody(t *testing.T, body string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseAs(grammar.AnnRoute, body, token.Position{Line: 1})
}

func TestDispatchRouteSchemes(t *testing.T) {
	var b Builder
	op := &oaispec.Operation{}

	block := parseRouteBody(t, "schemes: http, https, ws")
	for prop := range block.Properties() {
		if err := b.dispatchRouteKeyword(prop, op); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
	}

	want := []string{"http", "https", "ws"} //nolint:goconst // test fixture literals; consts would hurt readability.
	if len(op.Schemes) != len(want) {
		t.Fatalf("Schemes len: got %d, want %d", len(op.Schemes), len(want))
	}
	for i, s := range want {
		if op.Schemes[i] != s {
			t.Errorf("Schemes[%d]: got %q, want %q", i, op.Schemes[i], s)
		}
	}
}

func TestDispatchRouteKeywordDeprecated(t *testing.T) {
	var b Builder
	op := &oaispec.Operation{}

	block := parseRouteBody(t, "deprecated: true")
	for prop := range block.Properties() {
		if err := b.dispatchRouteKeyword(prop, op); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
	}

	if !op.Deprecated {
		t.Errorf("Deprecated: want true")
	}
}

// TestRawBlockAbsorbsSubContextKeywords verifies the grammar-level
// behaviour that lets a Parameters or Responses body contain lines
// whose first word reads as a keyword from a sub-context
// (Param / Schema / Items): they're absorbed as body text rather
// than terminating the multi-line block. Without this, `default:`,
// `in:`, `required:`, `max:` inside a Parameters body would
// prematurely stop the collection and produce a malformed spec.
func TestRawBlockAbsorbsSubContextKeywords(t *testing.T) {
	body := `Parameters:
+ name:     someNumber
  in:       path
  required: true
  type:     number
  max:      20
  min:      10
  default:  15
+ name:     flag
  in:       query
  type:     boolean
`
	block := parseRouteBody(t, body)

	var params grammar.Property
	for p := range block.Properties() {
		if p.Keyword.Name == grammar.KwParameters {
			params = p
			break
		}
	}
	if params.Keyword.Name != grammar.KwParameters {
		t.Fatalf("parameters property not found")
	}

	// Body must retain every source line, absorbed verbatim. We don't
	// pin the exact whitespace — the lexer is free to keep or
	// normalise inter-token spacing — but every value must survive
	// the absorption.
	for _, expected := range []string{
		"someNumber",
		"path",
		"required",
		"max",
		"default",
		"flag",
		"query",
		"boolean",
	} {
		if !strings.Contains(params.Body, expected) {
			t.Errorf("Body missing %q in:\n%s", expected, params.Body)
		}
	}
}
