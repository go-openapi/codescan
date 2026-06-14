// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// fixturesTest_Petstore_PetModel mirrors fixtures/goparsing/petstore/models/pet.go
// — a swagger:model with title/description, multiple validations
// (required, pattern, min/maxLength), and several un-annotated fields.
func TestFixtures_Petstore_PetModel(t *testing.T) {
	src := strings.TrimSpace(`
A Pet is the main product in the store.
It is used to describe the animals available in the store.

swagger:model pet
`)
	b := parseString(t, src)
	mb, ok := b.(*ModelBlock)
	require.True(t, ok)
	assert.Equal(t, "pet", mb.Name)
	assert.Equal(t, "A Pet is the main product in the store.", mb.Title())
	assert.Equal(t, "It is used to describe the animals available in the store.", mb.Description())
}

// TestFixtures_Petstore_PetField_RequiredAndPattern mirrors the Name
// field's docstring: required + pattern + min/maxLength, expressed via
// the alias forms ("minimum length", "maximum length") that the v1
// keyword aliases support.
func TestFixtures_Petstore_PetField_RequiredAndPattern(t *testing.T) {
	src := strings.TrimSpace(`
The name of the pet.

required: true
pattern: \w[\w-]+
minimum length: 3
maximum length: 50
`)
	b := parseString(t, src)
	ub, ok := b.(*UnboundBlock)
	require.True(t, ok)

	required, ok := ub.GetBool("required")
	require.True(t, ok)
	assert.True(t, required)

	pat, ok := ub.GetString("pattern")
	require.True(t, ok)
	assert.Equal(t, `\w[\w-]+`, pat)

	minLen, ok := ub.GetInt("minLength")
	require.True(t, ok)
	assert.Equal(t, int64(3), minLen)

	maxLen, ok := ub.GetInt("maxLength")
	require.True(t, ok)
	assert.Equal(t, int64(50), maxLen)

	// Empty diagnostics — every keyword resolves cleanly.
	for _, d := range ub.Diagnostics() {
		assert.NotEqual(t, SeverityError, d.Severity, "unexpected error diagnostic: %s", d)
	}
}

// TestFixtures_Petstore_ItemsPrefix_NestedArrayValidation mirrors the
// PhotoURLs field in fixtures/goparsing/petstore/models/pet.go which
// uses `items pattern: \.(jpe?g|png)$` for per-item validation.
func TestFixtures_Petstore_ItemsPrefix_NestedArrayValidation(t *testing.T) {
	src := strings.TrimSpace(`
The photo urls for the pet.

items pattern: \.(jpe?g|png)$
`)
	b := parseString(t, src)
	ub, ok := b.(*UnboundBlock)
	require.True(t, ok)

	var found bool
	for p := range ub.Properties() {
		if p.Keyword.Name == "pattern" {
			found = true
			assert.Equal(t, 1, p.ItemsDepth)
			assert.Equal(t, `\.(jpe?g|png)$`, p.Value)
		}
	}
	assert.True(t, found)
}

// TestFixtures_Petstore_ItemsPrefix_DeepNesting covers multiple levels
// of items.* prefix accumulation.
func TestFixtures_Petstore_ItemsPrefix_DeepNesting(t *testing.T) {
	src := strings.TrimSpace(`
items.items.items.maxLength: 4
`)
	b := parseString(t, src)
	for p := range b.Properties() {
		assert.Equal(t, "maxLength", p.Keyword.Name)
		assert.Equal(t, 3, p.ItemsDepth)
		assert.Equal(t, int64(4), p.Typed.Integer)
	}
}

// TestFixtures_Petstore_ParametersBlock mirrors PetID with
// `swagger:parameters getPetById deletePet updatePet` — three
// operationID references.
func TestFixtures_Petstore_ParametersBlock(t *testing.T) {
	src := strings.TrimSpace(`
A PetID parameter model.

This is used for operations that want the ID of an pet in the path

swagger:parameters getPetById deletePet updatePet
`)
	b := parseString(t, src)
	pb, ok := b.(*ParametersBlock)
	require.True(t, ok)
	assert.Equal(t, []string{"getPetById", "deletePet", "updatePet"}, pb.OperationIDs)
}

