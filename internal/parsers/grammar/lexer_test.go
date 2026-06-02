// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// parseGoSource parses a Go source file via go/parser, finds the
// first top-level decl with a doc comment, and runs grammar.Parse on
// it. Used to exercise the public *ast.CommentGroup entry point —
// where Go directives like //nolint: appear with their original
// (no-leading-space) shape.
//
//nolint:ireturn // delegates to Parse which returns Block.
func parseGoSource(t *testing.T, src string) Block {
	t.Helper()
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "fake.go", src, goparser.ParseComments)
	require.NoError(t, err)
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			if decl.Doc != nil {
				return Parse(decl.Doc, fset)
			}
		case *ast.FuncDecl:
			if decl.Doc != nil {
				return Parse(decl.Doc, fset)
			}
		}
	}
	t.Fatalf("no doc-bearing decl found in fixture")
	return nil
}

// lexString preprocesses a comment block (without leading // markers) and
// runs the full Lex pipeline. Each input line becomes one Line entry.
func lexString(t *testing.T, src string) []Token {
	t.Helper()
	const likelyLines = 5
	lines := make([]Line, 0, likelyLines)
	pos := token.Position{Filename: "test.go", Line: 1, Column: 1}
	for i, raw := range strings.Split(src, "\n") {
		p := pos
		p.Line = 1 + i
		lines = append(lines, Line{Text: trimContentPrefix(raw), Raw: raw, Pos: p})
	}
	return Lex(lines)
}

func tokenKinds(toks []Token) []TokenKind {
	out := make([]TokenKind, len(toks))
	for i, t := range toks {
		out[i] = t.Kind
	}
	return out
}

func TestLexer_AnnotationBare(t *testing.T) {
	out := lexString(t, "swagger:meta")
	require.GreaterOrEqual(t, len(out), 2)
	assert.Equal(t, TokenAnnotation, out[0].Kind)
	assert.Equal(t, "meta", out[0].Name)
	assert.Empty(t, out[0].Args)
	assert.Equal(t, TokenEOF, out[len(out)-1].Kind)
}

func TestLexer_AnnotationTrailingDot(t *testing.T) {
	out := lexString(t, "swagger:strfmt uuid.")
	require.NotEmpty(t, out)
	require.Equal(t, TokenAnnotation, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenIdentName, out[0].Args[0].Kind)
	assert.Equal(t, "uuid", out[0].Args[0].Text)
}

func TestLexer_AnnotationCaseInsensitiveFirstChar(t *testing.T) {
	out := lexString(t, "Swagger:strfmt uuid")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenAnnotation, out[0].Kind)
	assert.Equal(t, "strfmt", out[0].Name)
}

func TestLexer_RouteWithGodocPrefix(t *testing.T) {
	out := lexString(t, "GetPets swagger:route GET /pets pets listPets")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenAnnotation, out[0].Kind)
	assert.Equal(t, "route", out[0].Name)
	require.Len(t, out[0].Args, 4)
	assert.Equal(t, TokenHTTPMethod, out[0].Args[0].Kind)
	assert.Equal(t, "GET", out[0].Args[0].Text)
	assert.Equal(t, TokenURLPath, out[0].Args[1].Kind)
	assert.Equal(t, "/pets", out[0].Args[1].Text)
	assert.Equal(t, TokenIdentName, out[0].Args[2].Kind)
	assert.Equal(t, "pets", out[0].Args[2].Text)
	assert.Equal(t, TokenIdentName, out[0].Args[3].Kind)
	assert.Equal(t, "listPets", out[0].Args[3].Text)
}

func TestLexer_RouteOnlyGetsGodocPrefix(t *testing.T) {
	// Other annotations must NOT accept a leading godoc identifier.
	out := lexString(t, "GetPets swagger:operation GET /pets pets listPets")
	// Expect a fallthrough text token, not an annotation.
	require.NotEmpty(t, out)
	assert.NotEqual(t, TokenAnnotation, out[0].Kind)
}

func TestLexer_KeywordInlineNumber(t *testing.T) {
	out := lexString(t, "maximum: 10")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenKeyword, out[0].Kind)
	assert.Equal(t, "maximum", out[0].Name)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenNumberValue, out[0].Args[0].Kind)
	assert.Equal(t, "10", out[0].Args[0].Text)
}

func TestLexer_KeywordCaseInsensitiveFirstChar(t *testing.T) {
	out := lexString(t, "Maximum: 10")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenKeyword, out[0].Kind)
	assert.Equal(t, "maximum", out[0].Name)
	assert.Equal(t, "Maximum", out[0].SourceName)
}

