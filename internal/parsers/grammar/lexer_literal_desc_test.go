// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// descArg returns the folded argument text of the (first) swagger:description
// annotation in the token stream, or fails the test.
func descArg(t *testing.T, out []Token) string {
	t.Helper()
	for _, tk := range out {
		if tk.Kind == TokenAnnotation && tk.Name == labelDescription {
			require.Len(t, tk.Args, 1, "description annotation should carry one folded arg")
			return tk.Args[0].Text
		}
	}
	t.Fatalf("no swagger:description annotation in token stream")
	return ""
}

func hasAnnotation(out []Token, name string) bool {
	for _, tk := range out {
		if tk.Kind == TokenAnnotation && tk.Name == name {
			return true
		}
	}
	return false
}

func hasKind(out []Token, k TokenKind) bool {
	for _, tk := range out {
		if tk.Kind == k {
			return true
		}
	}
	return false
}

// TestLexer_DescriptionLiteral_PreservesMarkdown is the core contract: a
// `swagger:description |` block captures the body verbatim — indentation, a
// significant blank line, and markdown table pipes all survive.
func TestLexer_DescriptionLiteral_PreservesMarkdown(t *testing.T) {
	// Body lines carry the leading godoc `// ` convention space (modelled here
	// as a leading space in each line); it is dropped while author indentation
	// beyond it — the 2-space nested list — is preserved.
	src := strings.Join([]string{
		"swagger:description |",
		" Overview",
		"",
		" | col1 | col2 |",
		" |------|------|",
		"   - nested item",
	}, "\n")
	arg := descArg(t, lexString(t, src))

	want := "Overview\n\n| col1 | col2 |\n|------|------|\n  - nested item"
	assert.Equal(t, want, arg)
	// the `|` marker itself must not leak into the body.
	assert.NotEqual(t, "|", strings.Split(arg, "\n")[0])
}

// TestLexer_DescriptionLiteral_DashKeepsFollowingAnnotation is the regression
// the stage-1 literal mode exists for: a lone `---` in the body must not open a
// YAML fence and swallow the following annotation.
func TestLexer_DescriptionLiteral_DashKeepsFollowingAnnotation(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"Overview",
		"---",
		"after the dash",
		"swagger:model Foo",
	}, "\n")
	out := lexString(t, src)

	arg := descArg(t, out)
	assert.Equal(t, "Overview\n---\nafter the dash", arg, "the --- line is body, captured verbatim")
	assert.True(t, hasAnnotation(out, labelModel), "the following swagger:model must survive the --- in the body")
	assert.False(t, hasKind(out, TokenOpaqueYaml), "the body --- must not open a YAML fence")
}

// TestLexer_DescriptionLiteral_BlankDoesNotTerminate: unlike the default
// Option B fold, a blank line inside the literal block is body, not a
// terminator; the block ends at the next annotation.
func TestLexer_DescriptionLiteral_BlankDoesNotTerminate(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"para one",
		"",
		"para two",
		"swagger:model Foo",
	}, "\n")
	out := lexString(t, src)
	assert.Equal(t, "para one\n\npara two", descArg(t, out))
	assert.True(t, hasAnnotation(out, labelModel))
}

// TestLexer_DescriptionLiteral_KeywordLineIsBody: a keyword-looking line inside
// the block is captured as body, never treated as a terminator (decision 3 — no
// keyword sensitivity).
func TestLexer_DescriptionLiteral_KeywordLineIsBody(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"Body mentioning maximum: 5 inline",
		"default: not a keyword here",
	}, "\n")
	arg := descArg(t, lexString(t, src))
	assert.Contains(t, arg, "maximum: 5")
	assert.Contains(t, arg, "default: not a keyword here")
}

// TestLexer_DescriptionLiteral_MidLineSwaggerIsBody: only an annotation at the
// START of a line terminates the block; a `swagger:` token mid-line is prose.
func TestLexer_DescriptionLiteral_MidLineSwaggerIsBody(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"You can write swagger:model in the middle of a sentence.",
		"And mention swagger:meta too.",
		"swagger:model Foo",
	}, "\n")
	out := lexString(t, src)
	arg := descArg(t, out)
	assert.Equal(t, "You can write swagger:model in the middle of a sentence.\nAnd mention swagger:meta too.", arg)
	assert.True(t, hasAnnotation(out, labelModel), "the line-leading swagger:model terminates and survives")
}

// TestLexer_DescriptionLiteral_IndentedSwaggerStillTerminates documents the
// edge: the comment-prefix strip removes leading indentation before the
// annotation check, so even an indented line that BEGINS with `swagger:`
// terminates the block (you cannot hide it by indenting, e.g. in a code block).
func TestLexer_DescriptionLiteral_IndentedSwaggerStillTerminates(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"body line",
		"    swagger:model Foo",
	}, "\n")
	out := lexString(t, src)
	assert.Equal(t, "body line", descArg(t, out))
	assert.True(t, hasAnnotation(out, labelModel))
}

// TestLexer_DescriptionLiteral_TrailingBlankClipped: bare `|` clips trailing
// blank lines (interior ones are kept, see above).
func TestLexer_DescriptionLiteral_TrailingBlankClipped(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description |",
		"body",
		"",
		"",
	}, "\n")
	assert.Equal(t, "body", descArg(t, lexString(t, src)))
}

// TestLexer_DescriptionLiteral_EmptyBody: a marker with no body folds to an
// empty description rather than leaking the `|`.
func TestLexer_DescriptionLiteral_EmptyBody(t *testing.T) {
	assert.Equal(t, "", descArg(t, lexString(t, "swagger:description |")))
}

// TestLexer_DescriptionLiteral_PlainUnchanged: without the `|` marker the
// default Option B behaviour is intact — the body folds to the first blank line
// and trailing prose stays out of the description.
func TestLexer_DescriptionLiteral_PlainUnchanged(t *testing.T) {
	src := strings.Join([]string{
		"swagger:description short desc",
		"continued on next line",
		"",
		"this trailing prose is not part of the description",
	}, "\n")
	out := lexString(t, src)
	arg := descArg(t, out)
	assert.Equal(t, "short desc\ncontinued on next line", arg)
	assert.NotContains(t, arg, "trailing prose")
}
