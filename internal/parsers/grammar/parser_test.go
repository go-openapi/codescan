// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/token"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// parseString runs the full parser on a // -less comment block.
//
//nolint:ireturn // this test helper is here precisely to mock the interface.
func parseString(t *testing.T, src string) Block {
	t.Helper()
	pos := token.Position{Filename: "test.go", Line: 1, Column: 1}
	return (&DefaultParser{}).ParseText(src, pos)
}

func TestParser_ModelBlock_NameFromAnnotation(t *testing.T) {
	b := parseString(t, "swagger:model Pet")
	mb, ok := b.(*ModelBlock)
	require.True(t, ok, "expected *ModelBlock, got %T", b)
	assert.Equal(t, "Pet", mb.Name)
	assert.Equal(t, AnnModel, mb.AnnotationKind())
}

func TestParser_ResponseBlock_OptionalName(t *testing.T) {
	b1 := parseString(t, "swagger:response petResp")
	rb1, ok := b1.(*ResponseBlock)
	require.True(t, ok)
	assert.Equal(t, "petResp", rb1.Name)

	b2 := parseString(t, "swagger:response")
	rb2, ok := b2.(*ResponseBlock)
	require.True(t, ok)
	assert.Empty(t, rb2.Name)

	// `swagger:response *` is a synonym for the bare form: no name (the builder keys it by the type
	// name), no diagnostics.
	b3 := parseString(t, "swagger:response *")
	rb3, ok := b3.(*ResponseBlock)
	require.True(t, ok)
	assert.Empty(t, rb3.Name)
	assert.Empty(t, b3.Diagnostics())
}

func TestParser_NameBlock_CapturesIdentArg(t *testing.T) {
	b := parseString(t, "swagger:name jsonFieldName")
	nb, ok := b.(*NameBlock)
	require.True(t, ok, "expected *NameBlock, got %T", b)
	assert.Equal(t, "jsonFieldName", nb.Name)
	assert.Equal(t, AnnName, nb.AnnotationKind())
	assert.Empty(t, b.Diagnostics())
}

func TestParser_NameBlock_MissingArgEmitsDiagnostic(t *testing.T) {
	b := parseString(t, "swagger:name")
	nb, ok := b.(*NameBlock)
	require.True(t, ok, "expected *NameBlock, got %T", b)
	assert.Empty(t, nb.Name)
	require.NotEmpty(t, b.Diagnostics())
	assert.Equal(t, CodeMissingRequiredArg, b.Diagnostics()[0].Code)
}

// parseAllString runs ParseAll on a // -less comment block.
func parseAllString(t *testing.T, src string) []Block {
	t.Helper()
	pos := token.Position{Filename: "test.go", Line: 1, Column: 1}
	lines := preprocessText(src, pos)
	tokens := Lex(lines)
	return (&DefaultParser{}).parseAllTokens(tokens)
}

func TestParser_SingleLineCommentAsDescription(t *testing.T) {
	pos := token.Position{Filename: "test.go", Line: 1, Column: 1}
	parse := func(src string, on bool) Block {
		return NewParser(token.NewFileSet(),
			WithSingleLineCommentAsDescription(on)).ParseText(src, pos)
	}

	// Single-line title-shaped comment: default keeps it as title; the option moves it to the
	// description.
	const single = "A one-line comment.\n\nswagger:model Pet"
	def := parse(single, false)
	assert.Equal(t, "A one-line comment.", def.Title())
	assert.Empty(t, def.Description())

	on := parse(single, true)
	assert.Empty(t, on.Title(), "single-line comment is no longer a title")
	assert.Equal(t, "A one-line comment.", on.Description())

	// Multi-line comment: the title/description split is preserved in both modes — the option only
	// affects single-line comments.
	const multi = "Title line.\n\nDescription body.\n\nswagger:model Pet"
	multiOn := parse(multi, true)
	assert.Equal(t, "Title line.", multiOn.Title(), "multi-line title unchanged by the option")
	assert.Equal(t, "Description body.", multiOn.Description())
}

func TestParser_ParseAll_SingleAnnotation(t *testing.T) {
	blocks := parseAllString(t, "swagger:model Pet")
	require.Len(t, blocks, 1)
	mb, ok := blocks[0].(*ModelBlock)
	require.True(t, ok)
	assert.Equal(t, "Pet", mb.Name)
}

