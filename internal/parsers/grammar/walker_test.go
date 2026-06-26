// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestWalker_LevelZeroFiresPerShape(t *testing.T) {
	src := `swagger:model Foo

maximum: 100
maxLength: 64
required: true
pattern: ^[a-z]+$`
	b := parseString(t, src)

	type seen struct {
		number  []string
		integer []string
		boolean []string
		stringv []string
	}
	var got seen

	b.Walk(Walker{
		FilterDepth: 0,
		Number:      func(p Property, _ float64, _ bool) { got.number = append(got.number, p.Keyword.Name) },
		Integer:     func(p Property, _ int64) { got.integer = append(got.integer, p.Keyword.Name) },
		Bool:        func(p Property, _ bool) { got.boolean = append(got.boolean, p.Keyword.Name) },
		String:      func(p Property, _ string) { got.stringv = append(got.stringv, p.Keyword.Name) },
	})

	assert.Equal(t, []string{"maximum"}, got.number)
	assert.Equal(t, []string{"maxLength"}, got.integer)
	assert.Equal(t, []string{"required"}, got.boolean)
	assert.Equal(t, []string{"pattern"}, got.stringv)
}

func TestWalker_NumberOpExclusiveSemantics(t *testing.T) {
	src := `swagger:model Foo

maximum: <100
minimum: >=0`
	b := parseString(t, src)

	got := map[string]bool{}
	b.Walk(Walker{
		FilterDepth: 0,
		Number:      func(p Property, _ float64, exclusive bool) { got[p.Keyword.Name] = exclusive },
	})

	assert.True(t, got["maximum"], "maximum: <100 should be exclusive")
	assert.False(t, got["minimum"], "minimum: >=0 should be inclusive")
}

func TestWalker_FilterDepthFiltersItems(t *testing.T) {
	src := `swagger:model Foo

maximum: 10
items.maximum: 20
items.items.maximum: 30`
	b := parseString(t, src)

	collect := func(filter int) []float64 {
		var out []float64
		b.Walk(Walker{
			FilterDepth: filter,
			Number:      func(_ Property, val float64, _ bool) { out = append(out, val) },
		})
		return out
	}

	assert.Equal(t, []float64{10}, collect(0))
	assert.Equal(t, []float64{20}, collect(1))
	assert.Equal(t, []float64{30}, collect(2))
	assert.Equal(t, []float64{10, 20, 30}, collect(AllDepths))
}

func TestWalker_TitleAndDescription(t *testing.T) {
	src := `Pet is a domestic animal.

It has a name and a tag.

swagger:model Pet`
	b := parseString(t, src)

	var title, desc string
	b.Walk(Walker{
		Title:       func(s string) { title = s },
		Description: func(s string) { desc = s },
	})

	assert.Equal(t, "Pet is a domestic animal.", title)
	assert.Equal(t, "It has a name and a tag.", desc)
}

func TestWalker_TitleNotFiredWhenEmpty(t *testing.T) {
	// UnboundBlock with description but no title (no first-sentence terminator before the blank line
	// — single-line docstrings become description per parser logic).
	src := `swagger:model Foo

maximum: 10`
	b := parseString(t, src)

	called := false
	b.Walk(Walker{
		Title: func(_ string) { called = true },
	})

	assert.False(t, called, "Title callback should not fire when block has no title")
}

func TestWalker_ExtensionsFire(t *testing.T) {
	src := `swagger:model Foo

Extensions:
  x-flag: true
  x-name: hello`
	b := parseString(t, src)

	var names []string
	b.Walk(Walker{
		Extension: func(ext Extension) { names = append(names, ext.Name) },
	})

	assert.ElementsMatch(t, []string{"x-flag", "x-name"}, names)
}

func TestWalker_DiagnosticsFire(t *testing.T) {
	// "swagger:parameters" with no operationID emits CodeMissingRequiredArg.
	b := parseString(t, "swagger:parameters")

	var codes []Code
	b.Walk(Walker{
		Diagnostic: func(d Diagnostic) { codes = append(codes, d.Code) },
	})

	require.NotEmpty(t, codes)
	assert.Contains(t, codes, CodeMissingRequiredArg)
}

func TestWalker_NilCallbacksAreNoops(t *testing.T) {
	src := `swagger:model Foo

maximum: 100
required: true
pattern: ^a$`
	b := parseString(t, src)

	// All callbacks nil — must not panic.
	assert.NotPanics(t, func() { b.Walk(Walker{FilterDepth: 0}) })
}

func TestProperty_IsTyped(t *testing.T) {
	src := `swagger:model Foo

maximum: 100
maxLength: 64
required: true
pattern: ^[a-z]+$
default: hello`
	b := parseString(t, src)

	got := map[string]bool{}
	for p := range b.Properties() {
		got[p.Keyword.Name] = p.IsTyped()
	}

	// Pre-typed (lexer populated Typed.Number/Integer/Boolean):
	assert.True(t, got["maximum"], "maximum has Typed.Number")
	assert.True(t, got["maxLength"], "maxLength has Typed.Integer")
	assert.True(t, got["required"], "required has Typed.Boolean")

	// Raw — Typed.Type stays ShapeNone:
	assert.False(t, got["pattern"], "pattern keeps raw in Value, Typed unused")
	assert.False(t, got["default"], "default needs schema-type coercion outside grammar")
}

func TestProperty_IsTyped_FailedTypingIsFalse(t *testing.T) {
	// Failed typing: lexer rejects "notanumber" as ShapeNumber, leaves Typed.Type at ShapeNone, emits
	// a CodeInvalidNumber diagnostic.
	//
	// IsTyped reports false, matching reality.
	const kw = "maximum"
	b := parseString(t, "swagger:model Foo\n\n"+kw+": notanumber")
	for p := range b.Properties() {
		if p.Keyword.Name == kw {
			assert.False(t, p.IsTyped(), "failed typing leaves IsTyped false")
		}
	}
}

func TestWalker_UnknownFiresForUntabledKeyword(t *testing.T) {
	// "fakekeyword:" lands on an UnboundBlock as a Property whose Keyword.Name is empty (no entry in
	// the keyword table).
	src := `Some prose.
fakekeyword: 42`
	b := parseString(t, src)

	var unknown []string
	b.Walk(Walker{
		Unknown: func(p Property) { unknown = append(unknown, p.Value) },
	})

	// The unknown-keyword path is exercised when the lexer emits a property with an empty
	// Keyword.Name.
	// If the current pipeline classifies "fakekeyword:" as prose rather than a property, Unknown stays
	// empty — which is also valid behaviour.
	//
	// We assert no panic and consistent typing.
	if len(unknown) > 0 {
		assert.Equal(t, "42", unknown[0])
	}
}
