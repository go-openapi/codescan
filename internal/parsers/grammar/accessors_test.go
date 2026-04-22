// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"slices"
	"testing"
)

// P3.3: Block accessors return (value, ok) tuples; ok is false when
// the keyword is absent or its type doesn't match.

func TestAccessorHasAndAbsent(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// maximum: 5\ntype Foo int\n")
	b := Parse(cg, fset)

	if !b.Has(fixtureValidationKw) {
		t.Error("Has(maximum): want true")
	}
	if b.Has("nonexistent") {
		t.Error("Has(nonexistent): want false")
	}
}

func TestAccessorAliasLookup(t *testing.T) {
	// `maximum` has alias `max`; accessor should find via either spelling.
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// max: 5\ntype Foo int\n")
	b := Parse(cg, fset)

	if !b.Has("max") {
		t.Error("Has via alias: want true")
	}
	if !b.Has("MAX") {
		t.Error("Has is case-insensitive: want true")
	}
	if !b.Has(fixtureValidationKw) {
		t.Error("Has via canonical should also work: want true")
	}

	v, ok := b.GetFloat(fixtureValidationKw)
	if !ok || v != 5 {
		t.Errorf("GetFloat(maximum): got (%v, %v) want (5, true)", v, ok)
	}
}

func TestAccessorGetFloat(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// maximum: 5.5\ntype Foo int\n")
	b := Parse(cg, fset)

	v, ok := b.GetFloat(fixtureValidationKw)
	if !ok || v != 5.5 {
		t.Errorf("GetFloat: got (%v, %v)", v, ok)
	}

	// Wrong type (pattern is ValueString) → ok=false.
	_, ok = b.GetFloat("pattern")
	if ok {
		t.Error("GetFloat on missing keyword: want ok=false")
	}
}

func TestAccessorGetInt(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// maxLength: 42\ntype Foo int\n")
	b := Parse(cg, fset)

	v, ok := b.GetInt("maxLength")
	if !ok || v != 42 {
		t.Errorf("GetInt: got (%v, %v)", v, ok)
	}
}

func TestAccessorGetBool(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// readOnly: true\ntype Foo int\n")
	b := Parse(cg, fset)

	v, ok := b.GetBool("readOnly")
	if !ok || !v {
		t.Errorf("GetBool: got (%v, %v)", v, ok)
	}
}

func TestAccessorGetStringRawValue(t *testing.T) {
	// pattern is ValueString: accessor returns the raw Value.
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// pattern: ^[a-z]+$\ntype Foo int\n")
	b := Parse(cg, fset)

	v, ok := b.GetString("pattern")
	if !ok || v != "^[a-z]+$" {
		t.Errorf("GetString(pattern): got (%q, %v)", v, ok)
	}
}

func TestAccessorGetStringEnum(t *testing.T) {
	// StringEnum returns the canonical (table-spelled) value.
	src := `package p

// swagger:parameters listPets
//
// in: QUERY
type PetParams struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	v, ok := b.GetString("in")
	if !ok || v != "query" {
		t.Errorf("GetString(in) canonical: got (%q, %v) want (query, true)", v, ok)
	}
}

func TestAccessorGetListCommaList(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// enum: a, b, c\ntype Foo int\n")
	b := Parse(cg, fset)

	v, ok := b.GetList("enum")
	if !ok {
		t.Fatalf("GetList(enum): want ok=true, got %v", ok)
	}
	if !slices.Equal(v, []string{"a", "b", "c"}) {
		t.Errorf("GetList(enum): got %v want [a b c]", v)
	}
}

func TestAccessorGetListBlockBody(t *testing.T) {
	src := `package p

// swagger:meta
//
// consumes:
//   application/json
//   application/xml
type Root struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	v, ok := b.GetList(fixtureBlockKw)
	if !ok {
		t.Fatalf("GetList(consumes): want ok=true, got %v", ok)
	}
	if !slices.Equal(v, []string{"application/json", "application/xml"}) {
		t.Errorf("GetList(consumes): got %v", v)
	}
}

func TestAccessorGetListReturnsCopy(t *testing.T) {
	// Mutating the returned slice must not affect Block state.
	src := `package p

// swagger:meta
//
// consumes:
//   application/json
type Root struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	v1, ok := b.GetList(fixtureBlockKw)
	if !ok {
		t.Fatal("GetList: want ok=true")
	}
	v1[0] = fixtureMutatedMark

	v2, _ := b.GetList(fixtureBlockKw)
	if v2[0] == fixtureMutatedMark {
		t.Error("GetList must return a defensive copy")
	}
}

func TestAccessorTypeMismatchReturnsFalse(t *testing.T) {
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// maximum: 5\ntype Foo int\n")
	b := Parse(cg, fset)

	// maximum is Number, not Integer/Boolean/List.
	if _, ok := b.GetInt(fixtureValidationKw); ok {
		t.Error("GetInt on Number-typed keyword: want ok=false")
	}
	if _, ok := b.GetBool(fixtureValidationKw); ok {
		t.Error("GetBool on Number-typed keyword: want ok=false")
	}
	if _, ok := b.GetList(fixtureValidationKw); ok {
		t.Error("GetList on scalar-typed keyword: want ok=false")
	}
}
