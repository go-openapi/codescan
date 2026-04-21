// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"slices"
	"strings"
	"testing"
)

// This file holds one focused test per grammar-envelope production
// (architecture §2.1). It complements the per-component suites in
// preprocess_test.go, lexer_test.go, parser_test.go, typeconv_test.go,
// and context_test.go — those cover mechanisms; this file covers the
// named productions as discrete units.

const (
	fixtureModelName = "Foo"
	fixtureBlockKw   = "consumes"
)

// --- annotation-line ---

func TestProductionAnnotationOnly(t *testing.T) {
	// Annotation with no surrounding text or body. Block created,
	// zero properties, zero diagnostics.
	src := `package p

// swagger:model Foo
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	mb, ok := b.(*ModelBlock)
	if !ok {
		t.Fatalf("want *ModelBlock, got %T", b)
	}
	if mb.Name != fixtureModelName {
		t.Errorf("Name: got %q want Foo", mb.Name)
	}
	if b.Title() != "" || b.Description() != "" {
		t.Errorf("title/description must be empty: %q / %q", b.Title(), b.Description())
	}
	propCount := 0
	for range b.Properties() {
		propCount++
	}
	if propCount != 0 {
		t.Errorf("want 0 properties, got %d", propCount)
	}
	if len(b.Diagnostics()) != 0 {
		t.Errorf("unexpected diagnostics: %+v", b.Diagnostics())
	}
}

// --- title-paragraph ---

func TestProductionTitleOnly(t *testing.T) {
	// One paragraph of free text, no blank separator, no body.
	// First paragraph becomes Title; Description stays empty.
	src := `package p

// Foo is a thing.
// More detail on the same paragraph.
//
// swagger:model Foo
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	if b.Title() != "Foo is a thing. More detail on the same paragraph." {
		t.Errorf("Title: got %q", b.Title())
	}
	if b.Description() != "" {
		t.Errorf("Description should be empty, got %q", b.Description())
	}
}

// --- description-paragraphs ---

func TestProductionMultiParagraphDescription(t *testing.T) {
	// Title + two description paragraphs; verify \n\n paragraph join.
	src := `package p

// Foo is a thing.
//
// First description paragraph.
//
// Second description paragraph,
// continued on a second line.
//
// swagger:model Foo
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if b.Title() != "Foo is a thing." {
		t.Errorf("Title: got %q", b.Title())
	}
	wantDesc := "First description paragraph.\n\n" +
		"Second description paragraph, continued on a second line."
	if b.Description() != wantDesc {
		t.Errorf("Description:\n got: %q\nwant: %q", b.Description(), wantDesc)
	}
}

// --- property-line ---

func TestProductionPropertiesNoTitle(t *testing.T) {
	// Properties follow the annotation immediately; no title/description.
	src := `package p

// swagger:model Foo
// maximum: 10
// minimum: 0
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if b.Title() != "" || b.Description() != "" {
		t.Errorf("title/description must be empty: %q / %q", b.Title(), b.Description())
	}
	var names []string
	for p := range b.Properties() {
		names = append(names, p.Keyword.Name)
	}
	if !slices.Equal(names, []string{"maximum", "minimum"}) {
		t.Errorf("property names: got %v", names)
	}
}

func TestProductionPropertiesInterleavedWithText(t *testing.T) {
	// TEXT lines between properties are dropped by the parser body;
	// only properties survive.
	src := `package p

// swagger:model Foo
// maximum: 10
// some commentary between properties
// minimum: 0
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var names []string
	for p := range b.Properties() {
		names = append(names, p.Keyword.Name)
	}
	if !slices.Equal(names, []string{"maximum", "minimum"}) {
		t.Errorf("interleaved text should drop, leaving properties: got %v", names)
	}
}

// --- multi-line block (block-head) ---

func TestProductionBlockHeadProperty(t *testing.T) {
	// `consumes:` as a value-less block head. P1 only records the
	// head; body-line collection is P2.3.
	src := `package p

// swagger:meta
//
// consumes:
type Root struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var saw Property
	count := 0
	for p := range b.Properties() {
		saw = p
		count++
	}
	if count != 1 {
		t.Fatalf("want 1 property, got %d", count)
	}
	if saw.Keyword.Name != fixtureBlockKw {
		t.Errorf("keyword: got %q want consumes", saw.Keyword.Name)
	}
	if saw.Value != "" {
		t.Errorf("block-head Value must be empty: %q", saw.Value)
	}
}

// --- yaml-block ---

func TestProductionEmptyYAMLBody(t *testing.T) {
	// --- immediately followed by --- (empty body). One RawYAML with
	// empty text.
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	count := 0
	for y := range b.YAMLBlocks() {
		count++
		if y.Text != "" {
			t.Errorf("empty fence body: want empty text, got %q", y.Text)
		}
	}
	if count != 1 {
		t.Errorf("want 1 YAML block, got %d", count)
	}
	if len(b.Diagnostics()) != 0 {
		t.Errorf("unexpected diagnostics for balanced empty fences: %+v", b.Diagnostics())
	}
}

