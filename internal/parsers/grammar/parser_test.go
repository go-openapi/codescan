// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"slices"
	"testing"
)

// parseCommentGroup is a test helper: parse a Go snippet with
// comment-preserving mode and return the doc comment of its first
// declaration along with the FileSet.
func parseCommentGroup(t *testing.T, src string) (*ast.CommentGroup, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "t.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Decls) == 0 {
		t.Fatal("no decls in test source")
	}
	switch d := f.Decls[0].(type) {
	case *ast.GenDecl:
		return d.Doc, fset
	case *ast.FuncDecl:
		return d.Doc, fset
	}
	t.Fatal("decl has no doc comment")
	return nil, nil
}

// --- dispatch ---

func TestParseModelBlock(t *testing.T) {
	src := `package p

// swagger:model Foo
type Foo struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if b.AnnotationKind() != AnnModel {
		t.Fatalf("kind: got %v want AnnModel", b.AnnotationKind())
	}
	mb, ok := b.(*ModelBlock)
	if !ok {
		t.Fatalf("want *ModelBlock, got %T", b)
	}
	if mb.Name != "Foo" {
		t.Errorf("Name: got %q want Foo", mb.Name)
	}
}

func TestParseRouteBlock(t *testing.T) {
	src := `package p

// swagger:route GET /pets tags listPets
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	rb, ok := b.(*RouteBlock)
	if !ok {
		t.Fatalf("want *RouteBlock, got %T", b)
	}
	if rb.Method != http.MethodGet {
		t.Errorf("Method: got %q want GET", rb.Method)
	}
	if rb.Path != "/pets" {
		t.Errorf("Path: got %q want /pets", rb.Path)
	}
	if rb.Tags != "tags" {
		t.Errorf("Tags: got %q want tags", rb.Tags)
	}
	if rb.OpID != "listPets" {
		t.Errorf("OpID: got %q want listPets", rb.OpID)
	}
}

func TestParseRouteWithGodocIdentPrefix(t *testing.T) {
	// End-to-end: the leading "ListPets" identifier is stripped so the
	// parser sees a normal swagger:route annotation.
	src := `package p

// ListPets swagger:route GET /pets tags listPets
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	rb, ok := b.(*RouteBlock)
	if !ok {
		t.Fatalf("want *RouteBlock, got %T", b)
	}
	if rb.Method != http.MethodGet {
		t.Errorf("Method: got %q want GET", rb.Method)
	}
	if rb.Path != "/pets" {
		t.Errorf("Path: got %q want /pets", rb.Path)
	}
	if rb.OpID != "listPets" {
		t.Errorf("OpID: got %q want listPets", rb.OpID)
	}
	if len(b.Diagnostics()) != 0 {
		t.Errorf("godoc-prefixed route must parse cleanly, got %+v", b.Diagnostics())
	}
}

func TestParseRouteMalformed(t *testing.T) {
	src := `package p

// swagger:route GET
func DoIt() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	if len(b.Diagnostics()) == 0 {
		t.Error("expected at least one diagnostic for malformed route")
	}
	found := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidAnnotation {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("want diagnostic with CodeInvalidAnnotation, got %+v", b.Diagnostics())
	}
}

func TestParseParametersBlock(t *testing.T) {
	src := `package p

// swagger:parameters listPets getPet
type PetParams struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	pb, ok := b.(*ParametersBlock)
	if !ok {
		t.Fatalf("want *ParametersBlock, got %T", b)
	}
	if !slices.Equal(pb.TargetTypes, []string{"listPets", "getPet"}) {
		t.Errorf("TargetTypes: got %v want [listPets getPet]", pb.TargetTypes)
	}
}

func TestParseUnbound(t *testing.T) {
	src := `package p

// A freeform description.
// maximum: 10
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if _, ok := b.(*UnboundBlock); !ok {
		t.Fatalf("want *UnboundBlock, got %T", b)
	}
	if b.AnnotationKind() != AnnUnknown {
		t.Errorf("kind: got %v want AnnUnknown", b.AnnotationKind())
	}
}