func TestParser_ParseAll_NoAnnotation(t *testing.T) {
	blocks := parseAllString(t, "Just a docstring.")
	require.Len(t, blocks, 1)
	_, ok := blocks[0].(*UnboundBlock)
	require.True(t, ok)
	assert.Equal(t, "Just a docstring.", blocks[0].Title())
}

func TestParser_ParseAll_TwoAnnotations(t *testing.T) {
	src := `Pet model documentation.

swagger:model Pet
swagger:strfmt date-time`
	blocks := parseAllString(t, src)
	require.Len(t, blocks, 2)

	// First block: ModelBlock owns the pre-annotation prose.
	mb, ok := blocks[0].(*ModelBlock)
	require.True(t, ok, "expected blocks[0] *ModelBlock, got %T", blocks[0])
	assert.Equal(t, "Pet", mb.Name)
	assert.Equal(t, "Pet model documentation.", mb.PreambleTitle())

	// Second block: ClassifierBlock for swagger:strfmt.
	cb, ok := blocks[1].(*ClassifierBlock)
	require.True(t, ok, "expected blocks[1] *ClassifierBlock, got %T", blocks[1])
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, "date-time", arg)
}

func TestParser_ParseAll_ThreeAnnotations(t *testing.T) {
	src := `swagger:model Bag
swagger:strfmt
swagger:ignore`
	blocks := parseAllString(t, src)
	require.Len(t, blocks, 3)

	_, ok := blocks[0].(*ModelBlock)
	require.True(t, ok, "expected blocks[0] *ModelBlock, got %T", blocks[0])
	cb1, ok := blocks[1].(*ClassifierBlock)
	require.True(t, ok)
	assert.Equal(t, AnnStrfmt, cb1.AnnotationKind())
	cb2, ok := blocks[2].(*ClassifierBlock)
	require.True(t, ok)
	assert.Equal(t, AnnIgnore, cb2.AnnotationKind())
}

func TestBlock_AnnotationArg(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantArg string
		wantOK  bool
	}{
		{"model with name", "swagger:model Pet", "Pet", true},
		{"bare model", "swagger:model", "", false},
		{"response with name", "swagger:response petResp", "petResp", true},
		{"bare response", "swagger:response", "", false},
		{"name", "swagger:name jsonField", "jsonField", true},
		{"strfmt", "swagger:strfmt date-time", "date-time", true},
		{"bare strfmt", "swagger:strfmt", "", false},
		{"additionalProperties bool", "swagger:additionalProperties true", "true", true},
		{"additionalProperties type", "swagger:additionalProperties Thing", "Thing", true},
		{"additionalProperties array", "swagger:additionalProperties []integer", "[]integer", true},
		{"ignore", "swagger:ignore", "", false},
		{"alias", "swagger:alias", "", false},
		{"unbound prose", "Just a docstring.", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := parseString(t, tc.src)
			arg, ok := b.AnnotationArg()
			assert.Equal(t, tc.wantArg, arg)
			assert.Equal(t, tc.wantOK, ok)
		})
	}
}

func TestParser_AdditionalPropertiesBlock_IsClassifierWithArg(t *testing.T) {
	b := parseString(t, "swagger:additionalProperties Thing")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnAdditionalProperties, cb.AnnotationKind())
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, "Thing", arg)
}

func TestAnnotationKind_AdditionalProperties_RoundTrip(t *testing.T) {
	assert.Equal(t, "additionalProperties", AnnAdditionalProperties.String())
	assert.Equal(t, AnnAdditionalProperties, AnnotationKindFromName("additionalProperties"))
}

func TestAnnotationKind_PatternProperties_RoundTrip(t *testing.T) {
	assert.Equal(t, "patternProperties", AnnPatternProperties.String())
	assert.Equal(t, AnnPatternProperties, AnnotationKindFromName("patternProperties"))
}

func TestParser_PatternPropertiesBlock_CapturesRawPairList(t *testing.T) {
	// The whole `"<re>": <spec>, …` remainder is captured verbatim as one arg (it contains
	// spaces/colons/commas the builder parses).
	b := parseString(t, `swagger:patternProperties "^x-": string, "^\d+$": integer`)
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnPatternProperties, cb.AnnotationKind())
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, `"^x-": string, "^\d+$": integer`, arg)
}

func TestAnnotationKind_Title_RoundTrip(t *testing.T) {
	assert.Equal(t, "title", AnnTitle.String())
	assert.Equal(t, AnnTitle, AnnotationKindFromName("title"))
}

func TestAnnotationKind_Description_RoundTrip(t *testing.T) {
	assert.Equal(t, "description", AnnDescription.String())
	assert.Equal(t, AnnDescription, AnnotationKindFromName("description"))
}

