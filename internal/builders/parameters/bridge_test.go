// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parameters

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	oaispec "github.com/go-openapi/spec"
)

// ---------- collectParamItemsLevels ----------

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

func TestCollectParamItemsLevelsFlatSlice(t *testing.T) {
	it := newItemsChain(1)
	got := collectParamItemsLevels(arrayTypeElt(t, "[]string"), it, 1)
	if len(got) != 1 || got[0].level != 1 || got[0].items != it {
		t.Errorf("[]string: got %+v", got)
	}
}

func TestCollectParamItemsLevelsNestedSlice(t *testing.T) {
	it := newItemsChain(2)
	got := collectParamItemsLevels(arrayTypeElt(t, "[][]string"), it, 1)
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

func TestCollectParamItemsLevelsNilItems(t *testing.T) {
	got := collectParamItemsLevels(arrayTypeElt(t, "[]string"), nil, 1)
	if len(got) != 0 {
		t.Errorf("nil items: got %+v", got)
	}
}

// ---------- dispatchParamKeyword ----------

//nolint:ireturn // grammar.Block is the package's polymorphic return.
func parseParamBody(t *testing.T, body string) grammar.Block {
	t.Helper()
	p := grammar.NewParser(token.NewFileSet())
	return p.ParseAs(grammar.AnnParameters, body, token.Position{Line: 1})
}

func runDispatch(t *testing.T, param *oaispec.Parameter, body string) {
	t.Helper()
	b := parseParamBody(t, body)
	valid := paramValidations{param}
	scheme := &param.SimpleSchema
	for prop := range b.Properties() {
		if prop.ItemsDepth != 0 {
			continue
		}
		if err := dispatchParamKeyword(prop, param, valid, scheme); err != nil {
			t.Fatalf("dispatchParamKeyword: %v", err)
		}
	}
}

func TestDispatchParamKeywordNumeric(t *testing.T) {
	param := &oaispec.Parameter{}
	param.Type = "integer"
	runDispatch(t, param, "maximum: <10\nminimum: >=0\nmultipleOf: 2")

	if param.Maximum == nil || *param.Maximum != 10 || !param.ExclusiveMaximum {
		t.Errorf("maximum: got (%v, %v), want (10, true)", param.Maximum, param.ExclusiveMaximum)
	}
	if param.Minimum == nil || *param.Minimum != 0 || param.ExclusiveMinimum {
		t.Errorf("minimum: got (%v, %v), want (0, false)", param.Minimum, param.ExclusiveMinimum)
	}
	if param.MultipleOf == nil || *param.MultipleOf != 2 {
		t.Errorf("multipleOf: got %v", param.MultipleOf)
	}
}

func TestDispatchParamKeywordIntegerAndFlags(t *testing.T) {
	param := &oaispec.Parameter{}
	runDispatch(t, param, "minLength: 3\nmaxLength: 10\nminItems: 1\nmaxItems: 100\nunique: true\nrequired: true")

	if param.MinLength == nil || *param.MinLength != 3 {
		t.Errorf("minLength: %v", param.MinLength)
	}
	if param.MaxLength == nil || *param.MaxLength != 10 {
		t.Errorf("maxLength: %v", param.MaxLength)
	}
	if param.MinItems == nil || *param.MinItems != 1 {
		t.Errorf("minItems: %v", param.MinItems)
	}
	if param.MaxItems == nil || *param.MaxItems != 100 {
		t.Errorf("maxItems: %v", param.MaxItems)
	}
	if !param.UniqueItems {
		t.Errorf("unique: want true")
	}
	if !param.Required {
		t.Errorf("required: want true")
	}
}

func TestDispatchParamKeywordPatternAndEnum(t *testing.T) {
	param := &oaispec.Parameter{}
	param.Type = "string"
	runDispatch(t, param, "pattern: ^[a-z]+$\nenum: red, green, blue")

	if param.Pattern != "^[a-z]+$" {
		t.Errorf("pattern: %q", param.Pattern)
	}
	if len(param.Enum) != 3 || param.Enum[0] != "red" {
		t.Errorf("enum: %v", param.Enum)
	}
}

func TestDispatchParamKeywordDefaultExampleScheme(t *testing.T) {
	param := &oaispec.Parameter{}
	param.Type = "integer"
	runDispatch(t, param, "default: 42\nexample: 7")

	if param.Default != 42 {
		t.Errorf("default: got %v (%T), want 42", param.Default, param.Default)
	}
	if param.Example != 7 {
		t.Errorf("example: got %v (%T), want 7", param.Example, param.Example)
	}
}

func TestDispatchParamKeywordCollectionFormat(t *testing.T) {
	param := &oaispec.Parameter{}
	runDispatch(t, param, "collectionFormat: multi")

	if param.CollectionFormat != "multi" {
		t.Errorf("collectionFormat: %q", param.CollectionFormat)
	}
}

func TestDispatchParamKeywordRequiredFalse(t *testing.T) {
	param := &oaispec.Parameter{}
	param.Required = true // simulate a prior path-param default
	runDispatch(t, param, "required: false")

	if param.Required {
		t.Errorf("required: want false after explicit override")
	}
}