func TestParseNilCommentGroup(t *testing.T) {
	b := Parse(nil, token.NewFileSet())
	if _, ok := b.(*UnboundBlock); !ok {
		t.Fatalf("want *UnboundBlock, got %T", b)
	}
	if b.Title() != "" || b.Description() != "" {
		t.Errorf("empty block should have no title/description")
	}
}

// --- title / description ---

func TestParseTitleAndDescription(t *testing.T) {
	// Godoc-style: description first, annotation later.
	src := `package p

// Foo is the primary thing.
//
// It supports the following operations.
// More detail in the second paragraph.
//
// swagger:model Foo
// maximum: 100
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	if b.Title() != "Foo is the primary thing." {
		t.Errorf("Title: got %q", b.Title())
	}
	wantDesc := "It supports the following operations. More detail in the second paragraph."
	if b.Description() != wantDesc {
		t.Errorf("Description:\n got: %q\nwant: %q", b.Description(), wantDesc)
	}
}

// --- properties ---

func TestParseProperties(t *testing.T) {
	src := `package p

// swagger:model Foo
//
// maximum: 100
// minimum: 0
// pattern: ^[a-z]+$
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var names []string
	var values []string
	for p := range b.Properties() {
		names = append(names, p.Keyword.Name)
		values = append(values, p.Value)
	}
	if !slices.Equal(names, []string{"maximum", "minimum", "pattern"}) {
		t.Errorf("property names: got %v", names)
	}
	if !slices.Equal(values, []string{"100", "0", "^[a-z]+$"}) {
		t.Errorf("property values: got %v", values)
	}
}

func TestParseItemsNestedProperty(t *testing.T) {
	src := `package p

// swagger:model Foo
//
// items.maximum: 10
// items.items.minLength: 1
type Foo int
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var depths []int
	for p := range b.Properties() {
		depths = append(depths, p.ItemsDepth)
	}
	if !slices.Equal(depths, []int{1, 2}) {
		t.Errorf("items depths: got %v want [1 2]", depths)
	}
}

func TestParseBlockHeadProperty(t *testing.T) {
	src := `package p

// swagger:meta
//
// consumes:
type Root struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	count := 0
	for p := range b.Properties() {
		count++
		if p.Keyword.Name != "consumes" {
			t.Errorf("keyword: got %q", p.Keyword.Name)
		}
		if p.Value != "" {
			t.Errorf("block-head Value must be empty, got %q", p.Value)
		}
	}
	if count != 1 {
		t.Errorf("property count: got %d want 1", count)
	}
}

// --- YAML fence ---

func TestParseYAMLFenceBalanced(t *testing.T) {
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// responses:
//   200: successResponse
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	count := 0
	for y := range b.YAMLBlocks() {
		count++
		if !slices.Contains(splitLines(y.Text), "responses:") {
			t.Errorf("YAML body missing 'responses:' line:\n%s", y.Text)
		}
	}
	if count != 1 {
		t.Errorf("YAML blocks: got %d want 1", count)
	}
}

func TestParseYAMLFenceUnterminated(t *testing.T) {
	src := `package p

// swagger:operation GET /pets listPets
//
// ---
// responses:
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	foundUnterminated := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeUnterminatedYAML {
			foundUnterminated = true
			break
		}
	}
	if !foundUnterminated {
		t.Errorf("expected CodeUnterminatedYAML diagnostic, got %+v", b.Diagnostics())
	}
	// Body should still be captured up to EOF.
	count := 0
	for range b.YAMLBlocks() {
		count++
	}
	if count != 1 {
		t.Errorf("YAML blocks captured: got %d want 1", count)
	}
}

// --- no-panic guarantee ---

func TestParseDoesNotPanicOnAnythingWeird(t *testing.T) {
	weirdInputs := []string{
		`package p
// swagger:
func Foo() {}`,
		`package p
// swagger:unknownkind
func Foo() {}`,
		`package p
// ---
// ---
// ---
func Foo() {}`,
		`package p
// swagger:model
// swagger:route GET /x y
func Foo() {}`,
	}
	for i, src := range weirdInputs {
		t.Run("", func(t *testing.T) {
			cg, fset := parseCommentGroup(t, src)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("weird input #%d panicked: %v", i, r)
				}
			}()
			_ = Parse(cg, fset) // must not panic
		})
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
