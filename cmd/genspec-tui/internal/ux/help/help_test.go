// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package help

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The arrows still collapse and expand, and the help says so.
func TestPaging_TreeArrowsAreDocumented(t *testing.T) {
	body := testutils.StripANSI(strings.Join(helpLines(), "\n"))
	assert.Contains(t, body, "← →", "collapse/expand must be discoverable now that h/l are gone")
}

// The overlay is the only in-app record of the keymap, so a binding that gets dispatched but never listed is invisible.
//
// This is a coarse guard — it cannot prove completeness — but it fails if a documented key is dropped.
func TestHelp_ListsTheDispatchedBindings(t *testing.T) {
	body := testutils.StripANSI(strings.Join(helpLines(), "\n"))

	for _, k := range []string{
		"h", "?", "tab", "shift+tab", "c", "r", "o", "ctrl+q",
		"j k", "pgup", "home", "ctrl+j", "ctrl+y", "/", "n  N", "f",
		"F3", "shift+F3", "enter", "esc",
		"g", "i", "ctrl+f", "ctrl+s", "space",
	} {
		assert.Contains(t, body, k, "binding %q is dispatched but not in the help", k)
	}
}

// Every navigable pane pages.
//
// Stated as one test so a pane added later has an obvious place to be listed — and an obvious reason to support it.
func TestPaging_EveryNavigablePaneSupportsIt(t *testing.T) {
	body := testutils.StripANSI(strings.Join(helpLines(), "\n"))

	for _, section := range []string{"spec pane", "source tree", "file viewer", "diagnostics"} {
		idx := strings.Index(body, section)
		require.Positive(t, idx, "section %q", section)

		rest := body[idx:]
		if next := strings.Index(rest[len(section):], "\n\n"); next > 0 {
			rest = rest[:len(section)+next]
		}
		assert.Contains(t, rest, "pgup", "section %q must offer paging", section)
		assert.Contains(t, rest, "home", "section %q must offer home/end", section)
	}
}

func TestHelp_ContentIsWellFormed(t *testing.T) {
	require.NotEmpty(t, helpSections)

	titles := make(map[string]bool, len(helpSections))
	for _, sec := range helpSections {
		assert.NotEmpty(t, sec.title)
		assert.False(t, titles[sec.title], "duplicate section %q", sec.title)
		titles[sec.title] = true
		assert.NotEmpty(t, sec.entries, "section %q is empty", sec.title)

		keys := make(map[string]bool, len(sec.entries))
		for _, e := range sec.entries {
			assert.NotEmpty(t, e.keys, "section %q has an entry with no keys", sec.title)
			assert.NotEmpty(t, e.action, "entry %q in %q has no action", e.keys, sec.title)
			assert.False(t, keys[e.keys], "duplicate entry %q in section %q", e.keys, sec.title)
			keys[e.keys] = true
		}
	}
}

// The overlay is opened to look something up, not resumed, so it always starts at the top.
func TestHelp_OpensAtTheTop(t *testing.T) {
	o := newOverlay(20)
	o.Open()
	o.scrollBy(+5)
	require.Positive(t, o.scroll)

	o.Close()
	o.Open()

	assert.Zero(t, o.scroll)
}

func TestHelp_ClosesOnItsDismissKeys(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		testutils.KeyRune('h'),
		testutils.KeyRune('?'),
		{Type: tea.KeyEnter},
	} {
		o := newOverlay(20)
		o.Open()
		require.True(t, o.IsOpen())

		require.Nil(t, o.HandleKey(msg))

		assert.False(t, o.IsOpen(), "%v closes the help", msg)
	}
}

func TestHelp_Scrolls(t *testing.T) {
	o := newOverlay(20) // fewer visible rows than the keymap has
	o.Open()
	require.Greater(t, len(helpLines()), o.visibleRows(), "precondition: the keymap overflows")

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, o.scroll)

	// Clamped at the top...
	for range 10 {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	}
	assert.Zero(t, o.scroll)

	// ...and at the bottom.
	maxScroll := len(helpLines()) - o.visibleRows()
	for range len(helpLines()) + 5 {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, maxScroll, o.scroll)

	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, o.scroll)
	_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, maxScroll, o.scroll)
}

// A keymap that does not fit and cannot scroll would silently hide bindings.
func TestHelp_ShortTerminalStillReachesTheEnd(t *testing.T) {
	o := newOverlay(14)
	o.Open()

	for range len(helpLines()) {
		_ = o.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	}

	// The last ENTRY, not the last section title: on a very short window the title itself scrolls off while its rows are
	// still on screen, and it is the rows that must stay reachable.
	lastSection := helpSections[len(helpSections)-1]
	lastEntry := lastSection.entries[len(lastSection.entries)-1]
	assert.Contains(t, testutils.StripANSI(o.View()), lastEntry.action,
		"the end of the keymap must be reachable by scrolling")
}

func newOverlay(height int) *Overlay {
	o := New()
	o.SetSize(100, height)

	return &o
}
