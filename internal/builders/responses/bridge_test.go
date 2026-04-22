// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package responses

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// ---------- collectHeaderItemsLevels ----------

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

func newItemsChain(depth int) *oaispec.Items {
	if depth <= 0 {
		return nil
	}
	root := new(oaispec.Items)
	cur := root
	for range depth - 1 {
		cur.Items = new(oaispec.Items)
		cur = cur.Items
	}
	return root
}

func TestCollectHeaderItemsLevelsFlatSlice(t *testing.T) {
	it := newItemsChain(1)
	got := collectHeaderItemsLevels(arrayTypeElt(t, "[]string"), it, 1)
	if len(got) != 1 || got[0].level != 1 || got[0].items != it {
		t.Errorf("[]string: got %+v", got)
	}
}

func TestCollectHeaderItemsLevelsNestedSlice(t *testing.T) {
	it := newItemsChain(2)
	got := collectHeaderItemsLevels(arrayTypeElt(t, "[][]string"), it, 1)
	if len(got) != 2 {
		t.Fatalf("[][]string: got %d entries", len(got))
	}
	if got[0].level != 1 || got[0].items != it {
		t.Errorf("level 1: %+v", got[0])
	}
	if got[1].level != 2 || got[1].items != it.Items {
		t.Errorf("level 2: %+v", got[1])
	}
}

func TestCollectHeaderItemsLevelsNilItems(t *testing.T) {
	got := collectHeaderItemsLevels(arrayTypeElt(t, "[]string"), nil, 1)
	if len(got) != 0 {
		t.Errorf("nil items: got %+v", got)
	}
}

// ---------- dispatchHeaderKeyword ----------

//nolint:ireturn // grammar.Block is the package's polymorphic return.
func parseResponseBody(t *testing.T, body string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseAs(grammar.AnnResponse, body, token.Position{Line: 1})
}

func runDispatch(t *testing.T, header *oaispec.Header, body string) {
	t.Helper()
	b := parseResponseBody(t, body)
	valid := headerValidations{header}
	scheme := &header.SimpleSchema
	for prop := range b.Properties() {
		if prop.ItemsDepth != 0 {
			continue
		}
		dispatchHeaderKeyword(prop, valid, scheme)
	}
}

func TestDispatchHeaderKeywordNumeric(t *testing.T) {
	h := &oaispec.Header{}
	h.Type = "integer"
	runDispatch(t, h, "maximum: <10\nminimum: >=0\nmultipleOf: 2")

	if h.Maximum == nil || *h.Maximum != 10 || !h.ExclusiveMaximum {
		t.Errorf("maximum: got (%v, %v)", h.Maximum, h.ExclusiveMaximum)
	}
	if h.Minimum == nil || *h.Minimum != 0 || h.ExclusiveMinimum {
		t.Errorf("minimum: got (%v, %v)", h.Minimum, h.ExclusiveMinimum)
	}
	if h.MultipleOf == nil || *h.MultipleOf != 2 {
		t.Errorf("multipleOf: got %v", h.MultipleOf)
	}
}

func TestDispatchHeaderKeywordIntegerAndUnique(t *testing.T) {
	h := &oaispec.Header{}
	runDispatch(t, h, "minLength: 3\nmaxLength: 10\nminItems: 1\nmaxItems: 100\nunique: true")

	if h.MinLength == nil || *h.MinLength != 3 {
		t.Errorf("minLength: %v", h.MinLength)
	}
	if h.MaxLength == nil || *h.MaxLength != 10 {
		t.Errorf("maxLength: %v", h.MaxLength)
	}
	if h.MinItems == nil || *h.MinItems != 1 {
		t.Errorf("minItems: %v", h.MinItems)
	}
	if h.MaxItems == nil || *h.MaxItems != 100 {
		t.Errorf("maxItems: %v", h.MaxItems)
	}
	if !h.UniqueItems {
		t.Errorf("unique: want true")
	}
}

func TestDispatchHeaderKeywordPatternAndEnum(t *testing.T) {
	h := &oaispec.Header{}
	h.Type = "string"
	runDispatch(t, h, "pattern: ^[a-z]+$\nenum: red, green, blue")

	if h.Pattern != "^[a-z]+$" {
		t.Errorf("pattern: %q", h.Pattern)
	}
	if len(h.Enum) != 3 || h.Enum[0] != "red" {
		t.Errorf("enum: %v", h.Enum)
	}
}

func TestDispatchHeaderKeywordDefaultExampleScheme(t *testing.T) {
	h := &oaispec.Header{}
	h.Type = "integer"
	runDispatch(t, h, "default: 42\nexample: 7")

	if h.Default != 42 {
		t.Errorf("default: got %v (%T), want 42", h.Default, h.Default)
	}
	if h.Example != 7 {
		t.Errorf("example: got %v (%T), want 7", h.Example, h.Example)
	}
}

func TestDispatchHeaderKeywordCollectionFormat(t *testing.T) {
	h := &oaispec.Header{}
	runDispatch(t, h, "collectionFormat: csv")

	if h.CollectionFormat != "csv" {
		t.Errorf("collectionFormat: %q", h.CollectionFormat)
	}
}