// TestFixtures_Petstore_RouteBlock_GodocPrefixWithDeprecated covers the
// classic `// FooBar swagger:route ...` form plus an inline
// `Deprecated: true` and a `Responses:` raw block.
func TestFixtures_Petstore_RouteBlock_GodocPrefixWithDeprecated(t *testing.T) {
	src := strings.TrimSpace(`
GetPets swagger:route GET /pets pets listPets

Lists the pets known to the store.

By default it will only lists pets that are available for sale.
This can be changed with the status flag.

Deprecated: true
Responses:

	default: genericError
	    200: []pet
`)
	b := parseString(t, src)
	rb, ok := b.(*RouteBlock)
	require.True(t, ok)
	assert.Equal(t, "GET", rb.Method)
	assert.Equal(t, "/pets", rb.Path)
	assert.Equal(t, []string{"pets"}, rb.Tags)
	assert.Equal(t, "listPets", rb.OpID)
	assert.Equal(t, "Lists the pets known to the store.", rb.Title())
	assert.Contains(t, rb.Description(), "available for sale")

	dep, ok := rb.GetBool("deprecated")
	require.True(t, ok)
	assert.True(t, dep)

	respLines, ok := rb.GetList("responses")
	require.True(t, ok)
	joined := strings.Join(respLines, "\n")
	assert.Contains(t, joined, "default: genericError")
	assert.Contains(t, joined, "200: []pet")
}

// TestFixtures_OperationsAnnotation_YAMLBody mirrors
// fixtures/goparsing/classification/operations_annotation/operations.go's
// first operation: a YAML-fenced body holding parameters/responses.
func TestFixtures_OperationsAnnotation_YAMLBody(t *testing.T) {
	src := strings.TrimSpace(`
swagger:operation GET /pets pets getPet

List all pets

---
parameters:
  - name: limit
    in: query
    description: How many items to return at one time (max 100)
    required: false
    type: integer
    format: int32
consumes:
  - "application/json"
  - "application/xml"
produces:
  - "application/xml"
  - "application/json"
responses:
  "200":
    description: An paged array of pets
  default:
    description: unexpected error
---
`)
	b := parseString(t, src)
	ob, ok := b.(*InlineOperationBlock)
	require.True(t, ok)
	assert.Equal(t, "GET", ob.Method)
	assert.Equal(t, "/pets", ob.Path)
	assert.Equal(t, []string{"pets"}, ob.Tags)
	assert.Equal(t, "getPet", ob.OpID)
	// "List all pets" has no trailing punctuation and no internal
	// blank, so heuristics 1/2/3 don't fire — heuristic 4 classifies
	// the whole prose as Description. v1's helpers behaves the same
	// way on the equivalent ProseLines slice.
	assert.Empty(t, ob.Title())
	assert.Equal(t, "List all pets", ob.Description())

	yamls := []RawYAML{}
	for y := range ob.YAMLBlocks() {
		yamls = append(yamls, y)
	}
	require.Len(t, yamls, 1)
	assert.Contains(t, yamls[0].Text, "parameters:")
	assert.Contains(t, yamls[0].Text, "responses:")
	assert.False(t, yamls[0].Truncated)
}

