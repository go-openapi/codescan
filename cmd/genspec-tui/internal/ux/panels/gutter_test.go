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

// gutterMark is the marker as it appears in rendered output.
func gutterMark(r rune) string { return theme.Gutter().Render(string(r)) }

func TestSpec_GutterMarksOnlyTheGivenLines(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("aaa\nbbb\nccc\nddd")

	sp.SetGutter(map[int]rune{1: GutterAnchor, 2: GutterRef})
	sp.SetCursor(0) // keep the cursor off the lines under test

	view := sp.vp.View()
	require.Contains(t, view, gutterMark(GutterAnchor)+" bbb")
	require.Contains(t, view, gutterMark(GutterRef)+" ccc")
	assert.Contains(t, view, strings.Repeat(" ", gutterWidth)+"ddd",
		"unmarked lines are padded so the text stays aligned")
}

// No gutter installed means no gutter column: the pane costs no width before a scan has produced anything to mark.
func TestSpec_NoGutterCostsNoWidth(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("aaa\nbbb")

	// The viewport pads each line to its width, so compare prefixes: the text must start in column 0, not two columns in.
	// Line 1 is used throughout so the always-rendered cursor (line 0) does not wrap the line under test.
	sp.SetCursor(0)
	secondLine := func() string { return strings.Split(sp.vp.View(), "\n")[1] }

	assert.True(t, strings.HasPrefix(secondLine(), "bbb"),
		"content starts in column 0 when nothing is marked, got %q", secondLine())

	sp.SetGutter(nil)
	assert.True(t, strings.HasPrefix(secondLine(), "bbb"),
		"an explicitly nil gutter costs no width either, got %q", secondLine())

	// ...whereas installing one shifts the text right by the gutter width.
	sp.SetGutter(map[int]rune{0: GutterAnchor})
	assert.True(t, strings.HasPrefix(secondLine(), strings.Repeat(" ", gutterWidth)+"bbb"),
		"got %q", secondLine())
}

// The gutter is prefixed after highlighting.
//
// So both survive together, and the styles still apply to the text rather than to the marker column.
func TestSpec_GutterCoexistsWithSearchAndCursor(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("aaa\nbbb\nbcd")
	sp.SetGutter(map[int]rune{0: GutterAnchor, 2: GutterAnchor})

	n := sp.Search("b")
	require.Equal(t, 2, n, "the gutter must not disturb match counting")
	require.Equal(t, 1, sp.CursorLine(), "the search parked the cursor on the first match")

	view := sp.View(true)
	assert.Contains(t, view, gutterMark(GutterAnchor), "markers survive a search render")
	assert.Contains(t, view, theme.Match().Render("b"),
		"a match the cursor is NOT on keeps its substring highlight")
	assert.Contains(t, view, theme.Selected().Render("bbb"),
		"the cursor line takes the whole-line bar, and the style wraps the text "+
			"rather than the gutter")
}

func TestFileView_GutterMarksAnchoredLines(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "line1\nline2\nline3\nline4")
	fv.SetAnchors(map[int]bool{2: true}) // 1-based, matching token.Position

	view := fv.View(false, false)

	assert.Contains(t, view, gutterMark(GutterAnchor)+" 2 line2",
		"the anchored source line is marked")
	assert.Contains(t, view, strings.Repeat(" ", gutterWidth)+"1 line1",
		"other lines are padded, keeping the line numbers aligned")
}

func TestFileView_NoAnchorsNoGutter(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "line1\nline2")

	view := fv.View(false, false)

	assert.NotContains(t, view, gutterMark(GutterAnchor))
	assert.Contains(t, view, "1 line1", "the line numbers keep their original column")
}