func TestParser_TitleOverride_CapturesWholeLine(t *testing.T) {
	// The whole rest of the line is the title (a sentence with spaces).
	b := parseString(t, "swagger:title A Foo Widget")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnTitle, cb.AnnotationKind())
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, "A Foo Widget", arg)
	assert.Empty(t, cb.Diagnostics())
}

func TestParser_DescriptionOverride_CapturesWholeLine(t *testing.T) {
	b := parseString(t, "swagger:description A foo widget exposed via the public API.")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnDescription, cb.AnnotationKind())
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, "A foo widget exposed via the public API.", arg)
	assert.Empty(t, cb.Diagnostics())
}

func TestParser_DescriptionOverride_BareIsWellFormed(t *testing.T) {
	// A bare swagger:description / swagger:title is well-formed grammar (no parse diagnostic): the
	// empty value is the deliberate godoc-suppression affordance.
	// The emptiness *warning* (scan.empty-override) is the builder consumption point's job, not the
	// parser's (design D7 / §4).
	for _, src := range []string{"swagger:description", "swagger:title", "swagger:description   "} {
		b := parseString(t, src)
		assert.Emptyf(t, b.Diagnostics(), "%q is well-formed → no parser diagnostic", src)
		arg, hasArg := b.AnnotationArg()
		assert.Falsef(t, hasArg, "%q has no arg", src)
		assert.Emptyf(t, arg, "%q arg is empty", src)
	}
}

func TestParser_DescriptionOverride_MultiLineBody(t *testing.T) {
	// Option B: lines following swagger:description fold into the description (blank-line / keyword /
	// annotation / EOF terminated), joined with "\n".
	b := parseString(t, "swagger:description First line of the description.\nSecond line continues it.")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnDescription, cb.AnnotationKind())
	arg, hasArg := cb.AnnotationArg()
	require.True(t, hasArg)
	assert.Equal(t, "First line of the description.\nSecond line continues it.", arg)
}

func TestParser_DescriptionOverride_BodyStopsAtBlankAndKeyword(t *testing.T) {
	// A blank line ends the body; a following keyword is not swallowed.
	b := parseString(t, "swagger:description The value.\nmaximum: 100")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	arg, _ := cb.AnnotationArg()
	assert.Equal(t, "The value.", arg, "the maximum: keyword must not fold into the description")

	// Bare head + body-only (no inline) folds the body as the whole value.
	b2 := parseString(t, "swagger:description\nBody only, no inline head.")
	arg2, hasArg2 := b2.AnnotationArg()
	require.True(t, hasArg2)
	assert.Equal(t, "Body only, no inline head.", arg2)
}

func TestParser_DescriptionOverride_CoexistsWithKeywords(t *testing.T) {
	// description/title dispatch through the schema family (like swagger:name), so a co-located
	// validation keyword surfaces as a Property rather than being rejected as context-invalid under a
	// classifier block.
	b := parseString(t, "swagger:description The value.\nmaximum: 100")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok, "expected *ClassifierBlock, got %T", b)
	assert.Equal(t, AnnDescription, cb.AnnotationKind())
	arg, _ := cb.AnnotationArg()
	assert.Equal(t, "The value.", arg)
	assert.Empty(t, cb.Diagnostics(), "the co-located keyword must not be rejected")

	var hasMax bool
	for p := range cb.Properties() {
		if p.Keyword.Name == KwMaximum {
			hasMax = true
			assert.Equal(t, "100", p.Value)
		}
	}
	assert.True(t, hasMax, "maximum: must surface as a Property")
}

func TestParser_ParametersBlock_RequiresAtLeastOneArg(t *testing.T) {
	b := parseString(t, "swagger:parameters listPets getPet")
	pb, ok := b.(*ParametersBlock)
	require.True(t, ok)
	assert.Equal(t, ParamTargetOperations, pb.Target)
	assert.Equal(t, []string{"listPets", "getPet"}, pb.OperationIDs())
	assert.Empty(t, b.Diagnostics())

	bad := parseString(t, "swagger:parameters")
	pbad, ok := bad.(*ParametersBlock)
	require.True(t, ok)
	assert.Empty(t, pbad.OperationIDs())
	require.NotEmpty(t, bad.Diagnostics())
	assert.Equal(t, CodeMissingRequiredArg, bad.Diagnostics()[0].Code)
}

