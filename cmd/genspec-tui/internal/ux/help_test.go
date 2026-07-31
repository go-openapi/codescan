// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func newHelpModel(t *testing.T) *Model {
	t.Helper()
	m := New(codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}})
	t.Cleanup(m.Close)
	m.width, m.height = 100, 40
	m.ready = true

	return m
}

func keyRune(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// writeTempGo puts a Go file on disk and returns its path.
func writeTempGo(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestHelp_OpensOnHAndQuestionMark(t *testing.T) {
	for _, k := range []rune{'h', '?'} {
		m := newHelpModel(t)

		_, _ = m.handleKey(keyRune(k))

		assert.True(t, m.helpOpen, "%q opens the help", k)
		assert.Contains(t, stripANSI(m.View()), "Key bindings")
	}
}

// The overlay is opened to look something up, not resumed, so it always starts
// at the top.
func TestHelp_OpensAtTheTop(t *testing.T) {
	m := newHelpModel(t)
	m.helpOpen = true
	m.scrollHelp(+5)
	require.Positive(t, m.helpScroll)

	m.helpOpen = false
	_, _ = m.handleKey(keyRune('h'))

	assert.Zero(t, m.helpScroll)
}

func TestHelp_Closes(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		keyRune('h'),
		keyRune('?'),
		{Type: tea.KeyEnter},
	} {
		m := newHelpModel(t)
		_, _ = m.handleKey(keyRune('h'))
		require.True(t, m.helpOpen)

		_, _ = m.handleKey(msg)

		assert.False(t, m.helpOpen, "%v closes the help", msg)
	}
}

// While the overlay covers the UI, acting on a key whose effect the user cannot
// see would be worse than ignoring it.
func TestHelp_SwallowsOtherKeys(t *testing.T) {
	m := newHelpModel(t)
	_, _ = m.handleKey(keyRune('h'))

	for _, msg := range []tea.KeyMsg{keyRune('r'), keyRune('o'), keyRune('/'), {Type: tea.KeyF3}} {
		_, _ = m.handleKey(msg)
	}

	assert.True(t, m.helpOpen, "still open")
	assert.False(t, m.scanning, "r did not start a scan")
	assert.False(t, m.optionsOpen, "o did not open the options")
	assert.False(t, m.searching, "/ did not open search")
}

// `h` is an ordinary character in the editor; opening help there would make the
// buffer unusable.
func TestHelp_DoesNotHijackTheEditor(t *testing.T) {
	m := newHelpModel(t)
	m.loadFileQuietly(writeTempGo(t, "package p\n"))
	m.focused, m.leftMode = paneTree, modeView
	_ = m.fileView.StartEdit()
	require.True(t, m.fileView.Editing())

	_, _ = m.handleKey(keyRune('h'))

	assert.False(t, m.helpOpen, "the editor keeps plain h for typing")
	assert.Contains(t, m.fileView.Value(), "h", "and the character reached the buffer")
}

// ...but the read-only viewer passes it through, like the other global keys.
func TestHelp_OpensFromTheReadOnlyViewer(t *testing.T) {
	m := newHelpModel(t)
	m.loadFileQuietly(writeTempGo(t, "package p\n"))
	m.focused, m.leftMode = paneTree, modeView
	require.False(t, m.fileView.Editing())

	_, _ = m.handleKey(keyRune('h'))

	assert.True(t, m.helpOpen)
}

func TestHelp_Scrolls(t *testing.T) {
	m := newHelpModel(t)
	m.height = 20 // fewer visible rows than the keymap has
	_, _ = m.handleKey(keyRune('h'))
	require.Greater(t, len(helpLines()), m.helpVisibleRows(), "precondition: the keymap overflows")

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.helpScroll)

	// Clamped at the top...
	for range 10 {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	}
	assert.Zero(t, m.helpScroll)

	// ...and at the bottom.
	maxScroll := len(helpLines()) - m.helpVisibleRows()
	for range len(helpLines()) + 5 {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, maxScroll, m.helpScroll)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, m.helpScroll)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, maxScroll, m.helpScroll)
}

// A keymap that does not fit and cannot scroll would silently hide bindings.
func TestHelp_ShortTerminalStillReachesTheEnd(t *testing.T) {
	m := newHelpModel(t)
	m.height = 14
	_, _ = m.handleKey(keyRune('h'))

	for range len(helpLines()) {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	}

	// The last ENTRY, not the last section title: on a very short window the
	// title itself scrolls off while its rows are still on screen, and it is the
	// rows that must stay reachable.
	lastSection := helpSections[len(helpSections)-1]
	lastEntry := lastSection.entries[len(lastSection.entries)-1]
	assert.Contains(t, stripANSI(m.helpView()), lastEntry.action,
		"the end of the keymap must be reachable by scrolling")
}

func TestHelp_BannerIsInTheHeader(t *testing.T) {
	m := newHelpModel(t)
	m.cfg.WorkDir = "/a/very/long/path/that/would/otherwise/crowd/the/header/line/out"

	header := stripANSI(m.headerLine())

	assert.Contains(t, header, "h: help",
		"the banner must survive a long work dir — it is what reveals every other key")
	assert.Less(t, strings.Index(header, "h: help"), strings.Index(header, "JSON"),
		"and sit early in the line, where nothing can push it off")
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

// The overlay is the only in-app record of the keymap, so a binding that gets
// dispatched but never listed is invisible. This is a coarse guard — it cannot
// prove completeness — but it fails if a documented key is dropped.
func TestHelp_ListsTheDispatchedBindings(t *testing.T) {
	body := stripANSI(strings.Join(helpLines(), "\n"))

	for _, k := range []string{
		"h", "?", "tab", "shift+tab", "c", "r", "o", "ctrl+q",
		"j k", "pgup", "home", "ctrl+j", "ctrl+y", "/", "n  N", "f",
		"F3", "shift+F3", "enter", "esc",
		"g", "i", "ctrl+f", "ctrl+s", "space",
	} {
		assert.Contains(t, body, k, "binding %q is dispatched but not in the help", k)
	}
}