func TestProductionMultipleYAMLBlocks(t *testing.T) {
	// Two independently fenced YAML sections — both captured.
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// first: block
// ---
//
// Prose between the blocks.
//
// ---
// second: block
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var texts []string
	for y := range b.YAMLBlocks() {
		texts = append(texts, y.Text)
	}
	if len(texts) != 2 {
		t.Fatalf("want 2 YAML blocks, got %d: %v", len(texts), texts)
	}
	if !strings.Contains(texts[0], "first") {
		t.Errorf("block 0 missing 'first': %q", texts[0])
	}
	if !strings.Contains(texts[1], "second") {
		t.Errorf("block 1 missing 'second': %q", texts[1])
	}
}

// --- enum String() exhaustiveness (coverage-fill) ---

func TestEnumStringExhaustive(t *testing.T) {
	// Kind
	kindCases := []struct {
		in   Kind
		want string
	}{
		{KindUnknown, "unknown"},
		{KindParam, "param"},
		{KindHeader, "header"},
		{KindSchema, "schema"},
		{KindItems, "items"},
		{KindRoute, "route"},
		{KindOperation, "operation"},
		{KindMeta, "meta"},
		{KindResponse, "response"},
		{Kind(99), "unknown"},
	}
	for _, tc := range kindCases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("Kind(%d).String() = %q want %q", int(tc.in), got, tc.want)
		}
	}

	// ValueType
	vtCases := []struct {
		in   ValueType
		want string
	}{
		{ValueNone, "none"},
		{ValueNumber, "number"},
		{ValueInteger, "integer"},
		{ValueBoolean, "boolean"},
		{ValueString, "string"},
		{ValueCommaList, "comma-list"},
		{ValueStringEnum, "string-enum"},
		{ValueRawBlock, "raw-block"},
		{ValueRawValue, "raw-value"},
		{ValueType(99), "none"},
	}
	for _, tc := range vtCases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("ValueType(%d).String() = %q want %q", int(tc.in), got, tc.want)
		}
	}

	// TokenKind
	tkCases := []struct {
		in   TokenKind
		want string
	}{
		{TokenEOF, "EOF"},
		{TokenBlank, "BLANK"},
		{TokenText, "TEXT"},
		{TokenAnnotation, "ANNOTATION"},
		{TokenKeywordValue, "KEYWORD_VALUE"},
		{TokenKeywordBlockHead, "KEYWORD_BLOCK_HEAD"},
		{TokenYAMLFence, "YAML_FENCE"},
		{TokenKind(99), "?"},
	}
	for _, tc := range tkCases {
		if got := tc.in.String(); got != tc.want {
			t.Errorf("TokenKind(%d).String() = %q want %q", int(tc.in), got, tc.want)
		}
	}
}

// --- remaining AnnotationKind → Block dispatch paths ---

func TestAnnotationDispatchRemaining(t *testing.T) {
	cases := []struct {
		src  string
		want any
	}{
		{"package p\n\n// swagger:response okResp\ntype OK struct{}\n", (*ResponseBlock)(nil)},
		{"package p\n\n// swagger:meta\ntype Root struct{}\n", (*MetaBlock)(nil)},
		{"package p\n\n// swagger:strfmt mac\ntype MAC string\n", (*UnboundBlock)(nil)},
		{"package p\n\n// swagger:alias\ntype Alias string\n", (*UnboundBlock)(nil)},
		{"package p\n\n// swagger:allOf Base\ntype Derived struct{}\n", (*UnboundBlock)(nil)},
		{"package p\n\n// swagger:enum Colors\ntype Color int\n", (*UnboundBlock)(nil)},
		{"package p\n\n// swagger:ignore\ntype X struct{}\n", (*UnboundBlock)(nil)},
		{"package p\n\n// swagger:file\ntype F struct{}\n", (*UnboundBlock)(nil)},
	}
	for _, tc := range cases {
		cg, fset := parseCommentGroup(t, tc.src)
		b := Parse(cg, fset)
		// Compare concrete types reflectively via type switch.
		switch tc.want.(type) {
		case *ResponseBlock:
			if _, ok := b.(*ResponseBlock); !ok {
				t.Errorf("src=%q: want *ResponseBlock, got %T", tc.src, b)
			}
		case *MetaBlock:
			if _, ok := b.(*MetaBlock); !ok {
				t.Errorf("src=%q: want *MetaBlock, got %T", tc.src, b)
			}
		case *UnboundBlock:
			if _, ok := b.(*UnboundBlock); !ok {
				t.Errorf("src=%q: want *UnboundBlock, got %T", tc.src, b)
			}
		}
	}
}

// --- envelope order (all productions composed) ---

func TestProductionEnvelopeFullOrder(t *testing.T) {
	// A comment block exercising every production in its natural
	// order: title → description → properties → yaml body.
	src := `package p

// A one-line title.
//
// A description paragraph,
// continued on the next line.
//
// swagger:operation GET /pets tags listPets
//
// maximum: 100
//
// ---
// responses:
//   200: ok
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if _, ok := b.(*OperationBlock); !ok {
		t.Fatalf("want *OperationBlock, got %T", b)
	}
	if b.Title() != "A one-line title." {
		t.Errorf("Title: got %q", b.Title())
	}
	if !strings.HasPrefix(b.Description(), "A description paragraph") {
		t.Errorf("Description: got %q", b.Description())
	}
	propCount := 0
	for range b.Properties() {
		propCount++
	}
	if propCount == 0 {
		t.Error("expected at least one property (maximum)")
	}
	yamlCount := 0
	for range b.YAMLBlocks() {
		yamlCount++
	}
	if yamlCount != 1 {
		t.Errorf("want 1 YAML block, got %d", yamlCount)
	}
}
