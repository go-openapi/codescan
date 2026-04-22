// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/ast"
	"go/parser"
	"testing"

	oaispec "github.com/go-openapi/spec"
)

// fieldType parses a Go type expression like `[]string` or `[][]int`
// via go/parser and returns the parsed ast.Expr. Helper for the
// collectItemsLevels walk tests below.
func fieldType(t *testing.T, expr string) ast.Expr {
	t.Helper()
	e, err := parser.ParseExpr(expr)
	if err != nil {
		t.Fatalf("parseExpr %q: %v", expr, err)
	}
	return e
}

// arrayTypeElt returns the Elt of an ArrayType — the same entry point
// createParser uses before invoking parseArrayTypes / collectItemsLevels.
func arrayTypeElt(t *testing.T, expr string) ast.Expr {
	t.Helper()
	at, ok := fieldType(t, expr).(*ast.ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType for %q", expr)
	}
	return at.Elt
}

// newItemsChain returns a SchemaOrArray chain deep enough to hold
// `depth` items levels, mirroring the shape buildFromType produces
// when it walks a slice-of-slice-of-... type via Typable.Items().
func newItemsChain(depth int) *oaispec.SchemaOrArray {
	if depth <= 0 {
		return nil
	}
	root := &oaispec.SchemaOrArray{Schema: &oaispec.Schema{}}
	cur := root
	for range depth - 1 {
		cur.Schema.Items = &oaispec.SchemaOrArray{Schema: &oaispec.Schema{}}
		cur = cur.Schema.Items
	}
	return root
}

func TestCollectItemsLevelsFlatSlice(t *testing.T) {
	// []string — one level of items validations (grammar level 1).
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]string"), items, 1)
	if len(got) != 1 || got[0].level != 1 || got[0].schema != items.Schema {
		t.Errorf("[]string: want [{1, items.Schema}], got %+v", got)
	}
}

func TestCollectItemsLevelsNestedSlice(t *testing.T) {
	// [][]string — two levels: grammar 1 (outer element slice) and
	// grammar 2 (inner scalar string registered via the *ast.Ident
	// Obj==nil branch).
	items := newItemsChain(2)
	got := collectItemsLevels(arrayTypeElt(t, "[][]string"), items, 1)
	if len(got) != 2 {
		t.Fatalf("[][]string: want 2 levels, got %d (%+v)", len(got), got)
	}
	if got[0].level != 1 || got[0].schema != items.Schema {
		t.Errorf("[][]string level 1: got %+v", got[0])
	}
	if got[1].level != 2 || got[1].schema != items.Schema.Items.Schema {
		t.Errorf("[][]string level 2: got %+v", got[1])
	}
}

func TestCollectItemsLevelsPointerElt(t *testing.T) {
	// []*string — StarExpr unwraps without advancing level.
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]*string"), items, 1)
	if len(got) != 1 || got[0].level != 1 {
		t.Errorf("[]*string: want one level-1 entry, got %+v", got)
	}
}

func TestCollectItemsLevelsNamedElt(t *testing.T) {
	// []Foo — Ident with Obj set (resolved by go/parser to a local
	// scope binding). parser.ParseExpr does NOT resolve scope, so Obj
	// is nil here; we simulate the "Obj != nil" case by synthesising.
	items := newItemsChain(1)
	ident := &ast.Ident{Name: "Foo", Obj: ast.NewObj(ast.Typ, "Foo")}

	got := collectItemsLevels(ident, items, 1)
	// Obj != nil → skip registration at this level; recursion advances
	// to items.Schema.Items which is nil → terminates → empty result.
	if len(got) != 0 {
		t.Errorf("named ident: want no levels, got %+v", got)
	}
}

func TestCollectItemsLevelsStructElt(t *testing.T) {
	// []struct{X int} — StructType terminates without registering.
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]struct{X int}"), items, 1)
	if len(got) != 0 {
		t.Errorf("[]struct{...}: want no levels, got %+v", got)
	}
}

func TestCollectItemsLevelsNilItems(t *testing.T) {
	// nil items → no panic, empty result.
	got := collectItemsLevels(arrayTypeElt(t, "[]string"), nil, 1)
	if len(got) != 0 {
		t.Errorf("nil items: want empty, got %+v", got)
	}
	var empty oaispec.SchemaOrArray
	got = collectItemsLevels(arrayTypeElt(t, "[]string"), &empty, 1)
	if len(got) != 0 {
		t.Errorf("items with nil Schema: want empty, got %+v", got)
	}
}

// TestApplyItemsBridgeGuards verifies the applyItemsBridge guards
// short-circuit under expected conditions. It builds a minimal Builder
// with a nil ctx-dependent path and exercises the early returns.
func TestApplyItemsBridgeGuards(_ *testing.T) {
	var b Builder

	b.applyItemsBridge(nil, &oaispec.Schema{}) // fld=nil
	fld := &ast.Field{Type: &ast.ArrayType{Elt: &ast.Ident{Name: "string"}}}
	b.applyItemsBridge(fld, &oaispec.Schema{}) // fld.Doc=nil
	fld.Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "// items.maximum: 5"}}}
	b.applyItemsBridge(fld, nil) // ps=nil
	fld.Type = &ast.Ident{Name: "string"}
	b.applyItemsBridge(fld, &oaispec.Schema{}) // non-array type
}
