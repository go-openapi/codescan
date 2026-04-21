// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"testing"
)

// P3.1: NewParser returns a Parser interface bound to a FileSet;
// three methods (Parse, ParseText, ParseAs) cover builder and LSP
// usage.

func TestParserInterfaceParse(t *testing.T) {
	src := `package p

// swagger:model Foo
// maximum: 10
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	p := NewParser(fset)

	b := p.Parse(cg)
	if _, ok := b.(*ModelBlock); !ok {
		t.Fatalf("want *ModelBlock, got %T", b)
	}

	found := false
	for prop := range b.Properties() {
		if prop.Keyword.Name == fixtureValidationKw {
			found = true
		}
	}
	if !found {
		t.Error("expected maximum property on parsed block")
	}
}

func TestParserInterfaceParseText(t *testing.T) {
	// Raw content — no Go comment markers.
	text := "swagger:model Bar\nmaximum: 5\n"
	p := NewParser(token.NewFileSet())

	pos := token.Position{Filename: "x.go", Line: 10, Column: 1}
	b := p.ParseText(text, pos)

	mb, ok := b.(*ModelBlock)
	if !ok {
		t.Fatalf("want *ModelBlock, got %T", b)
	}
	if mb.Name != "Bar" {
		t.Errorf("Name: got %q want Bar", mb.Name)
	}
	// Pos of the annotation should reflect the passed-in base.
	if mb.Pos().Line != 10 {
		t.Errorf("Pos.Line: got %d want 10", mb.Pos().Line)
	}
}

func TestParserInterfaceParseAs(t *testing.T) {
	// LSP scenario: the user is editing properties; no annotation
	// line is present. ParseAs forces dispatch under the given kind.
	text := "maximum: 10\nminimum: 0\n"
	p := NewParser(token.NewFileSet())

	b := p.ParseAs(AnnModel, text, token.Position{Line: 1})

	if _, ok := b.(*ModelBlock); !ok {
		t.Fatalf("want *ModelBlock (forced), got %T", b)
	}
	var names []string
	for prop := range b.Properties() {
		names = append(names, prop.Keyword.Name)
	}
	if len(names) != 2 {
		t.Errorf("want 2 properties (maximum, minimum), got %d: %v", len(names), names)
	}
}

func TestParserInterfaceSatisfiedByImpl(t *testing.T) {
	// Compile-time assertion that *DefaultParser implements Parser.
	var _ Parser = (*DefaultParser)(nil)
	_ = t
}

func TestWithDiagnosticSinkStreams(t *testing.T) {
	// WithDiagnosticSink delivers each diagnostic to the callback in
	// addition to accumulating it on the returned Block.
	src := `package p

// swagger:model Foo
// in: query
// maximum: notanumber
type Foo int
`
	cg, fset := parseCommentGroup(t, src)

	var streamed []Diagnostic
	p := NewParser(fset, WithDiagnosticSink(func(d Diagnostic) {
		streamed = append(streamed, d)
	}))

	b := p.Parse(cg)

	// Block accumulation still populated.
	if len(b.Diagnostics()) == 0 {
		t.Fatal("Block should still accumulate diagnostics")
	}
	// Sink received every diagnostic.
	if len(streamed) != len(b.Diagnostics()) {
		t.Errorf("sink got %d, block got %d — must match",
			len(streamed), len(b.Diagnostics()))
	}
	// At least one diagnostic of each expected code (in: illegal +
	// maximum not-a-number).
	codes := map[Code]bool{}
	for _, d := range streamed {
		codes[d.Code] = true
	}
	if !codes[CodeContextInvalid] {
		t.Errorf("expected CodeContextInvalid in stream, got %+v", codes)
	}
	if !codes[CodeInvalidNumber] {
		t.Errorf("expected CodeInvalidNumber in stream, got %+v", codes)
	}
}

func TestWithDiagnosticSinkNilByDefault(t *testing.T) {
	// No options → sink is nil → behavior matches pre-P3.2.
	src := `package p

// swagger:model Foo
// in: query
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	p := NewParser(fset)

	b := p.Parse(cg)
	// Should still have the context-invalid diagnostic on the block.
	found := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeContextInvalid {
			found = true
		}
	}
	if !found {
		t.Error("no options path should still accumulate diagnostics")
	}
}

func TestPackageLevelParseStillWorks(t *testing.T) {
	// Backward-compat: the original top-level Parse(cg, fset) is a
	// thin wrapper around NewParser(fset).Parse(cg).
	src := `package p

// swagger:model Foo
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	if _, ok := b.(*ModelBlock); !ok {
		t.Fatalf("convenience wrapper broken: got %T", b)
	}
}
