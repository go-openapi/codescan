// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"testing"
)

// firstPropertyTyped extracts the first Property of the block and
// returns its Typed value. Helper for compact test cases.
//
//nolint:ireturn // returns Block per the package's polymorphic API
func firstPropertyTyped(t *testing.T, src string) (Property, Block) {
	t.Helper()
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	for p := range b.Properties() {
		return p, b
	}
	t.Fatal("block has no properties")
	return Property{}, nil
}

// --- Number ---

func TestTypeConvertNumber(t *testing.T) {
	p, b := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// maximum: 5.5\ntype Foo int\n")
	if p.Typed.Type != ValueNumber {
		t.Fatalf("Typed.Type: got %v want ValueNumber", p.Typed.Type)
	}
	if p.Typed.Number != 5.5 {
		t.Errorf("Number: got %v want 5.5", p.Typed.Number)
	}
	if p.Typed.Op != "" {
		t.Errorf("Op: got %q want empty", p.Typed.Op)
	}
	if len(b.Diagnostics()) != 0 {
		t.Errorf("unexpected diagnostics: %+v", b.Diagnostics())
	}
}

func TestTypeConvertNumberWithOperator(t *testing.T) {
	cases := []struct {
		raw    string
		wantOp string
		wantN  float64
	}{
		{"<5", "<", 5},
		{"<= 5.5", "<=", 5.5},
		{">=0", ">=", 0},
		{"> 3.14", ">", 3.14},
		{"= 42", "=", 42},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			src := "package p\n\n// swagger:model Foo\n// maximum: " + tc.raw + "\ntype Foo int\n"
			p, _ := firstPropertyTyped(t, src)
			if p.Typed.Op != tc.wantOp {
				t.Errorf("Op: got %q want %q", p.Typed.Op, tc.wantOp)
			}
			if p.Typed.Number != tc.wantN {
				t.Errorf("Number: got %v want %v", p.Typed.Number, tc.wantN)
			}
		})
	}
}

func TestTypeConvertNumberInvalid(t *testing.T) {
	_, b := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// maximum: notanumber\ntype Foo int\n")
	foundInvalid := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidNumber {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Errorf("want CodeInvalidNumber diagnostic, got %+v", b.Diagnostics())
	}
}

// --- Integer ---

func TestTypeConvertInteger(t *testing.T) {
	p, _ := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// maxLength: 42\ntype Foo int\n")
	if p.Typed.Type != ValueInteger {
		t.Fatalf("Typed.Type: got %v want ValueInteger", p.Typed.Type)
	}
	if p.Typed.Integer != 42 {
		t.Errorf("Integer: got %v want 42", p.Typed.Integer)
	}
}

func TestTypeConvertIntegerInvalid(t *testing.T) {
	_, b := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// maxLength: 5.5\ntype Foo int\n")
	foundInvalid := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidInteger {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Errorf("want CodeInvalidInteger diagnostic (integer rejects fractions), got %+v", b.Diagnostics())
	}
}

// --- Boolean ---

func TestTypeConvertBoolean(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"true", true},
		{"false", false},
		{"True", true},   // case-insensitive
		{"FALSE", false}, // case-insensitive
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			src := "package p\n\n// swagger:model Foo\n// readOnly: " + tc.raw + "\ntype Foo int\n"
			p, _ := firstPropertyTyped(t, src)
			if p.Typed.Type != ValueBoolean {
				t.Fatalf("Typed.Type: got %v", p.Typed.Type)
			}
			if p.Typed.Boolean != tc.want {
				t.Errorf("Boolean: got %v want %v", p.Typed.Boolean, tc.want)
			}
		})
	}
}

func TestTypeConvertBooleanRejectsNumeric(t *testing.T) {
	// stdlib strconv.ParseBool would accept "1" / "0" — we reject.
	_, b := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// readOnly: 1\ntype Foo int\n")
	foundInvalid := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidBoolean {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Errorf("want CodeInvalidBoolean for '1' (strict true/false only), got %+v", b.Diagnostics())
	}
}

// --- StringEnum ---

func TestTypeConvertStringEnum(t *testing.T) {
	// "in" is a StringEnum restricted to {query, path, header, body, formData}.
	src := `package p

// swagger:parameters listPets
//
// in: query
type PetParams struct{}
`
	p, _ := firstPropertyTyped(t, src)
	if p.Typed.Type != ValueStringEnum {
		t.Fatalf("Typed.Type: got %v want ValueStringEnum", p.Typed.Type)
	}
	if p.Typed.String != fixtureEnumInQuery {
		t.Errorf("String: got %q want query", p.Typed.String)
	}
}

func TestTypeConvertStringEnumCanonicalizes(t *testing.T) {
	// Enum lookup is case-insensitive but the canonical (table-spelled)
	// value is returned.
	src := `package p

// swagger:parameters listPets
//
// in: QUERY
type PetParams struct{}
`
	p, _ := firstPropertyTyped(t, src)
	if p.Typed.String != fixtureEnumInQuery {
		t.Errorf("canonicalized String: got %q want query (table casing)", p.Typed.String)
	}
}

func TestTypeConvertStringEnumInvalid(t *testing.T) {
	src := `package p

// swagger:parameters listPets
//
// in: bogus
type PetParams struct{}
`
	_, b := firstPropertyTyped(t, src)
	foundInvalid := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidStringEnum {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Errorf("want CodeInvalidStringEnum, got %+v", b.Diagnostics())
	}
}

// --- Non-primitive value types keep Typed zero ---

func TestTypeConvertNonPrimitivesStayZero(t *testing.T) {
	// pattern is ValueString (verbatim) → Typed should be zero.
	p, _ := firstPropertyTyped(t, "package p\n\n// swagger:model Foo\n// pattern: ^[a-z]+$\ntype Foo int\n")
	if p.Typed.Type != ValueNone {
		t.Errorf("pattern: Typed.Type should be ValueNone (no parse-time conversion), got %v", p.Typed.Type)
	}
	if p.Value != "^[a-z]+$" {
		t.Errorf("raw Value must survive verbatim: got %q", p.Value)
	}
}
