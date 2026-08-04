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

// A YAML block collection's delimiters have no character of their own, so they report the span of what they enclose —
// the opener the first token inside, the closer the last.
//
// Both therefore share a column with a token that does own the text, and the accumulator must let that token keep the
// run: a delimiter run starting where a value starts would paint the value as punctuation.
const hlYAML = `definitions:
  User:
    count: 3
    ok: true
tags:
  - a
  - b
`

func TestHighlight_YAMLBlockDelimitersDoNotStealTheirNeighboursColumn(t *testing.T) {
	idx := BuildYAMLIndex([]byte(hlYAML)).Highlight
	require.Positive(t, idx.Len())

	// The opening delimiters of `definitions:` — the root mapping's, emitted BEFORE the key it shares column 1 with.
	first := idx.Spans(0)
	require.NotEmpty(t, first)
	assert.Equal(t, theme.SyntaxKey, first[0].Kind, "the key owns column 1, not the mapping opener")
	assert.Equal(t, 1, first[0].Col)

	// ` ok: true` closes User and definitions, so two closing delimiters land on `true`'s own column, after it.
	assert.Equal(t,
		[]theme.SyntaxKind{theme.SyntaxKey, theme.SyntaxKeyword},
		kindsOn(idx, 3), "`true` keeps its class through the closers")

	// Same at the end of a block sequence: ` - b` closes the sequence and the document.
	assert.Equal(t,
		[]theme.SyntaxKind{theme.SyntaxString},
		kindsOn(idx, 6), "`b` keeps its class through the closers")
}

// Spans record where a run STARTS, so a line can never hold two runs at the same column, whichever lexer produced them
// — the renderer would emit one of them as a zero-width run and paint the other over its neighbour's text.
func TestHighlight_SpansStartAtDistinctColumns(t *testing.T) {
	for name, idx := range map[string]*HighlightIndex{
		"json": BuildJSONIndex([]byte(hlJSON)).Highlight,
		"yaml": BuildYAMLIndex([]byte(hlYAML)).Highlight,
	} {
		t.Run(name, func(t *testing.T) {
			for line, spans := range idx.All() {
				assert.GreaterOrEqual(t, line, 0, "nothing is filed under a negative line")
				for i, sp := range spans {
					assert.Positive(t, sp.Col, "line %d span %d: columns are 1-based", line, i)
					if i > 0 {
						assert.Greater(t, sp.Col, spans[i-1].Col, "line %d", line)
					}
				}
			}
		})
	}
}

func TestHighlight_NilAndEmpty(t *testing.T) {
	var nilIdx *HighlightIndex
	assert.Empty(t, nilIdx.Spans(0))
	assert.Zero(t, nilIdx.Len())
	assert.Nil(t, nilIdx.All())

	idx := BuildJSONIndex([]byte(`{}`)).Highlight
	assert.NotNil(t, idx, "an empty document still yields an index, not nil")
}

// One walk, three products — adding the highlight index must not have cost a second traversal, nor disturbed the
// other two.
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

// kindsOn returns the span kinds on a 0-based line, in column order.
func kindsOn(idx *HighlightIndex, line int) []theme.SyntaxKind {
	spans := idx.Spans(line)
	out := make([]theme.SyntaxKind, 0, len(spans))
	for _, sp := range spans {
		out = append(out, sp.Kind)
	}

	return out
}