// TestFixtures_Meta_PetstoreV1 mirrors fixtures/goparsing/meta/v1/doc.go
// — the canonical meta block: prose, single-line keywords, raw blocks,
// extensions, info-extensions, security, security-definitions, with
// `swagger:meta` at the *bottom* (godoc convention).
func TestFixtures_Meta_PetstoreV1(t *testing.T) {
	src := `Petstore API.

the purpose of this application is to provide an application
that is using plain go code to define an API

This should demonstrate all the possible comment annotations
that are available to turn go code into a fully compliant swagger 2.0 spec

Terms Of Service:
there are no TOS at this moment, use at your own risk we take no responsibility

	Schemes: http, https
	Host: localhost
	BasePath: /v2
	Version: 0.0.1
	License: MIT http://opensource.org/licenses/MIT
	Contact: John Doe<john.doe@example.com> http://john.doe.com

	Consumes:
	- application/json
	- application/xml

	Produces:
	- application/json
	- application/xml

	Extensions:
	x-meta-value: value
	x-meta-array:
	  - value1
	  - value2

	InfoExtensions:
	x-info-value: value

	Security:
	- api_key:

	SecurityDefinitions:
	api_key:
	     type: apiKey
	     name: KEY
	     in: header

swagger:meta`
	b := parseString(t, src)
	mb, ok := b.(*MetaBlock)
	require.True(t, ok, "expected *MetaBlock, got %T", b)
	assert.Equal(t, "Petstore API.", mb.Title())

	// Single-line keywords.
	host, ok := mb.GetString("host")
	require.True(t, ok)
	assert.Equal(t, "localhost", host)

	basePath, ok := mb.GetString("basePath")
	require.True(t, ok)
	assert.Equal(t, "/v2", basePath)

	version, ok := mb.GetString("version")
	require.True(t, ok)
	assert.Equal(t, "0.0.1", version)

	schemes, ok := mb.GetList("schemes")
	require.True(t, ok)
	assert.Equal(t, []string{"http", "https"}, schemes)

	// Raw blocks present.
	consumes, ok := mb.GetList("consumes")
	require.True(t, ok)
	joined := strings.Join(consumes, "\n")
	assert.Contains(t, joined, "application/json")
	assert.Contains(t, joined, "application/xml")

	// Extensions: top-level x-* entries surfaced.
	exts := []Extension{}
	for e := range mb.Extensions() {
		exts = append(exts, e)
	}
	assert.NotEmpty(t, exts)
	hasXMeta := false
	for _, e := range exts {
		if e.Name == "x-meta-value" {
			hasXMeta = true
			assert.Equal(t, "value", e.Value)
		}
	}
	assert.True(t, hasXMeta)

	// Security + securityDefinitions raw blocks present as Properties.
	sec, ok := mb.GetList("security")
	require.True(t, ok)
	assert.Contains(t, strings.Join(sec, "\n"), "api_key")

	secDef, ok := mb.GetList("securityDefinitions")
	require.True(t, ok)
	assert.Contains(t, strings.Join(secDef, "\n"), "type: apiKey")
}

// TestFixtures_Meta_TagsBlock pins the meta `Tags:` raw block
// (go-swagger#2655): the body is a YAML sequence of tag objects whose
// nested `externalDocs:` mapping — itself a meta-family keyword —
// must be absorbed as body text via the YAML-bodied indentation
// override, not terminate the block. A following sibling keyword at
// the same indent as `Tags:` still terminates it.
func TestFixtures_Meta_TagsBlock(t *testing.T) {
	src := "\tTags:\n" +
		"\t- name: pet\n" +
		"\t  description: Everything about your Pets\n" +
		"\t  externalDocs:\n" +
		"\t    description: Find out more\n" +
		"\t    url: http://swagger.io\n" +
		"\t- name: store\n" +
		"\tVersion: 1.0.0\n" +
		"\nswagger:meta"

	b := parseString(t, src)
	mb, ok := b.(*MetaBlock)
	require.True(t, ok, "expected *MetaBlock, got %T", b)

	// The sibling `Version:` at the same indent terminated the block.
	version, ok := mb.GetString("version")
	require.True(t, ok, "Version sibling must terminate the Tags block")
	assert.Equal(t, "1.0.0", version)

	// The raw Tags body preserves per-line indentation: the nested
	// externalDocs mapping is absorbed (indentation override) and the
	// list markers/depth survive, so the downstream YAML list parses.
	var tagsBody string
	var found bool
	for p := range mb.Properties() {
		if p.Keyword.Name == KwTags {
			tagsBody = p.Body
			found = true
		}
	}
	require.True(t, found, "Tags block must be present")
	assert.Contains(t, tagsBody, "- name: pet")
	assert.Contains(t, tagsBody, "  externalDocs:")
	assert.Contains(t, tagsBody, "    url: http://swagger.io")
	assert.Contains(t, tagsBody, "- name: store")
	assert.NotContains(t, tagsBody, "Version", "sibling must not bleed into the body")
	assert.Empty(t, b.Diagnostics())
}

// TestFixtures_Meta_TosKeywordVariants exercises the trailing-dot,
// alias spelling ("Terms Of Service" / "TermsOfService" / "tos") that
// the meta v3 / v4 fixtures show.
func TestFixtures_Meta_TosKeywordVariants(t *testing.T) {
	cases := []string{
		"Terms Of Service:\nuse at your own risk\nswagger:meta",
		"TermsOfService:\nuse at your own risk\nswagger:meta",
		"tos:\nuse at your own risk\nswagger:meta",
	}
	for _, src := range cases {
		b := parseString(t, src)
		_, ok := b.(*MetaBlock)
		require.True(t, ok, "expected MetaBlock for %q", src)
		tos, ok := b.GetList("tos")
		require.True(t, ok, "expected tos block to be present (%q)", src)
		assert.Contains(t, strings.Join(tos, "\n"), "use at your own risk")
	}
}

