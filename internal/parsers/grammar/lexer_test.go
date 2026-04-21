// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"reflect"
	"testing"
)

// mkLines turns plain strings into position-carrying Lines with dummy
// but non-zero positions so tests can assert Pos if they care.
func mkLines(texts ...string) []Line {
	out := make([]Line, len(texts))
	for i, t := range texts {
		out[i] = Line{Text: t, Pos: token.Position{Filename: "t.go", Line: i + 1, Column: 1, Offset: i * 50}}
	}
	return out
}

func TestLexEmpty(t *testing.T) {
	toks := Lex(nil)
	if len(toks) != 1 || toks[0].Kind != TokenEOF {
		t.Fatalf("Lex(nil): want [EOF], got %+v", toks)
	}
}

func TestLexBlankAndYAMLFence(t *testing.T) {
	toks := Lex(mkLines("", "   ", "---", "  ---  "))
	want := []TokenKind{TokenBlank, TokenBlank, TokenYAMLFence, TokenYAMLFence, TokenEOF}
	if len(toks) != len(want) {
		t.Fatalf("want %d tokens, got %d: %+v", len(want), len(toks), toks)
	}
	for i, w := range want {
		if toks[i].Kind != w {
			t.Errorf("tok %d: got %s want %s", i, toks[i].Kind, w)
		}
	}
}

func TestLexAnnotationSimple(t *testing.T) {
	toks := Lex(mkLines("swagger:model Foo"))
	if toks[0].Kind != TokenAnnotation {
		t.Fatalf("want ANNOTATION, got %s: %+v", toks[0].Kind, toks[0])
	}
	if toks[0].Text != "model" {
		t.Errorf("name: got %q want %q", toks[0].Text, "model")
	}
	if !reflect.DeepEqual(toks[0].Args, []string{"Foo"}) {
		t.Errorf("args: got %v want [Foo]", toks[0].Args)
	}
}

func TestLexAnnotationRoute(t *testing.T) {
	toks := Lex(mkLines("swagger:route GET /pets tags listPets"))
	if toks[0].Kind != TokenAnnotation {
		t.Fatalf("want ANNOTATION, got %s", toks[0].Kind)
	}
	if toks[0].Text != "route" {
		t.Errorf("name: got %q want route", toks[0].Text)
	}
	want := []string{"GET", "/pets", "tags", "listPets"}
	if !reflect.DeepEqual(toks[0].Args, want) {
		t.Errorf("args: got %v want %v", toks[0].Args, want)
	}
}

func TestLexAnnotationMalformed(t *testing.T) {
	// "swagger:" with no name falls back to TEXT.
	toks := Lex(mkLines("swagger:"))
	if toks[0].Kind != TokenText {
		t.Errorf("want TEXT (malformed annotation), got %s", toks[0].Kind)
	}
}

func TestLexKeywordValue(t *testing.T) {
	toks := Lex(mkLines("maximum: 10"))
	tok := toks[0]
	if tok.Kind != TokenKeywordValue {
		t.Fatalf("want KEYWORD_VALUE, got %s", tok.Kind)
	}
	if tok.Text != "maximum" {
		t.Errorf("name: got %q want maximum", tok.Text)
	}
	if tok.Value != "10" {
		t.Errorf("value: got %q want %q", tok.Value, "10")
	}
	if tok.Keyword == nil || tok.Keyword.Value.Type != ValueNumber {
		t.Errorf("expected Keyword resolved to ValueNumber, got %+v", tok.Keyword)
	}
}

func TestLexKeywordCanonicalizesAlias(t *testing.T) {
	toks := Lex(mkLines("MAX: 10", "max-length: 5"))
	if toks[0].Text != "maximum" {
		t.Errorf("MAX → canonical: got %q want maximum", toks[0].Text)
	}
	if toks[1].Text != "maxLength" {
		t.Errorf("max-length → canonical: got %q want maxLength", toks[1].Text)
	}
}

