// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// 0-based lines:  0 package  2 annotation  3 prose  4 type  5 tagged field
// 6 plain field  9 func with a predeclared const.
const goSrc = "package main\n" +
	"\n" +
	"// swagger:model User\n" +
	"// A user of the system.\n" +
	"type User struct {\n" +
	"\tName  string `json:\"name\"`\n" +
	"\tCount int\n" +
	"}\n" +
	"\n" +
	"func f() bool { return nil == nil }\n"

// firstKind returns the kind of the leftmost run on a 0-based line.
func firstKind(t *testing.T, idx *HighlightIndex, line int) theme.SyntaxKind {
	t.Helper()
	spans := idx.Spans(line)
	require.NotEmpty(t, spans, "line %d carries no runs", line)

	return spans[0].Kind
}

func TestGoHighlight_ClassifiesTokens(t *testing.T) {
	idx := BuildGoHighlight([]byte(goSrc))
	require.Positive(t, idx.Len())

	assert.Equal(t, theme.SyntaxKeyword, firstKind(t, idx, 0), "`package` is a keyword")
	assert.Equal(t, theme.SyntaxKeyword, firstKind(t, idx, 4), "`type` is a keyword")

	// A struct tag is a string literal; the field NAME and its type stay plain,
	// because identifiers are the bulk of Go source and need something to
	// contrast against.
	assert.Contains(t, kindsOn(idx, 5), theme.SyntaxString, "the struct tag")
	assert.Contains(t, kindsOn(idx, 5), theme.SyntaxPlain, "Name / string are identifiers")

	assert.Contains(t, kindsOn(idx, 9), theme.SyntaxKeyword, "`func`, `return` and `nil`")
	assert.Contains(t, kindsOn(idx, 9), theme.SyntaxPunct, "the parens and braces")
}

// The whole point of this pane is the annotations, so they must not read as
// dimmed-away commentary like the prose around them.
func TestGoHighlight_AnnotationCommentsOutrankOrdinaryOnes(t *testing.T) {
	idx := BuildGoHighlight([]byte(goSrc))

	assert.Equal(t, theme.SyntaxKey, firstKind(t, idx, 2), "// swagger:model User")
	assert.Equal(t, theme.SyntaxComment, firstKind(t, idx, 3), "// A user of the system.")
}

// An annotation may sit on any line of a doc block, or trail a declaration
// (AfterDeclComments) — classification is per line, not per comment token.
func TestGoHighlight_AnnotationAnywhereInABlock(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"/*\n" +
		"A user.\n" +
		"swagger:model User\n" +
		"*/\n" +
		"type User struct{} // swagger:model Other\n"

	idx := BuildGoHighlight([]byte(src))

	assert.Equal(t, theme.SyntaxComment, firstKind(t, idx, 3), "prose line of the block")
	assert.Equal(t, theme.SyntaxKey, firstKind(t, idx, 4), "annotation line of the block")

	// The trailing comment is a run of its own, after the declaration's runs.
	trailing := idx.Spans(6)
	require.NotEmpty(t, trailing)
	assert.Equal(t, theme.SyntaxKey, trailing[len(trailing)-1].Kind,
		"an inlined annotation is still the payload")
	assert.NotEqual(t, theme.SyntaxKey, trailing[0].Kind, "the code before it is not")
}

// A block comment or raw string covers lines that contain no token of their
// own. Without a run starting at their column 1 they would render plain, and
// the closing line would go plain up to whatever token follows.
func TestGoHighlight_MultiLineTokensCoverEveryLine(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"/* block\n" +
		"   still block */\n" +
		"var x = `raw\n" +
		"string`\n"

	idx := BuildGoHighlight([]byte(src))

	for _, tc := range []struct {
		line int
		kind theme.SyntaxKind
		why  string
	}{
		{2, theme.SyntaxComment, "the block comment opens"},
		{3, theme.SyntaxComment, "and continues onto a line with no token of its own"},
		{5, theme.SyntaxString, "the raw string continues"},
	} {
		spans := idx.Spans(tc.line)
		require.NotEmpty(t, spans, "line %d: %s", tc.line, tc.why)
		assert.Equal(t, tc.kind, spans[0].Kind, "line %d: %s", tc.line, tc.why)
	}

	assert.Equal(t, 1, idx.Spans(3)[0].Col, "a continuation run starts at the first column")
	assert.Equal(t, 1, idx.Spans(5)[0].Col)
}

