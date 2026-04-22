// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"strings"
	"testing"
)

// contextInvalidDiagnostics returns the subset of b's diagnostics
// with Code == CodeContextInvalid. Used by every test in this file.
func contextInvalidDiagnostics(b Block) []Diagnostic {
	var out []Diagnostic
	for _, d := range b.Diagnostics() {
		if d.Code == CodeContextInvalid {
			out = append(out, d)
		}
	}
	return out
}

func TestContextValidityLegalKeyword(t *testing.T) {
	// `maximum` is legal under swagger:model (KindSchema) — no diagnostic.
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// maximum: 5\ntype Foo int\n")
	b := Parse(cg, fset)
	if got := contextInvalidDiagnostics(b); len(got) != 0 {
		t.Errorf("unexpected context-invalid diagnostic: %+v", got)
	}
}

func TestContextValidityIllegalKeyword(t *testing.T) {
	// `in:` is KindParam only — illegal under swagger:model.
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// in: query\ntype Foo int\n")
	b := Parse(cg, fset)
	diag := contextInvalidDiagnostics(b)
	if len(diag) != 1 {
		t.Fatalf("want 1 context-invalid diagnostic, got %d: %+v", len(diag), b.Diagnostics())
	}
	if !strings.Contains(diag[0].Message, "in") {
		t.Errorf("diagnostic message should mention the keyword: %q", diag[0].Message)
	}
	if !strings.Contains(diag[0].Message, "param") {
		t.Errorf("diagnostic message should list legal contexts (param): %q", diag[0].Message)
	}
	if diag[0].Severity != SeverityWarning {
		t.Errorf("severity: got %v want warning", diag[0].Severity)
	}
}

func TestContextValidityMultipleIllegalKeywords(t *testing.T) {
	// Two illegal keywords under swagger:route: `version` (KindMeta)
	// and `license` (KindMeta).
	src := `package p

// swagger:route GET /pets tags listPets
//
// version: 1.0
// license: MIT
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	diag := contextInvalidDiagnostics(b)
	if len(diag) != 2 {
		t.Errorf("want 2 context-invalid diagnostics, got %d: %+v", len(diag), diag)
	}
}

func TestContextValidityLegalUnderMultipleAnnotations(t *testing.T) {
	// `consumes:` is legal under Meta, Route, Operation — all three
	// should produce zero context-invalid diagnostics.
	srcs := []string{
		"package p\n\n// swagger:meta\n//\n// consumes:\ntype Root struct{}\n",
		"package p\n\n// swagger:route GET /p tags o\n//\n// consumes:\nfunc F() {}\n",
		"package p\n\n// swagger:operation GET /p o\n//\n// consumes:\nfunc F() {}\n",
	}
	for _, src := range srcs {
		t.Run(src[:40], func(t *testing.T) {
			cg, fset := parseCommentGroup(t, src)
			b := Parse(cg, fset)
			if got := contextInvalidDiagnostics(b); len(got) != 0 {
				t.Errorf("unexpected context-invalid diagnostic: %+v", got)
			}
		})
	}
}

func TestContextValiditySkipsUnboundBlock(t *testing.T) {
	// No annotation -> UnboundBlock -> no context check.
	cg, fset := parseCommentGroup(t, "package p\n\n// maximum: 5\n// in: query\n// version: 1.0\ntype Foo int\n")
	b := Parse(cg, fset)
	if got := contextInvalidDiagnostics(b); len(got) != 0 {
		t.Errorf("UnboundBlock must skip context check, got %+v", got)
	}
}

func TestContextValidityDiagnosticsAreWarnings(t *testing.T) {
	// Context-invalid must never be fatal — the parser should still
	// have produced a valid Block.
	cg, fset := parseCommentGroup(t, "package p\n\n// swagger:model Foo\n// in: query\ntype Foo int\n")
	b := Parse(cg, fset)
	if _, ok := b.(*ModelBlock); !ok {
		t.Fatalf("parser should still produce ModelBlock, got %T", b)
	}
	for _, d := range b.Diagnostics() {
		if d.Code == CodeContextInvalid && d.Severity == SeverityError {
			t.Errorf("context-invalid must be warning, not error: %+v", d)
		}
	}
}
