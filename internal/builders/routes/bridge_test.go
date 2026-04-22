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

	want := []string{"http", "https", "ws"}
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
// fix that lets a Parameters or Responses body contain keywords
// whose natural context is Param/Schema/Items (not Route/Operation/
// Meta): they're absorbed as body text rather than terminating the
// multi-line block. Without this, `default:`, `in:`, `required:`,
// `max:` inside a Parameters body would prematurely stop the
// collection and produce a malformed spec.
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
		if p.Keyword.Name == "parameters" {
			params = p
			break
		}
	}
	if params.Keyword.Name != "parameters" {
		t.Fatalf("parameters property not found")
	}

	// Body must retain every source line, absorbed verbatim (names in
	// source form: `max` not the canonical `maximum`).
	var sb strings.Builder
	for _, l := range params.Body {
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	joined := sb.String()
	for _, expected := range []string{
		"+ name:     someNumber",
		"in: path",
		"required: true",
		"max: 20",
		"default: 15",
		"+ name:     flag",
	} {
		if !contains(joined, expected) {
			t.Errorf("Body missing %q in:\n%s", expected, joined)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