// go/token counts columns in BYTES; the renderer slices in RUNES. Any
// multi-byte character earlier on the line shifts every run after it.
func TestGoHighlight_ColumnsAreRunesNotBytes(t *testing.T) {
	// c1..5 `const`, 7 a, 8 comma, 10 b, 12 `=`, 14 the string (7 runes), 21
	// comma, 23 the number — which would be byte column 24, `é` being 2 bytes.
	src := "package p\n\nconst a, b = \"héllo\", 42\n"

	idx := BuildGoHighlight([]byte(src))

	var numberCol int
	for _, sp := range idx.Spans(2) {
		if sp.Kind == theme.SyntaxNumber {
			numberCol = sp.Col
		}
	}
	assert.Equal(t, 23, numberCol, "byte columns would report 24")

	line := []rune(strings.Split(src, "\n")[2])
	require.Equal(t, "42", string(line[numberCol-1:numberCol+1]),
		"the column must actually land on the token")
}

// The buffer is a file someone is halfway through editing. A highlighter that
// gives up on incomplete syntax goes blank exactly when you are typing.
func TestGoHighlight_TolerantOfBrokenSource(t *testing.T) {
	src := "package p\n\nfunc f( {\n\tx := \"unterminated\n"

	idx := BuildGoHighlight([]byte(src))

	assert.Positive(t, idx.Len(), "a broken buffer still highlights")
	assert.Equal(t, theme.SyntaxKeyword, firstKind(t, idx, 2), "`func` is still a keyword")
}

// Runs must ascend, or the renderer takes one back past the previous one and
// paints the line wrong.
func TestGoHighlight_SpansAscendByColumn(t *testing.T) {
	idx := BuildGoHighlight([]byte(goSrc))

	for line := range strings.Count(goSrc, "\n") + 1 {
		spans := idx.Spans(line)
		for i, sp := range spans {
			assert.Positive(t, sp.Col, "line %d: columns are 1-based", line)
			if i > 0 {
				assert.Greater(t, sp.Col, spans[i-1].Col, "line %d: runs must not overlap", line)
			}
		}
	}
}

// The scanner synthesises the semicolons the source does not write. Marking a
// run at a character that is not there styles the padding past the line's end.
func TestGoHighlight_NoRunPastTheEndOfALine(t *testing.T) {
	// The blank line inside the block comment is the case a walk over the
	// fixture corpus turned up: a continuation run would mark column 1 of a
	// line that has no column 1, styling the padding the pane fills it with.
	src := "package p\n" +
		"\n" +
		"/* a\n" +
		"\n" +
		"   b */\n" +
		"var x = 1\n"
	lines := strings.Split(src, "\n")

	idx := BuildGoHighlight([]byte(src))

	for line, text := range lines {
		for _, sp := range idx.Spans(line) {
			assert.LessOrEqual(t, sp.Col, len([]rune(text)),
				"line %d (%q): a run starts past its last character", line, text)
		}
	}
}

func TestGoHighlight_Empty(t *testing.T) {
	idx := BuildGoHighlight(nil)

	require.NotNil(t, idx, "an empty file still yields an index, not nil")
	assert.Zero(t, idx.Len())
}

// A model with its constraints where go-swagger actually puts them: the
// annotation on the TYPE, the keywords in each FIELD's doc comment.
const goKeywordSrc = "package main\n" + // 0
	"\n" + // 1
	"// swagger:model User\n" + // 2
	"type User struct {\n" + // 3
	"\t// the user's name\n" + // 4
	"\t//\n" + // 5
	"\t// required: true\n" + // 6
	"\t// min length: 3\n" + // 7
	"\t// note: not a grammar keyword\n" + // 8
	"\t// the id of the pet: as an integer\n" + // 9
	"\tName string\n" + // 10
	"}\n" // 11

// runAt returns the text and kind of the i-th run on a line, taking each run to
// the next one's column — the same rule the renderer uses.
func runAt(t *testing.T, idx *HighlightIndex, lines []string, line, i int) (string, theme.SyntaxKind) {
	t.Helper()
	spans := idx.Spans(line)
	require.Greater(t, len(spans), i, "line %d carries no run %d", line, i)

	runes := []rune(lines[line])
	start := spans[i].Col - 1
	end := len(runes)
	if i+1 < len(spans) {
		end = spans[i+1].Col - 1
	}
	require.LessOrEqual(t, end, len(runes), "line %d run %d ends past the line", line, i)

	return string(runes[start:end]), spans[i].Kind
}

