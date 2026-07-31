// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// These tests check both halves of the §6.5 contract: that the panels CHOOSE
// the right style for their role, and that the choice actually reaches the
// rendered output. The latter needs a forced colour profile — see TestMain.

// numberedContent returns n uniquely identifiable lines.
func numberedContent(n int) string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = "row" + strconv.Itoa(i)
	}
	return strings.Join(rows, "\n")
}

func TestTheme_DriverAndFollowerAreDistinct(t *testing.T) {
	assert.NotEqual(t,
		theme.Selected().GetBackground(), theme.Follower().GetBackground(),
		"the driver bar and the follower tint must be visually distinct (§6.5)")
}

func TestSpec_XrefStyleFollowsFocus(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("aaa\nbbb\nccc\nddd")
	sp.SetCursor(1)

	// The driver pane keeps focus in follow mode, so focused == drives.
	_ = sp.View(true)
	assert.Equal(t, theme.Selected().GetBackground(), sp.cursorStyle().GetBackground(),
		"a focused spec pane paints its xref line as the driver")

	_ = sp.View(false)
	assert.Equal(t, theme.Follower().GetBackground(), sp.cursorStyle().GetBackground(),
		"an unfocused spec pane is mirroring, so its xref line is a follower")
}

// A focus change must actually REPAINT: the style is baked into the viewport
// content at render time, so a missed re-render would leave the previous role's
// colour on screen even though xrefStyle() reports the new one.
func TestSpec_FocusChangeRepaints(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("aaa\nbbb\nccc\nddd")
	sp.SetCursor(1)

	driverView := sp.View(true)
	followerView := sp.View(false)

	require.Contains(t, driverView, theme.Selected().Render("bbb"),
		"the driver's xref line reaches the rendered output")
	require.Contains(t, followerView, theme.Follower().Render("bbb"),
		"the follower's xref line reaches the rendered output")
	assert.NotEqual(t, driverView, followerView, "the two roles must render differently")
}

// stylePrefix returns the SGR escape sequence a style emits, isolated from any
// text. Asserting on it pins WHICH style painted a line — comparing whole views
// would not, because the border and title also change with focus and would mask
// a nav line that never changed at all.
func stylePrefix(st lipgloss.Style) string {
	const sentinel = "\x00sentinel\x00"
	prefix, _, _ := strings.Cut(st.Render(sentinel), sentinel)
	return prefix
}

// The same for the source viewer: the nav line's style must reach the output,
// not merely be selected.
func TestFileView_NavStyleReachesOutput(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "line1\nline2\nline3\nline4")
	fv.GotoLine(1)

	driver, follower := stylePrefix(theme.Selected()), stylePrefix(theme.Follower())
	require.NotEqual(t, driver, follower, "precondition: the two styles emit different escapes")

	driverView := fv.View(true, true)
	assert.Contains(t, driverView, driver, "a focused viewer paints its nav line as the driver")
	assert.NotContains(t, driverView, follower)

	followerView := fv.View(false, true)
	assert.Contains(t, followerView, follower,
		"a mirroring viewer must not look like the pane the user is driving")
	assert.NotContains(t, followerView, driver)

	// With navActive false nothing is highlighted at all, so neither shows.
	plain := fv.View(false, false)
	assert.NotContains(t, plain, driver)
	assert.NotContains(t, plain, follower)
}

func TestSpec_HighlightLineCenters(t *testing.T) {
	sp := NewSpec()
	sp.SetSize(40, 13) // viewport height = 10
	sp.SetContent(numberedContent(60))

	sp.JumpTo(30)
	assert.Equal(t, 25, sp.TopLine(), "target - height/2")

	sp.JumpTo(2)
	assert.Equal(t, 0, sp.TopLine(), "clamped at the top rather than scrolling negative")
}

func TestFileView_NavStyleFollowsFocus(t *testing.T) {
	assert.Equal(t, theme.Selected().GetBackground(), navStyle(true).GetBackground(),
		"a focused viewer paints its nav line as the driver")
	assert.Equal(t, theme.Follower().GetBackground(), navStyle(false).GetBackground(),
		"an unfocused-but-mirroring viewer paints its nav line as a follower")
}

func TestFileView_GotoLineCenters(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 13) // visible = 10
	fv.SetFile("x.go", numberedContent(60))

	fv.GotoLine(30)
	assert.Equal(t, 30, fv.CurrentLine())
	assert.Equal(t, 25, fv.offset, "a jump centres its target, it does not merely reveal it")

	fv.GotoLine(1)
	assert.Equal(t, 0, fv.offset, "clamped at the top")
	fv.GotoLine(59)
	assert.Equal(t, 50, fv.offset, "clamped at the bottom (60 lines - 10 visible)")
}

// The nav keys must NOT centre: moving the cursor one line should scroll as
// little as possible, or the view lurches on every keypress.
func TestFileView_NavKeysScrollMinimally(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 13) // visible = 10
	fv.SetFile("x.go", numberedContent(60))

	for range 10 {
		fv.NavDown()
	}

	assert.Equal(t, 10, fv.CurrentLine())
	assert.Equal(t, 1, fv.offset,
		"the cursor stepped one line past the window, so the view scrolled by exactly one")
}
