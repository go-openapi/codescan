// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// ---------- collectItemsLevels ---------------------------------------

func fieldType(t *testing.T, expr string) ast.Expr {
	t.Helper()
	e, err := parser.ParseExpr(expr)
	if err != nil {
		t.Fatalf("parseExpr %q: %v", expr, err)
	}
	return e
}

func arrayTypeElt(t *testing.T, expr string) ast.Expr {
	t.Helper()
	at, ok := fieldType(t, expr).(*ast.ArrayType)
	if !ok {
		t.Fatalf("expected ArrayType for %q", expr)
	}
	return at.Elt
}

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
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]string"), items, 1)
	if len(got) != 1 || got[0].level != 1 || got[0].schema != items.Schema {
		t.Errorf("[]string: want [{1, items.Schema}], got %+v", got)
	}
}

func TestCollectItemsLevelsNestedSlice(t *testing.T) {
	items := newItemsChain(2)
	got := collectItemsLevels(arrayTypeElt(t, "[][]string"), items, 1)
	if len(got) != 2 {
		t.Fatalf("[][]string: want 2 levels, got %d (%+v)", len(got), got)
	}
	if got[0].level != 1 || got[0].schema != items.Schema {
		t.Errorf("level 1: got %+v", got[0])
	}
	if got[1].level != 2 || got[1].schema != items.Schema.Items.Schema {
		t.Errorf("level 2: got %+v", got[1])
	}
}

func TestCollectItemsLevelsPointerElt(t *testing.T) {
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]*string"), items, 1)
	if len(got) != 1 || got[0].level != 1 {
		t.Errorf("[]*string: want one level-1 entry, got %+v", got)
	}
}

func TestCollectItemsLevelsNamedElt(t *testing.T) {
	items := newItemsChain(1)
	ident := &ast.Ident{Name: "Foo", Obj: ast.NewObj(ast.Typ, "Foo")}

	got := collectItemsLevels(ident, items, 1)
	if len(got) != 0 {
		t.Errorf("named ident: want no levels, got %+v", got)
	}
}

func TestCollectItemsLevelsStructElt(t *testing.T) {
	items := newItemsChain(1)
	got := collectItemsLevels(arrayTypeElt(t, "[]struct{X int}"), items, 1)
	if len(got) != 0 {
		t.Errorf("[]struct{...}: want no levels, got %+v", got)
	}
}

func TestCollectItemsLevelsNilItems(t *testing.T) {
	got := collectItemsLevels(arrayTypeElt(t, "[]string"), nil, 1)
	if len(got) != 0 {
		t.Errorf("nil items: want empty, got %+v", got)
	}
	var empty oaispec.SchemaOrArray
	got = collectItemsLevels(arrayTypeElt(t, "[]string"), &empty, 1)
	if len(got) != 0 {
		t.Errorf("empty SchemaOrArray: want empty, got %+v", got)
	}
}

// ---------- applySchemaBlock dispatch ----------------------------------

// parseSchemaBody synthesises a grammar.Block from a raw body (no
// swagger annotation required) so tests can exercise the keyword
// dispatch without string-formatting a full comment group.
//
//nolint:ireturn // same rationale as items/bridge_test.go
func parseSchemaBody(t *testing.T, body string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseAs(grammar.AnnModel, body, token.Position{Line: 1})
}

func TestApplySchemaBlockNumeric(t *testing.T) {
	ps := &oaispec.Schema{}
	ps.Type = oaispec.StringOrArray{"integer"}
	b := parseSchemaBody(t, "maximum: <10\nminimum: >=0\nmultipleOf: 2")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if ps.Maximum == nil || *ps.Maximum != 10 || !ps.ExclusiveMaximum {
		t.Errorf("maximum: got (%v, %v), want (10, true)", ps.Maximum, ps.ExclusiveMaximum)
	}
	if ps.Minimum == nil || *ps.Minimum != 0 || ps.ExclusiveMinimum {
		t.Errorf("minimum: got (%v, %v), want (0, false)", ps.Minimum, ps.ExclusiveMinimum)
	}
	if ps.MultipleOf == nil || *ps.MultipleOf != 2 {
		t.Errorf("multipleOf: got %v, want 2", ps.MultipleOf)
	}
}

func TestApplySchemaBlockIntegerAndBoolean(t *testing.T) {
	ps := &oaispec.Schema{}
	b := parseSchemaBody(t, "minLength: 3\nmaxLength: 10\nminItems: 1\nmaxItems: 100\nunique: true")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if ps.MinLength == nil || *ps.MinLength != 3 {
		t.Errorf("minLength: %v", ps.MinLength)
	}
	if ps.MaxLength == nil || *ps.MaxLength != 10 {
		t.Errorf("maxLength: %v", ps.MaxLength)
	}
	if ps.MinItems == nil || *ps.MinItems != 1 {
		t.Errorf("minItems: %v", ps.MinItems)
	}
	if ps.MaxItems == nil || *ps.MaxItems != 100 {
		t.Errorf("maxItems: %v", ps.MaxItems)
	}
	if !ps.UniqueItems {
		t.Errorf("unique: want true")
	}
}

