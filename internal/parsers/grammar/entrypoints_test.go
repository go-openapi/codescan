// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The rest of the suite reaches past the exported entry points — parseString goes straight to
// ParseText, parseAllString to parseAllTokens — so the doors a caller actually uses were never
// opened, and neither was the block-comment arm of the preprocessor behind them.

// docGroup returns the doc comment of the first doc-bearing declaration in src, with its file set.
func docGroup(t *testing.T, src string) (*ast.CommentGroup, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "fake.go", src, goparser.ParseComments)
	require.NoError(t, err)

	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			if decl.Doc != nil {
				return decl.Doc, fset
			}
		case *ast.FuncDecl:
			if decl.Doc != nil {
				return decl.Doc, fset
			}
		}
	}
	t.Fatalf("no doc-bearing decl found in fixture")

	return nil, nil
}

// A /* */ doc comment is ordinary Go, and the preprocessor has a whole arm for it: the marker is
// only on the first line, continuation lines carry their own column, and the closing */ is not part
// of the text.
func TestPreprocess_BlockComment(t *testing.T) {
	cg, fset := docGroup(t, `package p

/*
Widget is a thing.

swagger:model gizmo
*/
type Widget struct{}
`)

	lines := Preprocess(cg, fset)
	require.NotEmpty(t, lines)

	texts := make([]string, 0, len(lines))
	for _, l := range lines {
		texts = append(texts, l.Text)
	}
	assert.Contains(t, texts, "Widget is a thing.")
	assert.Contains(t, texts, "swagger:model gizmo")
	for _, l := range lines {
		assert.NotContains(t, l.Text, "*/", "the closing marker is not part of any line")
	}

	// Every line keeps a position of its own, so a diagnostic can point at the offending line
	// rather than at the top of the block.
	assert.Positive(t, lines[0].Pos.Line)
	last := lines[len(lines)-1]
	assert.Greater(t, last.Pos.Line, lines[0].Pos.Line, "continuation lines advance")

	b := Parse(cg, fset)
	mb, ok := b.(*ModelBlock)
	require.True(t, ok, "expected *ModelBlock, got %T", b)
	assert.Equal(t, "gizmo", mb.Name, "an annotation in a block comment parses like any other")
}

// A single-line /* */ comment has no continuation at all — the whole body is one line.
func TestPreprocess_SingleLineBlockComment(t *testing.T) {
	cg, fset := docGroup(t, `package p

/* swagger:model gizmo */
type Widget struct{}
`)

	b := Parse(cg, fset)

	mb, ok := b.(*ModelBlock)
	require.True(t, ok, "expected *ModelBlock, got %T", b)
	assert.Equal(t, "gizmo", mb.Name)
}

// ParseAll slices a comment group carrying several annotations into one block each, in source
// order — the shape the scanner needs when a declaration is both a model and something else.
func TestParseAll(t *testing.T) {
	t.Run("several annotations", func(t *testing.T) {
		cg, fset := docGroup(t, `package p

// Widget is a thing.
//
// swagger:model gizmo
// swagger:name widget
type Widget struct{}
`)

		blocks := ParseAll(cg, fset)

		require.Len(t, blocks, 2)
		mb, ok := blocks[0].(*ModelBlock)
		require.True(t, ok, "expected *ModelBlock, got %T", blocks[0])
		assert.Equal(t, "gizmo", mb.Name)
		nb, ok := blocks[1].(*NameBlock)
		require.True(t, ok, "expected *NameBlock, got %T", blocks[1])
		assert.Equal(t, "widget", nb.Name)
	})

	// One annotation is the same thing Parse produces, wrapped — so callers need no special case.
	t.Run("a single annotation", func(t *testing.T) {
		cg, fset := docGroup(t, `package p

// swagger:model gizmo
type Widget struct{}
`)

		blocks := ParseAll(cg, fset)

		require.Len(t, blocks, 1)
		assert.Equal(t, AnnModel, blocks[0].AnnotationKind())
	})

	t.Run("prose with no annotation at all", func(t *testing.T) {
		cg, fset := docGroup(t, `package p

// Widget is just documented.
type Widget struct{}
`)

		blocks := ParseAll(cg, fset)

		require.Len(t, blocks, 1, "an unbound block still comes back, so the prose is not lost")
		assert.Equal(t, "Widget is just documented.", blocks[0].Title())
	})
}

// ParseTokens is the seam for a caller that already holds a token stream — an LSP re-lexing one
// edited line rather than the whole file.
func TestParseTokens(t *testing.T) {
	pos := token.Position{Filename: "test.go", Line: 1, Column: 1}
	tokens := Lex(preprocessText("swagger:model gizmo", pos))

	b := ParseTokens(tokens)

	mb, ok := b.(*ModelBlock)
	require.True(t, ok, "expected *ModelBlock, got %T", b)
	assert.Equal(t, "gizmo", mb.Name)
}

// A sink sees diagnostics as they are raised, rather than only after the block comes back — which
// is what lets a long-running caller report them incrementally.
func TestWithDiagnosticSink(t *testing.T) {
	cg, fset := docGroup(t, `package p

// swagger:name
type Widget struct{}
`)

	var streamed []Diagnostic
	b := NewParser(fset, WithDiagnosticSink(func(d Diagnostic) {
		streamed = append(streamed, d)
	})).Parse(cg)

	require.NotEmpty(t, b.Diagnostics(), "precondition: a missing required arg is a diagnostic")
	require.NotEmpty(t, streamed, "the sink saw it too")
	assert.Equal(t, b.Diagnostics()[0].Code, streamed[0].Code)
}