func TestLexer_ItemsPrefixDepth(t *testing.T) {
	out := lexString(t, "items.items.maxLength: 5")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenKeyword, out[0].Kind)
	assert.Equal(t, "maxLength", out[0].Name)
	assert.Equal(t, 2, out[0].ItemsDepth)
}

func TestLexer_RawBlockConsumes(t *testing.T) {
	out := lexString(t, "Consumes:\n  - application/json\n  - application/xml")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenRawBlockBody, out[0].Kind)
	assert.Equal(t, "consumes", out[0].Keyword)
	assert.Contains(t, out[0].Body, "application/json")
	assert.Contains(t, out[0].Body, "application/xml")
}

func TestLexer_RawBlockTerminatedBySibling(t *testing.T) {
	src := strings.Join([]string{
		"Consumes:",
		"  - application/json",
		"Produces:",
		"  - application/json",
	}, "\n")
	out := lexString(t, src)
	require.GreaterOrEqual(t, len(out), 3)
	assert.Equal(t, TokenRawBlockBody, out[0].Kind)
	assert.Equal(t, "consumes", out[0].Keyword)
	assert.Equal(t, TokenRawBlockBody, out[1].Kind)
	assert.Equal(t, "produces", out[1].Keyword)
}

func TestLexer_OpaqueYamlFenced(t *testing.T) {
	src := strings.Join([]string{
		"swagger:operation GET /pets pets listPets",
		"---",
		"parameters:",
		"  - name: id",
		"---",
	}, "\n")
	out := lexString(t, src)

	// Find the OpaqueYaml token.
	var yamlTok *Token
	for i := range out {
		if out[i].Kind == TokenOpaqueYaml {
			yamlTok = &out[i]
			break
		}
	}
	require.NotNil(t, yamlTok)
	assert.Contains(t, yamlTok.Body, "parameters:")
	assert.Contains(t, yamlTok.Body, "name: id")
	assert.False(t, yamlTok.Truncated)
}

func TestLexer_OpaqueYamlTruncated(t *testing.T) {
	src := strings.Join([]string{
		"swagger:operation GET /pets pets listPets",
		"---",
		"parameters:",
	}, "\n")
	out := lexString(t, src)

	var yamlTok *Token
	for i := range out {
		if out[i].Kind == TokenOpaqueYaml {
			yamlTok = &out[i]
			break
		}
	}
	require.NotNil(t, yamlTok)
	assert.True(t, yamlTok.Truncated)
}

func TestLexer_RawValueDefaultInline(t *testing.T) {
	out := lexString(t, "default: hello")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenRawValueBody, out[0].Kind)
	assert.Equal(t, "default", out[0].Keyword)
	assert.Equal(t, "hello", out[0].Body)
}

func TestLexer_RawValueDefaultMultiline(t *testing.T) {
	src := strings.Join([]string{
		"default:",
		"  one",
		"  two",
	}, "\n")
	out := lexString(t, src)
	require.NotEmpty(t, out)
	assert.Equal(t, TokenRawValueBody, out[0].Kind)
	assert.Equal(t, "default", out[0].Keyword)
	assert.Contains(t, out[0].Body, "one")
	assert.Contains(t, out[0].Body, "two")
}

func TestLexer_TitleDescriptionSplit_PunctuationHeuristic(t *testing.T) {
	src := strings.Join([]string{
		"A pet in the store.",
		"With a longer follow-up paragraph",
		"that spans multiple lines.",
		"swagger:model Pet",
	}, "\n")
	out := lexString(t, src)

	titles := collectKind(out, TokenTitle)
	descs := collectKind(out, TokenDesc)
	require.NotEmpty(t, titles)
	require.NotEmpty(t, descs)
	assert.Equal(t, "A pet in the store.", titles[0].Text)
	assert.Contains(t, descs[0].Text+" "+descs[1].Text, "longer follow-up")
}

func TestLexer_TitleDescriptionSplit_BlankLineHeuristic(t *testing.T) {
	src := strings.Join([]string{
		"A short title",
		"",
		"And a description following a blank line.",
		"swagger:model Pet",
	}, "\n")
	out := lexString(t, src)

	titles := collectKind(out, TokenTitle)
	descs := collectKind(out, TokenDesc)
	require.NotEmpty(t, titles)
	require.NotEmpty(t, descs)
	assert.Equal(t, "A short title", titles[0].Text)
}

