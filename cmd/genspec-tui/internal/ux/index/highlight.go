// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"sort"

	"github.com/go-openapi/core/json/lexers/token"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// HighlightIndex maps each rendered line to the lexical runs on it.
//
// It costs nothing to produce: the lexer already classifies every token while
// the pointer index is being built, and until now that classification was
// thrown away. Re-parsing the same bytes with a separate highlighting library
// would be both a second pass and a dependency, for information already in hand.
//
// Spans record only where a run STARTS. The renderer takes each run to the next
// span's column, which is what lets it slice RAW text at known boundaries and
// apply styling last — the only ordering in which a truncated line cannot cut
// through an escape sequence.
type HighlightIndex struct {
	byLine map[int][]theme.Span
}

// Spans returns the runs on a 0-based rendered line, ordered by column, or nil
// when the line has none.
func (x *HighlightIndex) Spans(line int) []theme.Span {
	if x == nil {
		return nil
	}

	return x.byLine[line]
}

// All returns the whole per-line map, for a renderer that wants to install it
// once rather than query per line. Nil for a nil index.
func (x *HighlightIndex) All() map[int][]theme.Span {
	if x == nil {
		return nil
	}

	return x.byLine
}

// Len reports how many lines carry spans (0 for a nil index).
func (x *HighlightIndex) Len() int {
	if x == nil {
		return 0
	}

	return len(x.byLine)
}

// syntaxKind maps a lexer token to the renderer's neutral classes. Delimiters
// carry no value and are the structural punctuation; keys are distinguished
// from strings because in a spec the key is what you scan for.
func syntaxKind(k token.Kind) theme.SyntaxKind {
	switch k {
	case token.Key:
		return theme.SyntaxKey
	case token.String:
		return theme.SyntaxString
	case token.Number:
		return theme.SyntaxNumber
	case token.Boolean, token.Null:
		return theme.SyntaxKeyword
	case token.Delimiter:
		return theme.SyntaxPunct
	case token.Unknown, token.EOF:
		return theme.SyntaxPlain
	default:
		return theme.SyntaxPlain
	}
}

// addSpan records one token's run. line is 0-based; col is the lexer's 1-based
// column. Tokens with no position (the EOF delimiters the YAML lexer reports at
// line 0) are dropped rather than attributed to the first line.
func (a *indexAccum) addSpan(line, col int, kind token.Kind) {
	if line < 0 || col < 1 {
		return
	}
	a.spans[line] = append(a.spans[line], theme.Span{Col: col, Kind: syntaxKind(kind)})
}

// finishSpans orders each line's runs by column.
//
// The sort is required, not defensive: the YAML lexer currently reports a
// mapping's value-delimiter BEFORE the key on the same line, so the stream is
// not in source order. Fred plans to improve that lexer's column reporting
// upstream; when it lands in source order this sort becomes a no-op and can go.
func (a *indexAccum) finishSpans() *HighlightIndex {
	for line := range a.spans {
		runs := a.spans[line]
		sort.Slice(runs, func(i, j int) bool { return runs[i].Col < runs[j].Col })
	}

	return &HighlightIndex{byLine: a.spans}
}