// The keyword is lifted out of the prose around it rather than the whole line
// being recoloured, so `// required: true` reads the way `"required": true`
// does on the spec side.
func TestGoHighlight_GrammarKeywordIsLiftedOutOfTheComment(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))
	lines := strings.Split(goKeywordSrc, "\n")

	marker, markerKind := runAt(t, idx, lines, 6, 0)
	keyword, keywordKind := runAt(t, idx, lines, 6, 1)
	value, valueKind := runAt(t, idx, lines, 6, 2)

	assert.Equal(t, "// ", marker, "the run starts at the comment token; the indent precedes it")
	assert.Equal(t, theme.SyntaxComment, markerKind, "the comment markers stay prose")
	assert.Equal(t, "required", keyword)
	assert.Equal(t, theme.SyntaxKeyword, keywordKind)
	assert.Equal(t, ": true", value)
	assert.Equal(t, theme.SyntaxComment, valueKind, "the value returns to prose")
}

// The table is the parser's own, so aliases and letter case come for free — and
// a multi-word keyword must be covered whole, not up to its first space.
func TestGoHighlight_GrammarKeywordAliasesAndMultiWord(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))
	lines := strings.Split(goKeywordSrc, "\n")

	keyword, kind := runAt(t, idx, lines, 7, 1)

	assert.Equal(t, "min length", keyword, "`min length` is one keyword, not `min`")
	assert.Equal(t, theme.SyntaxKeyword, kind)

	for _, spelling := range []string{"Min Length", "MIN LENGTH", "minLength"} {
		src := "package p\n\n// swagger:model U\n// " + spelling + ": 3\n"
		byLine := BuildGoHighlight([]byte(src))
		assert.Contains(t, kindsOn(byLine, 3), theme.SyntaxKeyword, spelling)
	}
}

// Only a LEADING keyword counts, and only one the table knows. Prose that
// merely contains a colon is still prose.
func TestGoHighlight_ProseIsNotMistakenForGrammar(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))

	for _, tc := range []struct {
		line int
		why  string
	}{
		{4, "a plain description"},
		{8, "`note` is not in the keyword table"},
		{9, "a colon buried in prose is not a keyword separator"},
	} {
		assert.Equal(t, []theme.SyntaxKind{theme.SyntaxComment}, kindsOn(idx, tc.line), tc.why)
	}
}

// Scope is the FILE: `name`, `in` and `example` are ordinary English words, so
// a file that declares no annotations gets no keyword highlighting at all.
func TestGoHighlight_KeywordsOnlyInAnnotatedFiles(t *testing.T) {
	src := "package p\n" +
		"\n" +
		"// helper does things.\n" +
		"// name: not a keyword here\n" +
		"// required: nor this\n" +
		"func helper() {}\n"

	idx := BuildGoHighlight([]byte(src))

	for _, line := range []int{2, 3, 4} {
		assert.Equal(t, []theme.SyntaxKind{theme.SyntaxComment}, kindsOn(idx, line),
			"line %d: nothing in this file is annotated", line)
	}
}

// The regression a walk over the fixture corpus caught: scoping to the comment
// GROUP lit up route and parameter bodies but missed every field constraint,
// because the `swagger:model` that makes them meaningful sits on the type.
func TestGoHighlight_FieldConstraintsUnderATypeAnnotation(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))

	assert.Contains(t, kindsOn(idx, 6), theme.SyntaxKeyword,
		"the annotation is on the type, two comment groups away")
	assert.Contains(t, kindsOn(idx, 7), theme.SyntaxKeyword)
}

// The annotation itself is not split: it names the block rather than setting a
// property, and keeps the spec-key class over the whole line.
func TestGoHighlight_AnnotationLineIsNotSplitIntoKeywordRuns(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))

	assert.Equal(t, []theme.SyntaxKind{theme.SyntaxKey}, kindsOn(idx, 2))
}

// Runs must ascend here too — three spans on one line is where an off-by-one
// in the keyword offsets would show up as an overlapping paint.
func TestGoHighlight_KeywordRunsAscend(t *testing.T) {
	idx := BuildGoHighlight([]byte(goKeywordSrc))

	for line := range strings.Count(goKeywordSrc, "\n") + 1 {
		spans := idx.Spans(line)
		for i := 1; i < len(spans); i++ {
			assert.Greater(t, spans[i].Col, spans[i-1].Col, "line %d", line)
		}
	}
}
