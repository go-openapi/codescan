// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const hlJSON = `{
  "definitions": {
    "User": {
      "count": 3,
      "ok": true,
      "boss": null,
      "name": "x"
    }
  }
}`

// kindsOn returns the span kinds on a 0-based line, in column order.
func kindsOn(idx *HighlightIndex, line int) []theme.SyntaxKind {
	spans := idx.Spans(line)
	out := make([]theme.SyntaxKind, 0, len(spans))
	for _, sp := range spans {
		out = append(out, sp.Kind)
	}

	return out
}

func TestHighlight_ClassifiesJSONTokens(t *testing.T) {
	idx := BuildJSONIndex([]byte(hlJSON)).Highlight
	require.Positive(t, idx.Len())

	// `      "count": 3,`
	assert.Equal(t,
		[]theme.SyntaxKind{theme.SyntaxKey, theme.SyntaxPunct, theme.SyntaxNumber, theme.SyntaxPunct},
		kindsOn(idx, 3))

	// booleans and null are one class — they are the literal keywords
	assert.Contains(t, kindsOn(idx, 4), theme.SyntaxKeyword, "true")
	assert.Contains(t, kindsOn(idx, 5), theme.SyntaxKeyword, "null")

	// a string VALUE is distinct from a key, which is what you scan a spec for
	assert.Contains(t, kindsOn(idx, 6), theme.SyntaxString)
}

// Spans record where a run starts, so columns must be 1-based and ascending —
// the renderer takes each run to the next one's column.
func TestHighlight_SpansAreOrderedByColumn(t *testing.T) {
	idx := BuildJSONIndex([]byte(hlJSON)).Highlight

	for line := range 8 {
		spans := idx.Spans(line)
		for i, sp := range spans {
			assert.Positive(t, sp.Col, "line %d span %d: columns are 1-based", line, i)
			if i > 0 {
				assert.Greater(t, sp.Col, spans[i-1].Col,
					"line %d: spans must ascend, or runs would overlap", line)
			}
		}
	}
}

// The YAML lexer currently reports a mapping's value-delimiter BEFORE the key on
// the same line, so the accumulator sorts. Without that, the renderer would take
// the first run from column 12 back to column 1 and paint the line wrong.
func TestHighlight_YAMLSpansAreSortedDespiteEmissionOrder(t *testing.T) {
	const hlYAML = `definitions:
  User:
    count: 3
    ok: true
`
	idx := BuildYAMLIndex([]byte(hlYAML)).Highlight
	require.Positive(t, idx.Len())

	for line := range 4 {
		spans := idx.Spans(line)
		for i := 1; i < len(spans); i++ {
			assert.Greater(t, spans[i].Col, spans[i-1].Col, "line %d", line)
		}
	}

	// Line 0 is `definitions:` — the key must come first despite being emitted
	// after its delimiter.
	first := idx.Spans(0)
	require.NotEmpty(t, first)
	assert.Equal(t, theme.SyntaxKey, first[0].Kind)
	assert.Equal(t, 1, first[0].Col)
}

// The YAML lexer reports its trailing EOF delimiters at line 0 / column 0.
// Attributing those to the first line would paint a run that is not there.
func TestHighlight_DropsPositionlessTokens(t *testing.T) {
	idx := BuildYAMLIndex([]byte("a: 1\n")).Highlight

	for _, sp := range idx.Spans(0) {
		assert.Positive(t, sp.Col)
	}
	assert.Empty(t, idx.Spans(-1), "nothing is filed under a negative line")
}

func TestHighlight_NilAndEmpty(t *testing.T) {
	var nilIdx *HighlightIndex
	assert.Empty(t, nilIdx.Spans(0))
	assert.Zero(t, nilIdx.Len())
	assert.Nil(t, nilIdx.All())

	idx := BuildJSONIndex([]byte(`{}`)).Highlight
	assert.NotNil(t, idx, "an empty document still yields an index, not nil")
}

// One walk, three products — adding the highlight index must not have cost a
// second traversal, nor disturbed the other two.
func TestHighlight_SharesTheWalkWithTheOtherIndexes(t *testing.T) {
	built := BuildJSONIndex([]byte(hlJSON))

	require.NotNil(t, built.Spec)
	require.NotNil(t, built.Refs)
	require.NotNil(t, built.Highlight)

	_, ok := built.Spec.LineForPointer("/definitions/User/properties")
	assert.False(t, ok, "this fixture has no properties node")

	line, ok := built.Spec.LineForPointer("/definitions/User/count")
	require.True(t, ok)
	assert.NotEmpty(t, built.Highlight.Spans(line),
		"the line the pointer index found must also carry spans")
}