func TestParser_ParametersBlock_Targets(t *testing.T) {
	t.Run("shared register-only", func(t *testing.T) {
		pb, ok := parseString(t, "swagger:parameters *").(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, ParamTargetShared, pb.Target)
		assert.Empty(t, pb.OperationIDs())
		assert.Empty(t, pb.Args)
		assert.Empty(t, pb.Diagnostics())
	})

	t.Run("shared register plus operation ids", func(t *testing.T) {
		pb, ok := parseString(t, "swagger:parameters * listPets createPet").(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, ParamTargetShared, pb.Target)
		assert.Equal(t, []string{"listPets", "createPet"}, pb.Args)
		assert.Nil(t, pb.OperationIDs()) // not an operations target
	})

	t.Run("path target", func(t *testing.T) {
		pb, ok := parseString(t, "swagger:parameters /pets").(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, ParamTargetPath, pb.Target)
		assert.Equal(t, "/pets", pb.Path)
		assert.Empty(t, pb.Args)
	})

	t.Run("path reference with shared name", func(t *testing.T) {
		pb, ok := parseString(t, "swagger:parameters /pets X-Request-ID").(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, ParamTargetPath, pb.Target)
		assert.Equal(t, "/pets", pb.Path)
		assert.Equal(t, []string{"X-Request-ID"}, pb.Args)
	})

	t.Run("operation reference with shared name (dashes preserved)", func(t *testing.T) {
		pb, ok := parseString(t, "swagger:parameters listPets X-Request-ID").(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, ParamTargetOperations, pb.Target)
		assert.Equal(t, []string{"listPets", "X-Request-ID"}, pb.Args)
	})

	t.Run("duplicate argument dropped + recorded", func(t *testing.T) {
		dup := "createThing"
		pb, ok := parseString(t, "swagger:parameters * "+dup+" "+dup).(*ParametersBlock)
		require.True(t, ok)
		assert.Equal(t, []string{"createThing"}, pb.Args)
		assert.Equal(t, []string{"createThing"}, pb.Dups)
	})
}

func TestParser_RouteBlock_BasicArgs(t *testing.T) {
	src := `Lists pets.

swagger:route GET /pets pets listPets

Consumes:
  - application/json

Produces:
  - application/json`
	b := parseString(t, src)
	rb, ok := b.(*RouteBlock)
	require.True(t, ok, "expected *RouteBlock, got %T", b)
	assert.Equal(t, "GET", rb.Method)
	assert.Equal(t, "/pets", rb.Path)
	assert.Equal(t, []string{"pets"}, rb.Tags)
	assert.Equal(t, "listPets", rb.OpID)
	assert.Equal(t, "Lists pets.", rb.Title())

	// Body raw blocks present as Properties.
	consumes, ok := rb.GetList("consumes")
	require.True(t, ok)
	assert.Contains(t, consumes[0], "application/json")
}

func TestParser_RouteBlock_RejectsYAMLBody(t *testing.T) {
	src := `swagger:route GET /pets pets listPets

---
parameters:
  - name: id
---`
	b := parseString(t, src)
	require.NotEmpty(t, b.Diagnostics())
	found := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeUnexpectedToken {
			found = true
		}
	}
	assert.True(t, found, "expected an unexpected-token diagnostic for OPAQUE_YAML under swagger:route")
}

func TestParser_InlineOperation_AllowsYAMLBody(t *testing.T) {
	src := `swagger:operation GET /pets pets listPets

---
parameters:
  - name: id
    in: query
---`
	b := parseString(t, src)
	ob, ok := b.(*InlineOperationBlock)
	require.True(t, ok)
	assert.Equal(t, "listPets", ob.OpID)

	yamlCount := 0
	for range ob.YAMLBlocks() {
		yamlCount++
	}
	assert.Equal(t, 1, yamlCount)

	for _, d := range b.Diagnostics() {
		assert.NotEqual(t, CodeUnexpectedToken, d.Code)
	}
}

func TestParser_MetaBlock_KeywordsAndYAML(t *testing.T) {
	src := `Booking API.

Version: 1.0.0
Host: api.example.com

Consumes:
  - application/json

---
servers:
  - url: https://api.example.com/v1
---

swagger:meta`
	b := parseString(t, src)
	mb, ok := b.(*MetaBlock)
	require.True(t, ok)
	assert.Equal(t, "Booking API.", mb.Title())

	v, ok := mb.GetString("version")
	require.True(t, ok)
	assert.Equal(t, "1.0.0", v)

	h, ok := mb.GetString("host")
	require.True(t, ok)
	assert.Equal(t, "api.example.com", h)

	cons, ok := mb.GetList("consumes")
	require.True(t, ok)
	assert.NotEmpty(t, cons)

	yamlCount := 0
	for range mb.YAMLBlocks() {
		yamlCount++
	}
	assert.Equal(t, 1, yamlCount)
}

