// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"slices"
	"testing"
)

// P2.3: KEYWORD_BLOCK_HEAD tokens collect subsequent TEXT lines as
// Property.Body until the next structured token (per legacy stop S6).

// firstPropertyOf returns the first Property of the parsed block,
// failing the test if the block has none.
func firstPropertyOf(t *testing.T, src string) Property {
	t.Helper()
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)
	for p := range b.Properties() {
		return p
	}
	t.Fatal("block has no properties")
	return Property{}
}

func TestBlockBodyConsumes(t *testing.T) {
	// consumes: with a two-item body → Body = ["application/json", "application/xml"].
	src := `package p

// swagger:meta
//
// consumes:
//   application/json
//   application/xml
type Root struct{}
`
	prop := firstPropertyOf(t, src)
	if prop.Keyword.Name != fixtureBlockKw {
		t.Fatalf("keyword: got %q want consumes", prop.Keyword.Name)
	}
	if !slices.Equal(prop.Body, []string{"application/json", "application/xml"}) {
		t.Errorf("Body: got %q", prop.Body)
	}
}

func TestBlockBodyStopsAtNextKeyword(t *testing.T) {
	// Two block heads in sequence — each captures its own body only.
	src := `package p

// swagger:meta
//
// consumes:
//   application/json
// produces:
//   application/xml
type Root struct{}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var props []Property
	for p := range b.Properties() {
		props = append(props, p)
	}
	if len(props) != 2 {
		t.Fatalf("want 2 properties, got %d", len(props))
	}
	if props[0].Keyword.Name != fixtureBlockKw {
		t.Errorf("prop 0 keyword: got %q", props[0].Keyword.Name)
	}
	if !slices.Equal(props[0].Body, []string{"application/json"}) {
		t.Errorf("prop 0 Body: got %q want [application/json]", props[0].Body)
	}
	if props[1].Keyword.Name != "produces" {
		t.Errorf("prop 1 keyword: got %q", props[1].Keyword.Name)
	}
	if !slices.Equal(props[1].Body, []string{"application/xml"}) {
		t.Errorf("prop 1 Body: got %q want [application/xml]", props[1].Body)
	}
}

func TestBlockBodyStopsAtAnnotation(t *testing.T) {
	// Body collection stops at the boundary — but any annotation
	// would be a separate block anyway, so this mainly exercises
	// the safety case where post-annotation tokens include a stray
	// swagger:* line.
	src := `package p

// swagger:meta
//
// consumes:
//   application/json
//   application/xml
// swagger:ignore
type Root struct{}
`
	prop := firstPropertyOf(t, src)
	if !slices.Equal(prop.Body, []string{"application/json", "application/xml"}) {
		t.Errorf("Body: got %q", prop.Body)
	}
}

func TestBlockBodyStopsAtYAMLFence(t *testing.T) {
	// A fence terminates body collection; fence body is captured
	// independently via YAMLBlocks().
	src := `package p

// swagger:operation GET /pets listPets
//
// consumes:
//   application/json
//
// ---
// responses:
//   200: ok
// ---
func ListPets() {}
`
	cg, fset := parseCommentGroup(t, src)
	b := Parse(cg, fset)

	var props []Property
	for p := range b.Properties() {
		props = append(props, p)
	}
	if len(props) != 1 {
		t.Fatalf("want 1 property (consumes), got %d", len(props))
	}
	if !slices.Equal(props[0].Body, []string{"application/json"}) {
		t.Errorf("consumes Body: got %q", props[0].Body)
	}

	yamlCount := 0
	for range b.YAMLBlocks() {
		yamlCount++
	}
	if yamlCount != 1 {
		t.Errorf("want 1 YAML block, got %d", yamlCount)
	}
}

func TestBlockBodyTrailingBlanksTrimmed(t *testing.T) {
	// Blank lines at the end of the comment group are dropped from
	// the body; internal blanks between body lines are preserved.
	src := "package p\n\n// swagger:meta\n//\n// consumes:\n//   application/json\n//\n//   application/xml\n//\n//\ntype Root struct{}\n"
	prop := firstPropertyOf(t, src)
	want := []string{"application/json", "", "application/xml"}
	if !slices.Equal(prop.Body, want) {
		t.Errorf("Body: got %q want %q", prop.Body, want)
	}
}

func TestBlockBodyNonBlockKeywordsUnaffected(t *testing.T) {
	// A non-block keyword (maximum:) still produces an empty Body.
	src := `package p

// swagger:model Foo
// maximum: 10
type Foo int
`
	prop := firstPropertyOf(t, src)
	if prop.Keyword.Name != fixtureValidationKw {
		t.Fatalf("keyword: got %q", prop.Keyword.Name)
	}
	if len(prop.Body) != 0 {
		t.Errorf("non-block keyword must have nil/empty Body, got %q", prop.Body)
	}
}
