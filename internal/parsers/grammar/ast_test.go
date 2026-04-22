// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"slices"
	"testing"
)

func TestAnnotationKindString(t *testing.T) {
	cases := []struct {
		in   AnnotationKind
		want string
	}{
		{AnnRoute, "route"},
		{AnnOperation, "operation"},
		{AnnModel, "model"},
		{AnnMeta, "meta"},
		{AnnEnumDecl, "enum"},
		{AnnUnknown, "unknown"},
		{AnnotationKind(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("%d.String() = %q want %q", int(tc.in), got, tc.want)
		}
	}
}

func TestAnnotationKindFromName(t *testing.T) {
	cases := []struct {
		in   string
		want AnnotationKind
	}{
		{"route", AnnRoute},
		{"operation", AnnOperation},
		{"model", AnnModel},
		{"meta", AnnMeta},
		{"allOf", AnnAllOf},
		{"enum", AnnEnumDecl},
		{"default", AnnDefaultName},
		{"bogus", AnnUnknown},
		{"", AnnUnknown},
	}
	for _, tc := range cases {
		if got := AnnotationKindFromName(tc.in); got != tc.want {
			t.Errorf("FromName(%q) = %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestTypedBlocksImplementInterface(_ *testing.T) {
	// Compile-time: if these assertions fail, ast.go is broken.
	var _ Block = (*ModelBlock)(nil)
	var _ Block = (*RouteBlock)(nil)
	var _ Block = (*OperationBlock)(nil)
	var _ Block = (*ParametersBlock)(nil)
	var _ Block = (*ResponseBlock)(nil)
	var _ Block = (*MetaBlock)(nil)
	var _ Block = (*UnboundBlock)(nil)
}

func TestBaseBlockAccessors(t *testing.T) {
	pos := token.Position{Filename: "t.go", Line: 3, Column: 1}
	b := newBaseBlock(AnnModel, pos)
	b.title = "A title."
	b.description = "A paragraph describing the model."
	b.properties = []Property{
		{Keyword: Keyword{Name: "maximum"}, Value: "10"},
		{Keyword: Keyword{Name: "minimum"}, Value: "0"},
	}
	b.yamlBlocks = []RawYAML{
		{Text: "foo: bar\n"},
	}
	b.extensions = []Extension{
		{Name: "x-custom", Value: "42"},
	}
	b.diagnostics = []Diagnostic{
		{Severity: SeverityWarning, Code: CodeUnknownKeyword, Message: "ignored"},
	}

	mb := &ModelBlock{baseBlock: b, Name: "Foo"}

	if mb.Name != "Foo" {
		t.Errorf("Name: got %q want Foo", mb.Name)
	}
	if mb.Pos().Line != 3 {
		t.Errorf("Pos.Line: got %d want 3", mb.Pos().Line)
	}
	if mb.Title() != "A title." {
		t.Errorf("Title: got %q", mb.Title())
	}
	if mb.AnnotationKind() != AnnModel {
		t.Errorf("Kind: got %v want AnnModel", mb.AnnotationKind())
	}
	if len(mb.Diagnostics()) != 1 {
		t.Errorf("Diagnostics: got %d want 1", len(mb.Diagnostics()))
	}

	var props []Property
	for p := range mb.Properties() {
		props = append(props, p)
	}
	if len(props) != 2 {
		t.Errorf("iterated %d properties want 2", len(props))
	}
	if !slices.Equal([]string{props[0].Keyword.Name, props[1].Keyword.Name}, []string{"maximum", "minimum"}) {
		t.Errorf("iteration order: got %q,%q", props[0].Keyword.Name, props[1].Keyword.Name)
	}

	yamlCount := 0
	for range mb.YAMLBlocks() {
		yamlCount++
	}
	if yamlCount != 1 {
		t.Errorf("YAML blocks: got %d want 1", yamlCount)
	}

	extCount := 0
	for range mb.Extensions() {
		extCount++
	}
	if extCount != 1 {
		t.Errorf("extensions: got %d want 1", extCount)
	}
}

func TestIteratorEarlyBreak(t *testing.T) {
	b := newBaseBlock(AnnModel, token.Position{})
	b.properties = []Property{
		{Keyword: Keyword{Name: "maximum"}},
		{Keyword: Keyword{Name: "minimum"}},
		{Keyword: Keyword{Name: "pattern"}},
	}
	mb := &ModelBlock{baseBlock: b}

	seen := 0
	for range mb.Properties() {
		seen++
		if seen == 2 {
			break
		}
	}
	if seen != 2 {
		t.Errorf("early-break: iterated %d want 2", seen)
	}
}