func TestParser_ClassifierBlock_Strfmt(t *testing.T) {
	b := parseString(t, "A MAC address.\n\nswagger:strfmt mac")
	cb, ok := b.(*ClassifierBlock)
	require.True(t, ok)
	assert.Equal(t, AnnStrfmt, cb.AnnotationKind())
	require.Len(t, cb.Args, 1)
	assert.Equal(t, "mac", cb.Args[0].Text)
	assert.Empty(t, cb.Diagnostics())
}

func TestParser_ClassifierBlock_StrfmtMissingArg(t *testing.T) {
	b := parseString(t, "swagger:strfmt")
	require.NotEmpty(t, b.Diagnostics())
	assert.Equal(t, CodeMissingRequiredArg, b.Diagnostics()[0].Code)
}

// TestParser_ClassifierBlock_TypeWellFormed pins the relaxed swagger:type parsing (F3): a
// well-formed argument — canonical name, Go builtin, array, or an arbitrary identifier standing
// for a scanned-type reference — no longer raises a parser diagnostic; semantic resolution (and
// any unknown-type diagnostic) is the builder's job.
//
// Only a structurally malformed token still raises CodeInvalidTypeRef.
func TestParser_ClassifierBlock_TypeWellFormed(t *testing.T) {
	for _, arg := range []string{"string", "integer", "int64", "[]string", "custom", "Custom"} {
		b := parseString(t, "swagger:type "+arg)
		assert.Emptyf(t, b.Diagnostics(), "%q is well-formed → no parser diagnostic", arg)
	}

	bad := parseString(t, "swagger:type foo bar")
	require.NotEmpty(t, bad.Diagnostics())
	assert.Equal(t, CodeInvalidTypeRef, bad.Diagnostics()[0].Code)
}

func TestParser_EnumDecl_NameOnly(t *testing.T) {
	b := parseString(t, "swagger:enum Priority")
	eb, ok := b.(*EnumDeclBlock)
	require.True(t, ok)
	assert.Equal(t, "Priority", eb.Name)
	assert.Equal(t, enumFormNameOnly, eb.InlineForm)
}

func TestParser_EnumDecl_PlainList(t *testing.T) {
	b := parseString(t, "swagger:enum 1, 2, 3")
	eb, ok := b.(*EnumDeclBlock)
	require.True(t, ok)
	assert.Empty(t, eb.Name)
	assert.Equal(t, enumFormPlainOnly, eb.InlineForm)
}

// TestParser_EnumDecl_Bare pins the relaxed bare-swagger:enum contract (F4b): a bare `swagger:enum`
// (no name, no inline values, no body) is structurally valid — it produces an EnumDeclBlock with
// an empty Name and raises NO parse diagnostic.
//
// The builder infers the enum name from the declared type and collects its consts; "no consts
// found" is a builder-level concern, not a grammar error.
func TestParser_EnumDecl_Bare(t *testing.T) {
	b := parseString(t, "swagger:enum")
	eb, ok := b.(*EnumDeclBlock)
	require.True(t, ok, "expected *EnumDeclBlock, got %T", b)
	assert.Empty(t, eb.Name)
	for _, d := range b.Diagnostics() {
		assert.NotEqual(t, CodeMissingRequiredArg, d.Code, "bare swagger:enum must not error")
	}
}

func TestParser_UnboundBlock(t *testing.T) {
	src := `Name of the user.
required: true
maxLength: 64`
	b := parseString(t, src)
	ub, ok := b.(*UnboundBlock)
	require.True(t, ok)
	assert.Equal(t, AnnUnknown, ub.AnnotationKind())
	// UnboundBlocks now run title/desc classification too — first line ending in punctuation is
	// title (heuristic 2).
	// Required for the schema builder's PreambleTitle path on indirectly-referenced non-annotated
	// types (interfaces / aliases).
	assert.Equal(t, "Name of the user.", ub.Title())
	assert.Empty(t, ub.Description())

	required, ok := ub.GetBool("required")
	require.True(t, ok)
	assert.True(t, required)

	maxLen, ok := ub.GetInt("maxLength")
	require.True(t, ok)
	assert.Equal(t, int64(64), maxLen)
}

