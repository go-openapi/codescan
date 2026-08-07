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

// highlightedFileView loads three lines and colours the middle one.
//
// A test can then tell styled from not styled without depending on the palette.
func highlightedFileView() FileView {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "package p\n\nvar x = 1\n")
	fv.SetSpans(map[int][]theme.Span{
		2: {
			{Col: 1, Kind: theme.SyntaxKeyword},
			{Col: 5, Kind: theme.SyntaxPlain},
			{Col: 7, Kind: theme.SyntaxPunct},
			{Col: 9, Kind: theme.SyntaxNumber},
		},
	})

	return fv
}

// runOpener is the SGR prefix a syntax class renders with.
//
// Assertions match on it rather than on a whole styled token, because the LAST run on a line also absorbs the padding
// that fills the pane's width.
func runOpener(t *testing.T, kind theme.SyntaxKind) string {
	t.Helper()
	rendered := theme.Syntax(kind).Render("x")
	prefix, _, found := strings.Cut(rendered, "x")
	require.True(t, found)
	require.NotEmpty(t, prefix, "colour is off — see TestMain")

	return prefix
}

// The invariant the spec pane is held to, asserted here too: colouring changes how a line is drawn, never what it says.
func TestFileViewSyntax_TextSurvivesHighlighting(t *testing.T) {
	fv := highlightedFileView()

	plain := plainOf(fv.View(true, true))

	for _, want := range []string{"package p", "var x = 1"} {
		assert.Contains(t, plain, want)
	}
}

func TestFileViewSyntax_AppliesTheStyle(t *testing.T) {
	fv := highlightedFileView()

	view := fv.View(true, true)

	assert.Contains(t, view, runOpener(t, theme.SyntaxNumber)+"1",
		"the number run reaches the rendered output")
}

// The nav line answers a question the user asked, so it takes the whole row.
//
// Syntax colour underneath would break the bar up with its own resets.
func TestFileViewSyntax_NavLineWinsOverSyntax(t *testing.T) {
	fv := highlightedFileView()
	fv.GotoLine(2) // the only highlighted line

	view := fv.View(true, true)

	assert.NotContains(t, view, runOpener(t, theme.SyntaxNumber),
		"the cursor row is one bar, not a patchwork of syntax runs")
	assert.Contains(t, plainOf(view), "var x = 1", "and it still says what it said")
}

// A non-Go file gets no spans at all, and must render exactly as before.
func TestFileViewSyntax_NoSpansRendersPlain(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("go.mod", "module x\n\ngo 1.25\n")

	// The panel FRAME is styled either way, so compare the text rows alone.
	body := fv.viewerBody(false, false)

	assert.Equal(t, plainOf(body), body, "nothing in an unhighlighted viewer is styled")
	assert.Contains(t, body, "module x")
}

// Highlighting must not disturb the link gutter, which is prefixed before it.
func TestFileViewSyntax_CoexistsWithTheGutter(t *testing.T) {
	fv := highlightedFileView()
	fv.SetAnchors(map[int]bool{3: true}) // anchors are 1-based

	plain := plainOf(fv.View(false, false))

	assert.Contains(t, plain, string(GutterAnchor))
	assert.Contains(t, plain, "var x = 1")
}

// Spans are keyed by line, so scrolling must not shift them.
//
// The run installed for line 2 has to still land on line 2 once line 2 is drawn from an offset.
func TestFileViewSyntax_SurvivesScrolling(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 7) // 4 visible rows out of 6 lines, so the window must move
	fv.SetFile("x.go", "aaa\nbbb\nZZZ\nddd\neee\n")
	fv.SetSpans(map[int][]theme.Span{2: {{Col: 1, Kind: theme.SyntaxNumber}}})

	fv.GotoLine(3) // scrolls the window down, with the cursor NOT on line 2
	view := fv.View(false, false)
	require.Contains(t, plainOf(view), "ZZZ", "the target line is on screen")

	assert.Contains(t, view, runOpener(t, theme.SyntaxNumber)+"ZZZ",
		"the run must follow its line, not the window")
	assert.Equal(t, 1, strings.Count(plainOf(view), "ZZZ"))
}