// TestFixtures_Petstore_ResponseBlock with body marker.
func TestFixtures_Petstore_ResponseBlock(t *testing.T) {
	src := strings.TrimSpace(`
A GenericError is the default error message that is generated.
For certain status codes there are more appropriate error structures.

swagger:response genericError
`)
	b := parseString(t, src)
	rb, ok := b.(*ResponseBlock)
	require.True(t, ok)
	assert.Equal(t, "genericError", rb.Name)
	assert.Equal(t, "A GenericError is the default error message that is generated.", rb.Title())
	assert.Equal(t, "For certain status codes there are more appropriate error structures.", rb.Description())
}

// TestFixtures_Petstore_StrfmtAnnotation_FieldLevel mirrors the
// Birthday field's strfmt tag.
func TestFixtures_Petstore_StrfmtAnnotation_FieldLevel(t *testing.T) {
	src := strings.TrimSpace(`
The pet's birthday

swagger:strfmt date
`)
	b := parseString(t, src)
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok)
	assert.Equal(t, AnnStrfmt, cb.AnnotationKind())
	require.Len(t, cb.Args, 1)
	assert.Equal(t, "date", cb.Args[0].Text)
	assert.Equal(t, TokenIdentName, cb.Args[0].Kind)
}

// TestFixtures_Petstore_ParameterIn covers the `in:` keyword inside an
// UnboundBlock (a parameter struct field).
func TestFixtures_Petstore_ParameterIn(t *testing.T) {
	src := strings.TrimSpace(`
The ID of the pet

in: path
required: true
`)
	b := parseString(t, src)
	in, ok := b.GetString("in")
	require.True(t, ok)
	assert.Equal(t, "path", in)

	required, ok := b.GetBool("required")
	require.True(t, ok)
	assert.True(t, required)
}

// TestFixtures_AllowedHTTPMethods covers the closed HTTP method
// vocabulary inspired by fixtures/enhancements/all-http-methods.
func TestFixtures_AllowedHTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}
	for _, m := range methods {
		src := "swagger:route " + m + " /thing tag op" + m
		b := parseString(t, src)
		rb, ok := b.(*RouteBlock)
		require.True(t, ok, "expected RouteBlock for method %q", m)
		assert.Equal(t, m, rb.Method)
		assert.Empty(t, b.Diagnostics(), "method %q produced diagnostics: %v", m, b.Diagnostics())
	}
}

// TestFixtures_AllowedHTTPMethods_LowercaseNormalised checks
// case-insensitive HTTP method matching.
func TestFixtures_AllowedHTTPMethods_LowercaseNormalised(t *testing.T) {
	b := parseString(t, "swagger:route get /pets pets listPets")
	rb, ok := b.(*RouteBlock)
	require.True(t, ok)
	assert.Equal(t, "GET", rb.Method, "method canonical form is upper case")
}

// TestFixtures_GodocLinterTrailingDot covers the trailing-dot elision
// the godot linter triggers across annotation lines.
func TestFixtures_GodocLinterTrailingDot(t *testing.T) {
	cases := []struct {
		src      string
		want     string
		annKind  AnnotationKind
		argCount int
	}{
		{"swagger:strfmt uuid.", "uuid", AnnStrfmt, 1},
		{"swagger:meta.", "", AnnMeta, 0},
		{"swagger:model Pet.", "Pet", AnnModel, 1},
	}
	for _, tc := range cases {
		b := parseString(t, tc.src)
		assert.EqualT(t, tc.annKind, b.AnnotationKind(), "in: %s", tc.src)
		switch tc.annKind {
		case AnnStrfmt:
			cb, ok := b.(*ClassifierBlock)
			require.TrueT(t, ok)
			require.Len(t, cb.Args, tc.argCount)
			assert.Equal(t, tc.want, cb.Args[0].Text)
		case AnnModel:
			mb, ok := b.(*ModelBlock)
			require.TrueT(t, ok)
			assert.Equal(t, tc.want, mb.Name)
		case AnnMeta:
			_, ok := b.(*MetaBlock)
			require.True(t, ok)
		default:
			require.FailNow(t, "test configuration error: missing assertion")
		}
	}
}