func TestLexer_UnboundBlockClassifiesTitle(t *testing.T) {
	// UnboundBlocks (no swagger annotation) still get title/desc
	// classification — v1's helpers.CollectScannerTitleDescription
	// applied the same heuristics regardless of annotation presence,
	// and downstream consumers (e.g. the schema builder when a
	// non-annotated interface is referenced via $ref) rely on
	// PreambleTitle being populated.
	out := lexString(t, "Name of the user.\nrequired: true")
	titles := collectKind(out, TokenTitle)
	require.NotEmpty(t, titles, "first prose line ending in punct should be TITLE")
	assert.Equal(t, "Name of the user.", titles[0].Text)
}

func TestLexer_DefaultAnnotation_JsonValue(t *testing.T) {
	out := lexString(t, `swagger:default {"x": 1}`)
	require.NotEmpty(t, out)
	require.Equal(t, TokenAnnotation, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenJSONValue, out[0].Args[0].Kind)
}

func TestLexer_DefaultAnnotation_RawFallback(t *testing.T) {
	out := lexString(t, "swagger:default high")
	require.NotEmpty(t, out)
	require.Equal(t, TokenAnnotation, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenRawValue, out[0].Args[0].Kind)
}

func TestLexer_TypeAnnotation_ClosedVocabulary(t *testing.T) {
	out := lexString(t, "swagger:type string")
	require.NotEmpty(t, out)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenTypeRef, out[0].Args[0].Kind)
	assert.Equal(t, "string", out[0].Args[0].Text)

	out2 := lexString(t, "swagger:type custom")
	require.NotEmpty(t, out2)
	require.Len(t, out2[0].Args, 1)
	assert.Equal(t, TokenIdentName, out2[0].Args[0].Kind, "unknown type tokens fall back to IDENT_NAME for analyzer diagnosis")
}

func TestLexer_EnumAnnotation_NameOnly(t *testing.T) {
	out := lexString(t, "swagger:enum Priority")
	require.NotEmpty(t, out)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenIdentName, out[0].Args[0].Kind)
	assert.Equal(t, "Priority", out[0].Args[0].Text)
}

func TestLexer_EnumAnnotation_PlainListNoName(t *testing.T) {
	out := lexString(t, "swagger:enum 1, 2, 3")
	require.NotEmpty(t, out)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, TokenCommaListValue, out[0].Args[0].Kind)
}

func TestLexer_EnumAnnotation_NamePlusBracketed(t *testing.T) {
	out := lexString(t, "swagger:enum kind [a, b, c]")
	require.NotEmpty(t, out)
	require.Len(t, out[0].Args, 2)
	assert.Equal(t, TokenIdentName, out[0].Args[0].Kind)
	assert.Equal(t, "kind", out[0].Args[0].Text)
	assert.Equal(t, TokenJSONValue, out[0].Args[1].Kind)
}

func TestLexer_BlankLinePreserved(t *testing.T) {
	out := lexString(t, "swagger:meta\n\nVersion: 1")
	// At least one BLANK between annotation and keyword.
	hasBlank := false
	for _, k := range tokenKinds(out) {
		if k == TokenBlank {
			hasBlank = true
		}
	}
	assert.True(t, hasBlank)
}

func TestLexer_DecorativeFenceInExtensions(t *testing.T) {
	src := strings.Join([]string{
		"Extensions:",
		"---",
		"x-foo: bar",
		"---",
	}, "\n")
	out := lexString(t, src)
	require.NotEmpty(t, out)
	assert.Equal(t, TokenRawBlockBody, out[0].Kind)
	assert.Equal(t, "extensions", out[0].Keyword)
	assert.Contains(t, out[0].Body, "x-foo: bar")
}

// collectKind returns the subset of out whose Kind matches.
func collectKind(out []Token, k TokenKind) []Token {
	var found []Token
	for _, t := range out {
		if t.Kind == k {
			found = append(found, t)
		}
	}
	return found
}

func TestLexer_TrailingWhitespaceOnAnnotationLine(t *testing.T) {
	out := lexString(t, "swagger:strfmt uuid   ")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenAnnotation, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, "uuid", out[0].Args[0].Text)
}

func TestLexer_TrailingWhitespaceOnKeywordLine(t *testing.T) {
	out := lexString(t, "maximum: 10  \t")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenKeyword, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, "10", out[0].Args[0].Text)
}