func TestLexKeywordBlockHead(t *testing.T) {
	toks := Lex(mkLines("consumes:"))
	tok := toks[0]
	if tok.Kind != TokenKeywordBlockHead {
		t.Fatalf("want KEYWORD_BLOCK_HEAD, got %s", tok.Kind)
	}
	if tok.Text != "consumes" || tok.Value != "" {
		t.Errorf("unexpected token: %+v", tok)
	}
}

func TestLexItemsPrefix(t *testing.T) {
	cases := []struct {
		in        string
		wantName  string
		wantDepth int
	}{
		{"maximum: 5", "maximum", 0},
		{"items.maximum: 5", "maximum", 1},
		{"items.items.maximum: 5", "maximum", 2},
		{"Items.Items.maximum: 5", "maximum", 2},
		{"items items maximum: 5", "maximum", 2}, //nolint:dupword // space-separated items prefix is valid per rxItemsPrefixFmt
		{"items.items.items.minLength: 1", "minLength", 3},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			toks := Lex(mkLines(tc.in))
			tok := toks[0]
			if tok.Kind != TokenKeywordValue {
				t.Fatalf("want KEYWORD_VALUE, got %s", tok.Kind)
			}
			if tok.Text != tc.wantName {
				t.Errorf("name: got %q want %q", tok.Text, tc.wantName)
			}
			if tok.ItemsDepth != tc.wantDepth {
				t.Errorf("depth: got %d want %d", tok.ItemsDepth, tc.wantDepth)
			}
		})
	}
}

func TestLexItemsDoesNotOvereat(t *testing.T) {
	// "maxItems" must stay a single word — the "items" suffix inside
	// it is not a prefix marker.
	toks := Lex(mkLines("maxItems: 3"))
	if toks[0].Kind != TokenKeywordValue || toks[0].Text != "maxItems" {
		t.Errorf("maxItems must remain a single keyword, got %+v", toks[0])
	}
	if toks[0].ItemsDepth != 0 {
		t.Errorf("maxItems must have depth 0, got %d", toks[0].ItemsDepth)
	}
}

func TestLexUnknownKeywordIsText(t *testing.T) {
	toks := Lex(mkLines("not-a-keyword: hello"))
	if toks[0].Kind != TokenText {
		t.Errorf("want TEXT, got %s", toks[0].Kind)
	}
}

func TestLexPlainText(t *testing.T) {
	toks := Lex(mkLines("This is a description line."))
	if toks[0].Kind != TokenText {
		t.Errorf("want TEXT, got %s", toks[0].Kind)
	}
	if toks[0].Text != "This is a description line." {
		t.Errorf("text: got %q", toks[0].Text)
	}
}

func TestLexPreservesPositions(t *testing.T) {
	toks := Lex(mkLines("swagger:model Foo", "maximum: 5"))
	if toks[0].Pos.Line != 1 {
		t.Errorf("line 1 token: got Pos.Line %d want 1", toks[0].Pos.Line)
	}
	if toks[1].Pos.Line != 2 {
		t.Errorf("line 2 token: got Pos.Line %d want 2", toks[1].Pos.Line)
	}
}

func TestLexItemsPrefixAdvancesPos(t *testing.T) {
	toks := Lex(mkLines("items.maximum: 5"))
	tok := toks[0]
	// "items." is 6 bytes; Pos must advance past it to point at 'm'.
	// Column started at 1 → expect 7.
	if tok.Pos.Column != 7 {
		t.Errorf("items.maximum Pos.Column: got %d want 7", tok.Pos.Column)
	}
}

func TestLexEOFIsAlwaysLast(t *testing.T) {
	toks := Lex(mkLines("swagger:model Foo", "", "maximum: 5"))
	if toks[len(toks)-1].Kind != TokenEOF {
		t.Errorf("last token must be EOF, got %s", toks[len(toks)-1].Kind)
	}
}
