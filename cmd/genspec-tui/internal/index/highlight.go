// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"github.com/go-openapi/core/json/lexers/token"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// HighlightIndex maps each rendered line to the lexical runs on it.
//
// It costs nothing to produce: the lexer already classifies every token while the pointer index is being built,
// and until now that classification was thrown away.
//
// Re-parsing the same bytes with a separate highlighting library would be both a second pass and a dependency,
// for information already in hand.
//
// Spans record only where a run STARTS.
//
// The renderer takes each run to the next span's column, which is what lets it slice RAW text at known boundaries and
// apply styling last — the only ordering in which a truncated line cannot cut through an escape sequence.
type HighlightIndex struct {
	byLine map[int][]theme.Span
}

// Spans returns the runs on a 0-based rendered line, ordered by column, or nil when the line has none.
func (x *HighlightIndex) Spans(line int) []theme.Span {
	if x == nil {
		return nil
	}

	return x.byLine[line]
}

// All returns the whole per-line map, for a renderer that wants to install it once rather than query per line.
//
// Nil for a nil index.
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

// syntaxKind maps a lexer token to the renderer's neutral classes.
//
// Delimiters carry no value and are the structural punctuation; keys are distinguished from strings because in a spec
// the key is what you scan for.
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

// addSpan records one token's run. line is 0-based; col is the lexer's 1-based column.
//
// Both lexers emit in non-decreasing position order, so runs arrive in column order and no sort is needed.
//
// They do not arrive in strictly increasing order, though. That's currently due to some quirks in the yaml
// lexer. A YAML block collection has no "{" / "[" / "}" / "]" character for its delimiters to point at,
// so they take the span of what they enclose: the opener reports the first token inside, the closer the last.
//
// Either way a delimiter shares its column with the token that owns the text there, and the two would otherwise make a
// zero-width run followed by one painting its neighbour as punctuation.
// The token with characters of its own wins the column.
func (a *indexAccum) addSpan(line, col int, kind token.Kind) {
	runs := a.spans[line]
	syntax := syntaxKind(kind)

	if n := len(runs); n > 0 && runs[n-1].Col == col {
		if syntax != theme.SyntaxPunct {
			runs[n-1].Kind = syntax
		}

		return
	}

	a.spans[line] = append(runs, theme.Span{Col: col, Kind: syntax})
}