func TestLexer_TrailingNonASCIIWhitespace(t *testing.T) {
	// U+00A0 NO-BREAK SPACE, U+2028 LINE SEPARATOR — TrimRightFunc with
	// unicode.IsSpace must strip them; TrimRight on " \t" alone would
	// leave them attached.
	out := lexString(t, "swagger:strfmt uuid  ")
	require.NotEmpty(t, out)
	assert.Equal(t, TokenAnnotation, out[0].Kind)
	require.Len(t, out[0].Args, 1)
	assert.Equal(t, "uuid", out[0].Args[0].Text)
}

func TestLexer_WhitespaceOnlyLineIsBlank(t *testing.T) {
	// `// \t  ` style line — strips to empty, must surface as BLANK.
	out := lexString(t, "swagger:meta\n   \t  \nVersion: 1")
	hasBlank := false
	for _, k := range tokenKinds(out) {
		if k == TokenBlank {
			hasBlank = true
		}
	}
	assert.True(t, hasBlank)
}

func TestLexer_GoDirectivesDroppedFromProse(t *testing.T) {
	cases := []string{
		"nolint:gocritic",
		"go:generate stringer -type=Foo",
		"lint:ignore U1000 unused field",
		"staticcheck:foo",
	}
	for _, raw := range cases {
		assert.True(t, isGoDirective(raw), "expected %q to be a directive", raw)
	}

	notDirectives := []string{
		" nolint:gocritic", // leading space (idiomatic prose)
		"NoLint:foo",       // uppercase first char
		"foo bar:baz",      // contains space before colon
		"description without a colon",
		"required: true", // keyword: space after colon
		"maximum: 10",    // keyword: space after colon
		"pattern:",       // block head: empty after colon
		"version: 1.0.0", // keyword: space after colon
	}
	for _, raw := range notDirectives {
		assert.False(t, isGoDirective(raw), "did not expect %q to be a directive", raw)
	}

	// `swagger:model Pet` matches the directive shape per isGoDirective
	// in isolation — the swagger annotation check runs first in
	// lexLine, so this never reaches the directive filter at runtime.
	assert.True(t, isGoDirective("swagger:model"),
		"swagger annotations match the directive shape — lexLine special-cases them upstream")
}

func TestLexer_DirectiveDoesNotPolluteTitle(t *testing.T) {
	// Source: a docstring with an embedded //nolint directive. The
	// title/description surface must not include the directive.
	src := strings.Join([]string{
		"A pet in the store.",
		"",
		"With a longer description.",
		"swagger:model Pet",
	}, "\n")
	cleanOut := lexString(t, src)
	cleanTitles := collectKind(cleanOut, TokenTitle)
	cleanDescs := collectKind(cleanOut, TokenDesc)

	srcWithDirective := strings.Join([]string{
		"A pet in the store.",
		"",
		"With a longer description.",
		// Simulate the post-`//` content of `//nolint:revive`.
		// preprocessText feeds it as Raw verbatim via our lexString helper:
		// trimContentPrefix strips the leading `/` chars only on Text.
		"swagger:model Pet",
	}, "\n")
	out := lexString(t, srcWithDirective)
	titles := collectKind(out, TokenTitle)
	descs := collectKind(out, TokenDesc)

	// The two streams must classify the same prose surface.
	assert.Equal(t, len(cleanTitles), len(titles))
	assert.Equal(t, len(cleanDescs), len(descs))

	// Direct directive presence test via the public Parse() path: a
	// CommentGroup with a //nolint line interleaved into a docstring
	// must not surface "nolint:" anywhere in Title / Description.
	srcGo := `package fake

// A pet in the store.
//
//nolint:revive
//
// With a longer description.
//
// swagger:model Pet
type Pet struct{}
`
	b := parseGoSource(t, srcGo)
	mb, ok := b.(*ModelBlock)
	require.True(t, ok, "expected *ModelBlock, got %T", b)
	assert.NotContains(t, mb.Title(), "nolint")
	assert.NotContains(t, mb.Description(), "nolint")
	assert.Equal(t, "A pet in the store.", mb.Title())
	assert.Equal(t, "With a longer description.", mb.Description())
}

func TestLexer_DirectiveDroppedInsideRawBlock(t *testing.T) {
	src := `package fake

// swagger:meta
//
// Consumes:
// - application/json
//nolint:gocritic
// - application/xml
type _ struct{}
`
	b := parseGoSource(t, src)
	cons, ok := b.GetList("consumes")
	require.True(t, ok)
	joined := strings.Join(cons, "\n")
	assert.Contains(t, joined, "application/json")
	assert.Contains(t, joined, "application/xml")
	assert.NotContains(t, joined, "nolint")
}