func TestApplySchemaBlockPatternAndEnum(t *testing.T) {
	ps := &oaispec.Schema{}
	ps.Type = oaispec.StringOrArray{"string"}
	b := parseSchemaBody(t, "pattern: ^[a-z]+$\nenum: red, green, blue")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if ps.Pattern != "^[a-z]+$" {
		t.Errorf("pattern: %q", ps.Pattern)
	}
	if len(ps.Enum) != 3 || ps.Enum[0] != "red" || ps.Enum[1] != "green" || ps.Enum[2] != "blue" {
		t.Errorf("enum: %v", ps.Enum)
	}
}

func TestApplySchemaBlockDefaultAndExampleIntegerScheme(t *testing.T) {
	ps := &oaispec.Schema{}
	ps.Type = oaispec.StringOrArray{"integer"}
	b := parseSchemaBody(t, "default: 42\nexample: 7")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if ps.Default != 42 {
		t.Errorf("default: got %v (%T), want 42", ps.Default, ps.Default)
	}
	if ps.Example != 7 {
		t.Errorf("example: got %v (%T), want 7", ps.Example, ps.Example)
	}
}

func TestApplySchemaBlockRequiredAndDiscriminator(t *testing.T) {
	enclosing := &oaispec.Schema{}
	ps := &oaispec.Schema{}
	b := parseSchemaBody(t, "required: true\ndiscriminator: true")

	applySchemaBlock(b, schemaBlockTargets{enclosing: enclosing, ps: ps, name: "kind"})

	if len(enclosing.Required) != 1 || enclosing.Required[0] != "kind" {
		t.Errorf("required: %v", enclosing.Required)
	}
	if enclosing.Discriminator != "kind" {
		t.Errorf("discriminator: %q", enclosing.Discriminator)
	}
}

func TestApplySchemaBlockRequiredFalseRemoves(t *testing.T) {
	enclosing := &oaispec.Schema{}
	enclosing.Required = []string{"kind", "other"}
	ps := &oaispec.Schema{}
	b := parseSchemaBody(t, "required: false")

	applySchemaBlock(b, schemaBlockTargets{enclosing: enclosing, ps: ps, name: "kind"})

	if len(enclosing.Required) != 1 || enclosing.Required[0] != "other" {
		t.Errorf("required false: %v", enclosing.Required)
	}
}

func TestApplySchemaBlockRequiredSkipsOnEmptyName(t *testing.T) {
	// Top-level declaration case: name is "", required is a no-op.
	enclosing := &oaispec.Schema{}
	ps := &oaispec.Schema{}
	b := parseSchemaBody(t, "required: true")

	applySchemaBlock(b, schemaBlockTargets{enclosing: enclosing, ps: ps, name: ""})

	if len(enclosing.Required) != 0 {
		t.Errorf("top-level required: want empty, got %v", enclosing.Required)
	}
}

func TestApplySchemaBlockReadOnly(t *testing.T) {
	ps := &oaispec.Schema{}
	b := parseSchemaBody(t, "readOnly: true")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if !ps.ReadOnly {
		t.Errorf("readOnly: want true")
	}
}

func TestApplySchemaBlockItemsDepthSkipped(t *testing.T) {
	// Level-0 dispatch must NOT fire on ItemsDepth>=1 properties;
	// those belong to items.ApplyBlock.
	ps := &oaispec.Schema{}
	ps.Type = oaispec.StringOrArray{"integer"}
	b := parseSchemaBody(t, "maximum: 5\nitems.maximum: 99")

	applySchemaBlock(b, schemaBlockTargets{enclosing: &oaispec.Schema{}, ps: ps, name: "x"})

	if ps.Maximum == nil || *ps.Maximum != 5 {
		t.Errorf("schema-level maximum: %v", ps.Maximum)
	}
	// items.maximum at level 1 is invisible to the schema dispatcher.
	// The schema's Items isn't populated here (no array type), so the
	// non-dispatch is the assertion.
}

// parseProseBody produces a grammar.Block from prose-only text (no
// swagger annotation), matching the shape of struct-field docstrings
// where the grammar returns an UnboundBlock and populates ProseLines
// from the pre-body tokens.
//
//nolint:ireturn // grammar.Block is the package's polymorphic return.
func parseProseBody(t *testing.T, text string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseText(text, token.Position{Line: 1})
}

func TestProseLinesPreservesLineBreaks(t *testing.T) {
	// Multi-line paragraph followed by blank and a second paragraph.
	b := parseProseBody(t, "First line.\nsecond line.\n\nSecond para.")
	got := b.ProseLines()
	want := []string{"First line.", "second line.", "", "Second para."}
	if !equalStrings(got, want) {
		t.Errorf("ProseLines: got %#v, want %#v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