// TestFixtures_CRLFNormalisation ensures \r\n line endings produce the
// same token stream as \n.
func TestFixtures_CRLFNormalisation(t *testing.T) {
	const src = "swagger:model Pet\r\nrequired: true\r\nmaxLength: 5"

	b := parseString(t, src)
	mb, ok := b.(*ModelBlock)
	require.True(t, ok)
	assert.Equal(t, "Pet", mb.Name)

	required, ok := mb.GetBool("required")
	require.True(t, ok)
	assert.True(t, required)

	maxLen, ok := mb.GetInt("maxLength")
	require.True(t, ok)
	assert.Equal(t, int64(5), maxLen)
}

// TestFixtures_ComparisonOperatorOnNumber covers `maximum: <5` v1 form.
func TestFixtures_ComparisonOperatorOnNumber(t *testing.T) {
	src := strings.TrimSpace(`
swagger:model Foo

maximum: <=10
minimum: >0
`)
	b := parseString(t, src)
	for p := range b.Properties() {
		switch p.Keyword.Name {
		case "maximum":
			assert.Equal(t, "<=", p.Typed.Op)
			assert.InDelta(t, 10.0, p.Typed.Number, 0)
		case "minimum":
			assert.Equal(t, ">", p.Typed.Op)
			assert.InDelta(t, 0.0, p.Typed.Number, 0)
		}
	}
}

// TestFixtures_AllOf_OptionalClassName covers swagger:allOf with and
// without the polymorphic class name (mirrors the docs examples in
// 23-classifier-grammar.md).
func TestFixtures_AllOf_OptionalClassName(t *testing.T) {
	bare := parseString(t, "swagger:allOf")
	cbBare, ok := bare.(*ClassifierBlock)
	require.True(t, ok)
	assert.Empty(t, cbBare.Args)
	assert.Empty(t, bare.Diagnostics())

	named := parseString(t, "swagger:allOf Animal")
	cbNamed, ok := named.(*ClassifierBlock)
	require.True(t, ok)
	require.Len(t, cbNamed.Args, 1)
	assert.Equal(t, "Animal", cbNamed.Args[0].Text)
}

// TestFixtures_DefaultAnnotation_JSONForms covers the JsonValue branch
// (objects, arrays, numbers, booleans) and the RawValue fallback.
func TestFixtures_DefaultAnnotation_JSONForms(t *testing.T) {
	jsonCases := []string{
		`swagger:default {"limit": 10, "offset": 0}`,
		`swagger:default [1,2,3]`,
		`swagger:default 42`,
		`swagger:default true`,
		`swagger:default null`,
		`swagger:default "literal-string"`,
	}
	for _, src := range jsonCases {
		b := parseString(t, src)
		cb, ok := b.(*ClassifierBlock)
		require.True(t, ok, "expected classifier for %q, got %T", src, b)
		require.Len(t, cb.Args, 1, src)
		assert.Equal(t, TokenJSONValue, cb.Args[0].Kind, "%q", src)
	}

	// Bare ident → falls back to RAW_VALUE.
	rawSrc := "swagger:default high"
	b := parseString(t, rawSrc)
	cb, ok := b.(*ClassifierBlock)
	require.TrueT(t, ok)
	require.Len(t, cb.Args, 1)
	assert.Equal(t, TokenRawValue, cb.Args[0].Kind)
}

// TestFixtures_EnumDecl_BracketedHybrid covers the hybrid-list example
// from 23-classifier-grammar.md ("a, {x:1}, c, [1,2,3], …").
func TestFixtures_EnumDecl_BracketedHybrid(t *testing.T) {
	b := parseString(t, `swagger:enum my_enum [a, {"x":1, "y":[1,2,3]}, c, [1,2,3], ["u","v"]]`)
	eb, ok := b.(*EnumDeclBlock)
	require.True(t, ok)
	assert.Equal(t, "my_enum", eb.Name)
	assert.Equal(t, enumFormNamePlusBracketed, eb.InlineForm)
	require.Len(t, eb.InlineArgs, 1)
	assert.Equal(t, TokenJSONValue, eb.InlineArgs[0].Kind)
}

