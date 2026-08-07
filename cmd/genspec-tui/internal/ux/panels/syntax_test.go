// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// plainOf strips SGR escapes, leaving the text a user actually sees.
func plainOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}

			continue
		}
		b.WriteByte(s[i])
	}

	return b.String()
}

// countLine is a sample rendered JSON line paired with the spans the lexer reports for it.
//
// Columns are 1-based, as the lexer gives them.
func countLine() (string, []theme.Span) {
	return `  "count": 3,`, []theme.Span{
		{Col: 3, Kind: theme.SyntaxKey},
		{Col: 10, Kind: theme.SyntaxPunct},
		{Col: 12, Kind: theme.SyntaxNumber},
		{Col: 13, Kind: theme.SyntaxPunct},
	}
}

// THE invariant: colouring never changes what is written, at any width.
//
// If this holds under truncation then no escape was ever cut, because the visible text would not survive it.
func TestRenderSpans_NeverAltersTheVisibleText(t *testing.T) {
	raw, spans := countLine()

	for width := 1; width <= len([]rune(raw))+5; width++ {
		got := renderSpans(raw, spans, width)
		assert.Equal(t, fit(raw, width), plainOf(got),
			"width %d: the visible text must match the plain fit exactly", width)
	}
}

func TestRenderSpans_AppliesTheStylePerRun(t *testing.T) {
	raw, spans := countLine()

	got := renderSpans(raw, spans, len([]rune(raw)))

	assert.Contains(t, got, theme.Syntax(theme.SyntaxKey).Render(`"count"`))
	assert.Contains(t, got, theme.Syntax(theme.SyntaxNumber).Render("3"))
	assert.True(t, strings.HasPrefix(got, "  "),
		"leading indentation is emitted unstyled, not swallowed by the first run")
}

func TestRenderSpans_Edges(t *testing.T) {
	raw, spans := countLine()

	assert.Empty(t, renderSpans(raw, spans, 0), "no width, nothing to draw")
	assert.Equal(t, fit(raw, 8), renderSpans(raw, nil, 8), "no spans renders the plain fit")

	// A span starting beyond the (truncated) line must not panic or invent text.
	beyond := []theme.Span{{Col: 999, Kind: theme.SyntaxKey}}
	assert.Equal(t, fit(raw, 6), plainOf(renderSpans(raw, beyond, 6)))
}

// Multi-byte content must be sliced by rune, not byte, or the columns drift.
func TestRenderSpans_MultiByte(t *testing.T) {
	raw := `  "café": "naïve",`
	spans := []theme.Span{
		{Col: 3, Kind: theme.SyntaxKey},
		{Col: 9, Kind: theme.SyntaxPunct},
		{Col: 11, Kind: theme.SyntaxString},
	}

	for width := 1; width <= len([]rune(raw))+2; width++ {
		got := renderSpans(raw, spans, width)
		require.Equal(t, fit(raw, width), plainOf(got), "width %d", width)
	}
}

// An unmapped kind renders unstyled rather than wrong.
//
// SyntaxPlain is the zero value precisely so a token nobody classified degrades quietly.
func TestRenderSpans_PlainKindIsUnstyled(t *testing.T) {
	got := renderSpans("abc", []theme.Span{{Col: 1, Kind: theme.SyntaxPlain}}, 3)

	assert.Equal(t, "abc", plainOf(got))
}