func TestParser_SchemaBody_NumericValidation(t *testing.T) {
	src := `swagger:model Foo

maximum: 100
minimum: <0`
	b := parseString(t, src)
	mb, ok := b.(*ModelBlock)
	require.True(t, ok)

	maximum, ok := mb.GetFloat("maximum")
	require.True(t, ok)
	assert.InDelta(t, 100.0, maximum, 0)

	// Operator preserved on Property.Typed.
	for p := range mb.Properties() {
		if p.Keyword.Name == "minimum" {
			assert.Equal(t, "<", p.Typed.Op)
			assert.InDelta(t, 0.0, p.Typed.Number, 0)
		}
	}
}

func TestParser_SchemaBody_InvalidNumber(t *testing.T) {
	b := parseString(t, "swagger:model Foo\n\nmaximum: notanumber")
	found := false
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidNumber {
			found = true
		}
	}
	assert.True(t, found)
}

func TestParser_SchemaBody_ExtensionsBlockExtractsXEntries(t *testing.T) {
	src := `swagger:model Foo

Extensions:
  x-flag: true
  x-name: hello`
	b := parseString(t, src)

	count := 0
	for ext := range b.Extensions() {
		count++
		switch ext.Name {
		case "x-flag":
			// YAML-typed: unquoted `true` is a bool.
			assert.Equal(t, true, ext.Value)
		case "x-name":
			// YAML-typed: unquoted `hello` is a string.
			assert.Equal(t, "hello", ext.Value)
		}
	}
	assert.Equal(t, 2, count)
}

// TestParser_SchemaBody_ExtensionsBlockTypedNested asserts that nested YAML mappings surface as
// typed map[string]any, not as yaml.v3's map[any]any or as a flat string.
//
// Closes the round-2 promise of `.claude/plans/typed-extensions.md`.
func TestParser_SchemaBody_ExtensionsBlockTypedNested(t *testing.T) {
	src := `swagger:model Foo

Extensions:
  x-config:
    enabled: true
    threshold: 0.5
    tags: [a, b, c]`
	b := parseString(t, src)

	var found bool
	for ext := range b.Extensions() {
		if ext.Name != "x-config" {
			continue
		}
		found = true
		cfg, ok := ext.Value.(map[string]any)
		require.True(t, ok, "x-config: want map[string]any, got %T", ext.Value)
		assert.Equal(t, true, cfg["enabled"])
		assert.Equal(t, 0.5, cfg["threshold"])
		tags, ok := cfg["tags"].([]any)
		require.True(t, ok, "x-config.tags: want []any, got %T", cfg["tags"])
		assert.Equal(t, []any{"a", "b", "c"}, tags)
	}
	assert.True(t, found, "x-config Extension should be present")
}

// TestParser_SchemaBody_ExtensionsBlockMalformedYAMLEmitsDiagnostic asserts the new
// CodeInvalidYAMLExtensions code fires when the body fails YAML parsing.
func TestParser_SchemaBody_ExtensionsBlockMalformedYAMLEmitsDiagnostic(t *testing.T) {
	src := `swagger:model Foo

Extensions:
  x-broken: [unclosed`
	b := parseString(t, src)

	count := 0
	for range b.Extensions() {
		count++
	}
	assert.Equal(t, 0, count, "malformed YAML: no Extension entries should be emitted")

	var sawDiag bool
	for _, d := range b.Diagnostics() {
		if d.Code == CodeInvalidYAMLExtensions {
			sawDiag = true
			break
		}
	}
	assert.True(t, sawDiag, "expected CodeInvalidYAMLExtensions diagnostic")
}

func TestParser_SchemaBody_DefaultRawValue(t *testing.T) {
	b := parseString(t, "swagger:model Foo\n\ndefault: hello")
	def, ok := b.GetString("default")
	require.True(t, ok)
	assert.Equal(t, "hello", def)
}

func TestParser_SchemaBody_EnumRawValue(t *testing.T) {
	src := `swagger:model Foo

enum: a, b, c`
	b := parseString(t, src)
	v, ok := b.GetString("enum")
	require.True(t, ok)
	assert.Equal(t, "a, b, c", v)
}

func TestParser_RouteBlock_GodocPrefix(t *testing.T) {
	src := `GetPets swagger:route GET /pets pets listPets`
	b := parseString(t, src)
	rb, ok := b.(*RouteBlock)
	require.True(t, ok)
	assert.Equal(t, "GET", rb.Method)
	assert.Equal(t, "/pets", rb.Path)
	assert.Equal(t, "listPets", rb.OpID)
}