// TestFixtures_EnumDecl_MultilineBody backports the Q15 multi-line
// value-list body shape on swagger:enum.
func TestFixtures_EnumDecl_MultilineBody(t *testing.T) {
	src := strings.TrimSpace(`
swagger:enum Priority

enum:
  - low
  - medium
  - high
`)
	b := parseString(t, src)
	eb, ok := b.(*EnumDeclBlock)
	require.True(t, ok)
	assert.Equal(t, "Priority", eb.Name)
	assert.NotEmpty(t, eb.BodyValues)
	assert.Contains(t, eb.BodyValues, "low")
	assert.Contains(t, eb.BodyValues, "medium")
	assert.Contains(t, eb.BodyValues, "high")
}

// TestFixtures_BookingMeta_LeadingAnnotation covers the
// `swagger:meta` placed at the top of the comment group.
func TestFixtures_BookingMeta_LeadingAnnotation(t *testing.T) {
	src := strings.TrimSpace(`
swagger:meta

Schemes: http, https
Host: api.example.com
BasePath: /v2
Version: 1.4.0
License: MIT https://opensource.org/licenses/MIT
Contact: API Team team@example.com
`)
	b := parseString(t, src)
	mb, ok := b.(*MetaBlock)
	require.True(t, ok)

	assert.Equal(t, AnnMeta, mb.AnnotationKind())

	v, ok := mb.GetString("version")
	require.True(t, ok)
	assert.Equal(t, "1.4.0", v)
}

// TestFixtures_OperationDeprecatedAndExternalDocs covers the cross-
// over deprecated keyword + externalDocs raw block under an inline
// operation block.
func TestFixtures_OperationDeprecatedAndExternalDocs(t *testing.T) {
	src := strings.TrimSpace(`
swagger:operation GET /pets pets listPets

Lists pets.

deprecated: true
externalDocs:
  description: User Guide
  url: https://example.com/docs
`)
	b := parseString(t, src)
	ob, ok := b.(*InlineOperationBlock)
	require.True(t, ok)

	dep, ok := ob.GetBool("deprecated")
	require.True(t, ok)
	assert.True(t, dep)

	docs, ok := ob.GetList("externalDocs")
	require.True(t, ok)
	joined := strings.Join(docs, "\n")
	assert.Contains(t, joined, "description: User Guide")
	assert.Contains(t, joined, "url: https://example.com/docs")
}

// TestFixtures_DecorativeYAMLFenceInExtensions confirms decorative
// `--- … ---` fences around an extensions: body produce byte-identical
// behaviour with and without the fences (the v1 quirk noted in
// 10-shared.md and 40-lexer.md §5).
func TestFixtures_DecorativeYAMLFenceInExtensions(t *testing.T) {
	withFence := strings.TrimSpace(`
swagger:meta

Extensions:
---
x-foo: bar
x-baz: 1
---
`)
	withoutFence := strings.TrimSpace(`
swagger:meta

Extensions:
x-foo: bar
x-baz: 1
`)

	b1 := parseString(t, withFence)
	b2 := parseString(t, withoutFence)

	exts1 := collectExtensionsAsMap(b1)
	exts2 := collectExtensionsAsMap(b2)
	assert.Equal(t, exts2, exts1, "decorative fence should be transparent")
}

// TestFixtures_BlockBodyCrossover_KeywordsAcrossSiblings confirms
// `Consumes:` and `Produces:` consecutive raw blocks each terminate at
// the next sibling structural keyword, not at blank lines.
func TestFixtures_BlockBodyCrossover_KeywordsAcrossSiblings(t *testing.T) {
	src := strings.TrimSpace(`
swagger:meta

Consumes:
- application/json

- application/xml

Produces:
- application/json
`)
	b := parseString(t, src)
	cons, ok := b.GetList("consumes")
	require.True(t, ok)
	consJoined := strings.Join(cons, "\n")
	assert.Contains(t, consJoined, "application/json")
	assert.Contains(t, consJoined, "application/xml", "blank lines do not terminate a raw block body")

	prod, ok := b.GetList("produces")
	require.True(t, ok)
	assert.Contains(t, strings.Join(prod, "\n"), "application/json")
}

