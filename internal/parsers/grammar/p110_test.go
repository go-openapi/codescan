// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"strings"
	"testing"
)

// Tests for P1.10 catch-up items: verbatim YAML body contract and the
// preprocessor's `-` stripping behavior.

// --- verbatim YAML body ---

func TestP110YAMLBodyPreservesIndentation(t *testing.T) {
	// The 2-space indent on "200:" must survive into RawYAML.Text —
	// this is what internal/parsers/yaml/ needs to parse YAML cleanly.
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// responses:
//   200: successResponse
//   404: notFound
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	count := 0
	for y := range b.YAMLBlocks() {
		count++
		lines := splitLines(y.Text)
		if len(lines) < 3 {
			t.Fatalf("want at least 3 body lines, got %d: %q", len(lines), y.Text)
		}
		// Line 0: " responses:" — the godoc convention space after
		// `// ` is now preserved in Line.Raw so YAML bodies with tab
		// indentation (e.g. go119 fixture) round-trip faithfully.
		if lines[0] != " responses:" {
			t.Errorf("line 0: got %q want %q", lines[0], " responses:")
		}
		// Line 1: `   200: successResponse` — godoc space + source
		// 2-space indent preserved.
		if lines[1] != "   200: successResponse" {
			t.Errorf("line 1: got %q want %q", lines[1], "   200: successResponse")
		}
		if lines[2] != "   404: notFound" {
			t.Errorf("line 2: got %q want %q", lines[2], "   404: notFound")
		}
	}
	if count != 1 {
		t.Fatalf("want 1 YAML block, got %d", count)
	}
}

func TestP110YAMLBodyRaw_TokenKind(t *testing.T) {
	// Direct lexer check: interior content between fences becomes
	// TokenRawLine, not TokenText or TokenKeywordValue.
	mk := func(n int, text, raw string) Line {
		return Line{Text: text, Raw: raw, Pos: token.Position{Line: n, Column: 1}}
	}
	lines := []Line{
		mk(1, "---", "---"),
		mk(2, "responses:", "responses:"),
		mk(3, "200: ok", "  200: ok"),
		mk(4, "---", "---"),
	}
	toks := Lex(lines)
	want := []TokenKind{TokenYAMLFence, TokenRawLine, TokenRawLine, TokenYAMLFence, TokenEOF}
	if len(toks) != len(want) {
		t.Fatalf("want %d tokens, got %d: %+v", len(want), len(toks), toks)
	}
	for i, w := range want {
		if toks[i].Kind != w {
			t.Errorf("tok %d: got %s want %s", i, toks[i].Kind, w)
		}
	}
	// TokenRawLine.Text must be Line.Raw (indent preserved).
	if toks[2].Text != "  200: ok" {
		t.Errorf("TokenRawLine.Text: got %q want %q", toks[2].Text, "  200: ok")
	}
}

func TestP110BlockCommentYAMLBody(t *testing.T) {
	// YAML inside a /* ... */ block: the block-continuation " * "
	// stripping must still produce clean YAML with preserved indent.
	src := "package p\n\n/*\nswagger:operation GET /pets listPets\n\n---\nresponses:\n  200: ok\n---\n*/\nfunc F(){}\n"
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	for y := range b.YAMLBlocks() {
		lines := splitLines(y.Text)
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), y.Text)
		}
		if lines[0] != "responses:" {
			t.Errorf("line 0: got %q want %q", lines[0], "responses:")
		}
		if lines[1] != "  200: ok" {
			t.Errorf("line 1: got %q want %q (indent must survive block continuation strip)", lines[1], "  200: ok")
		}
	}
}

// --- preprocessor `-` stripping parity lock-in ---

func TestP110DashNotStrippedInProse(t *testing.T) {
	// P1.10 decision: the preprocessor keeps leading `-` in content
	// (v2 divergence from v1, which silently stripped them). This
	// preserves bullet-list semantics and keeps the `---` YAML fence
	// marker detectable without special-casing.
	//
	// Asserted at the Block level: a bullet-list paragraph feeds
	// cleanly into Title/Description, with the `-` intact. (Bullets
	// within a single paragraph are merged per godoc convention.)
	src := `package p

// Summary line.
//
// - first item
// - second item
//
// swagger:model Foo
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if b.Title() != "Summary line." {
		t.Errorf("Title: got %q want %q", b.Title(), "Summary line.")
	}
	if !strings.Contains(b.Description(), "- first item") {
		t.Errorf("Description should preserve bullet `-`: got %q", b.Description())
	}
	if !strings.Contains(b.Description(), "- second item") {
		t.Errorf("Description should preserve bullet `-`: got %q", b.Description())
	}
}

func TestP110DashPreservedOnlySurvivesVerbatimInText(t *testing.T) {
	// Low-level confirmation at the preprocessor: a single `- foo` line
	// produces Line.Text with the `-` intact.
	src := "package p\n\n// - not stripped\ntype Foo int\n"
	cg, fset := parseCommentGroup(t, src)
	lines := Preprocess(cg, fset)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	if lines[0].Text != "- not stripped" {
		t.Errorf("Text: got %q want %q", lines[0].Text, "- not stripped")
	}
}
