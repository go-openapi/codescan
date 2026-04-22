// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const wantModelFoo = "swagger:model Foo"

// parseSource is a test helper that parses a Go source file and returns
// the comment group attached to its first top-level declaration, plus
// the FileSet used during parsing.
func parseSource(t *testing.T, src string) (*ast.CommentGroup, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Decls) == 0 {
		t.Fatal("no decls in test source")
	}
	switch d := f.Decls[0].(type) {
	case *ast.GenDecl:
		if d.Doc != nil {
			return d.Doc, fset
		}
	case *ast.FuncDecl:
		if d.Doc != nil {
			return d.Doc, fset
		}
	}
	t.Fatal("decl has no doc comment")
	return nil, nil
}

func TestPreprocessNil(t *testing.T) {
	if got := Preprocess(nil, token.NewFileSet()); got != nil {
		t.Errorf("nil CommentGroup: want nil, got %v", got)
	}
	cg := &ast.CommentGroup{}
	if got := Preprocess(cg, nil); got != nil {
		t.Errorf("nil FileSet: want nil, got %v", got)
	}
}

func TestPreprocessSingleLineComment(t *testing.T) {
	cg, fset := parseSource(t, "package p\n\n// swagger:model Foo\ntype Foo struct{}\n")
	lines := Preprocess(cg, fset)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != wantModelFoo {
		t.Errorf("text: got %q want %q", lines[0].Text, wantModelFoo)
	}
	if lines[0].Pos.Line != 3 {
		t.Errorf("line: got %d want 3", lines[0].Pos.Line)
	}
}

func TestPreprocessMultipleLineComments(t *testing.T) {
	src := `package p

// swagger:model Foo
// maximum: 10
// minimum: 0
type Foo int
`
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	want := []string{wantModelFoo, "maximum: 10", "minimum: 0"}
	if len(lines) != len(want) {
		t.Fatalf("want %d lines, got %d", len(want), len(lines))
	}
	for i, w := range want {
		if lines[i].Text != w {
			t.Errorf("line %d text: got %q want %q", i, lines[i].Text, w)
		}
		if lines[i].Pos.Line != 3+i {
			t.Errorf("line %d: pos.Line = %d want %d", i, lines[i].Pos.Line, 3+i)
		}
	}
}

func TestPreprocessBlockComment(t *testing.T) {
	src := `package p

/*
 * swagger:model Foo
 * maximum: 10
 */
type Foo int
`
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	// Expect 4 lines: empty first, two content, empty last.
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d: %+v", len(lines), lines)
	}
	if lines[1].Text != wantModelFoo {
		t.Errorf("line 1: got %q want %q", lines[1].Text, wantModelFoo)
	}
	if lines[2].Text != "maximum: 10" {
		t.Errorf("line 2: got %q want %q", lines[2].Text, "maximum: 10")
	}
	// Positions should increment.
	if lines[1].Pos.Line != lines[0].Pos.Line+1 {
		t.Errorf("block-comment line positions must increment: %+v", lines)
	}
}

func TestPreprocessStripsMarkdownTablePipe(t *testing.T) {
	src := `package p

// | swagger:model Foo |
type Foo int
`
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	// Leading pipe stripped; content (including trailing pipe) preserved.
	if lines[0].Text != "swagger:model Foo |" {
		t.Errorf("got %q want %q", lines[0].Text, "swagger:model Foo |")
	}
}

func TestPreprocessPreservesEmbeddedWhitespace(t *testing.T) {
	src := `package p

//     indented content
type Foo int
`
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	// trimContentPrefix strips leading whitespace; embedded spaces
	// inside Text remain.
	if lines[0].Text != "indented content" {
		t.Errorf("got %q want %q", lines[0].Text, "indented content")
	}
}

func TestPreprocessColumnPrecisionLineComment(t *testing.T) {
	// "// foo" — 'f' sits at column 4 (slash, slash, space, f).
	src := "package p\n\n// foo\ntype Foo int\n"
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d", len(lines))
	}
	if lines[0].Text != "foo" {
		t.Fatalf("text: got %q want %q", lines[0].Text, "foo")
	}
	if lines[0].Pos.Column != 4 {
		t.Errorf("Column: got %d want 4", lines[0].Pos.Column)
	}
	if lines[0].Pos.Line != 3 {
		t.Errorf("Line: got %d want 3", lines[0].Pos.Line)
	}
}

func TestPreprocessColumnPrecisionBlockComment(t *testing.T) {
	// Block:
	//   /*
	//    * swagger:model Foo
	//    */
	// Continuation line " * swagger:model Foo" — 's' of "swagger"
	// sits at column 4 (space, *, space, s).
	src := "package p\n\n/*\n * swagger:model Foo\n */\ntype Foo int\n"
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	// 3 lines: empty opening, content, empty-ish closing.
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %+v", len(lines), lines)
	}
	content := lines[1]
	if content.Text != wantModelFoo {
		t.Fatalf("text: got %q want %q", content.Text, wantModelFoo)
	}
	if content.Pos.Line != 4 {
		t.Errorf("Line: got %d want 4", content.Pos.Line)
	}
	if content.Pos.Column != 4 {
		t.Errorf("Column: got %d want 4", content.Pos.Column)
	}
}

func TestPreprocessColumnPrecisionIndentedBlockComment(t *testing.T) {
	// Block comment not attached to a decl — parse directly via
	// the AST so we can exercise stripComment's continuation-line
	// offset math without going through a declaration's Doc field.
	src := "package p\n\nvar _ = /*\n\t* bar\n*/ 42\n"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "t.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Comments) == 0 {
		t.Fatal("no comments")
	}
	lines := Preprocess(f.Comments[0], fset)
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d: %+v", len(lines), lines)
	}
	// Line 1 content is "bar" (continuation line uses a tab + * + space).
	if lines[1].Text != "bar" {
		t.Fatalf("text: got %q want %q", lines[1].Text, "bar")
	}
	// Continuation lines always start at Column=1 in source; 'bar'
	// follows "\t* " which is 3 bytes, so Column = 4.
	if lines[1].Pos.Column != 4 {
		t.Errorf("Column: got %d want 4", lines[1].Pos.Column)
	}
}

func TestPreprocessOffsetAdvancesMonotonically(t *testing.T) {
	src := "package p\n\n// one\n// two\n// three\ntype Foo int\n"
	cg, fset := parseSource(t, src)
	lines := Preprocess(cg, fset)
	for i := 1; i < len(lines); i++ {
		if lines[i].Pos.Offset <= lines[i-1].Pos.Offset {
			t.Errorf("offset did not advance: line %d offset %d, line %d offset %d",
				i-1, lines[i-1].Pos.Offset, i, lines[i].Pos.Offset)
		}
	}
}

func TestPreprocessMultiCommentGroup(t *testing.T) {
	// A comment group with multiple *ast.Comment entries separated by
	// only whitespace — Go groups them into a single CommentGroup.
	src := `package p

// first
// second
// third
type Foo int
`
	cg, fset := parseSource(t, src)
	if len(cg.List) < 2 {
		t.Fatalf("expected multi-entry CommentGroup, got %d", len(cg.List))
	}
	lines := Preprocess(cg, fset)
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	want := []string{"first", "second", "third"}
	for i, w := range want {
		if lines[i].Text != w {
			t.Errorf("line %d: got %q want %q", i, lines[i].Text, w)
		}
	}
}