// TestFixtures_FullPipeline_FromCommentGroup exercises the public
// Parse(*ast.CommentGroup, *token.FileSet) entry — same as scanner does.
func TestFixtures_FullPipeline_FromCommentGroup(t *testing.T) {
	src := `package fake

// A Pet is the main product in the store.
//
// swagger:model pet
type Pet struct {
	// The name of the pet.
	//
	// required: true
	// pattern: \w+
	Name string ` + "`json:\"name\"`" + `
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fake.go", src, parser.ParseComments)
	require.NoError(t, err)

	// Find each declared type's documentation comment and parse it.
	var got []Block
	ast.Inspect(file, func(n ast.Node) bool {
		switch d := n.(type) {
		case *ast.GenDecl:
			if d.Doc != nil {
				got = append(got, Parse(d.Doc, fset))
			}
		case *ast.Field:
			if d.Doc != nil {
				got = append(got, Parse(d.Doc, fset))
			}
		}
		return true
	})

	require.GreaterOrEqual(t, len(got), 2)
	mb, ok := got[0].(*ModelBlock)
	require.True(t, ok, "expected first decl to be ModelBlock, got %T", got[0])
	assert.Equal(t, "pet", mb.Name)

	ub, ok := got[1].(*UnboundBlock)
	require.True(t, ok, "expected field doc to be UnboundBlock, got %T", got[1])
	required, ok := ub.GetBool("required")
	require.True(t, ok)
	assert.True(t, required)
	pat, ok := ub.GetString("pattern")
	require.True(t, ok)
	assert.Equal(t, `\w+`, pat)
}

// TestFixtures_CrossSchemeKeyword_Schemes confirms the `schemes:`
// keyword is legal under meta, route, and operation but warns
// elsewhere (e.g. under model).
func TestFixtures_CrossSchemeKeyword_Schemes(t *testing.T) {
	cases := []struct {
		src      string
		wantWarn bool
		comment  string
	}{
		{"swagger:meta\n\nSchemes: http", false, "meta"},
		{"swagger:route GET /pets pets listPets\n\nSchemes: http", false, "route"},
		{"swagger:operation GET /pets pets listPets\n\nSchemes: http", false, "operation"},
		{"swagger:model Foo\n\nSchemes: http", true, "model"},
	}
	for _, tc := range cases {
		b := parseString(t, tc.src)
		hasContextWarn := false
		for _, d := range b.Diagnostics() {
			if d.Code == CodeContextInvalid {
				hasContextWarn = true
			}
		}
		if tc.wantWarn {
			assert.True(t, hasContextWarn, "%s: expected context-invalid diagnostic", tc.comment)
		} else {
			assert.False(t, hasContextWarn, "%s: unexpected context-invalid diagnostic", tc.comment)
		}
	}
}

// TestFixtures_StrfmtKeywordAlias_MaxLen confirms that the alias
// "max len" / "max-len" / "maxLen" are equivalent to "maxLength".
func TestFixtures_StrfmtKeywordAlias_MaxLen(t *testing.T) {
	for _, alias := range []string{"max len", "max-len", "maxLen", "maximum length", "maximumLength"} {
		src := alias + ": 42"
		b := parseString(t, src)
		v, ok := b.GetInt("maxLength")
		require.True(t, ok, "alias %q did not resolve to maxLength", alias)
		assert.Equal(t, int64(42), v)
	}
}

// TestFixtures_DecimalNumberValue_Roundtrip confirms NUMBER_VALUE
// parses signed/unsigned decimals and fractional values.
func TestFixtures_DecimalNumberValue_Roundtrip(t *testing.T) {
	cases := []struct {
		src  string
		op   string
		want float64
	}{
		{"maximum: 10", "", 10},
		{"maximum: 3.14", "", 3.14},
		{"maximum: <-1", "<", -1},
		{"maximum: >=2.5", ">=", 2.5},
	}
	for _, tc := range cases {
		b := parseString(t, tc.src)
		var found bool
		for p := range b.Properties() {
			if p.Keyword.Name == "maximum" {
				found = true
				assert.Equal(t, tc.op, p.Typed.Op)
				assert.InDelta(t, tc.want, p.Typed.Number, 1e-9)
			}
		}
		assert.True(t, found, "%q produced no maximum property", tc.src)
	}
}

// collectExtensionsAsMap turns the iter.Seq[Extension] into a map for
// equality comparison. Extension.Value is YAML-typed (`any`); callers
// that only care about presence/equality compare on the typed value.
func collectExtensionsAsMap(b Block) map[string]any {
	out := map[string]any{}
	for e := range b.Extensions() {
		out[e.Name] = e.Value
	}
	return out
}
