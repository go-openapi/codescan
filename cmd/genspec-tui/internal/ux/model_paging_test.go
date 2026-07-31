// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Writing the keymap out is what exposed these: laid side by side, three panes
// were missing the paging the spec pane had, and `h` turned out to be taken.

// `h` is advertised in the header, so no pane may shadow it — least of all the
// source tree, which is where the app starts. The tree's vim-style h/l aliases
// for collapse/expand gave way to the arrows, which always worked.
func TestPaging_TreeDoesNotShadowTheHelpKey(t *testing.T) {
	m := New(codescan.Options{WorkDir: t.TempDir(), Packages: []string{"./..."}})
	t.Cleanup(m.Close)
	m.width, m.height = 100, 40
	m.focused, m.leftMode = paneTree, modeBrowse

	_, _ = m.handleKey(keyRune('h'))

	assert.True(t, m.helpOpen, "h opens the help in the pane the app starts in")
}

// The arrows still collapse and expand, and the help says so.
func TestPaging_TreeArrowsAreDocumented(t *testing.T) {
	body := stripANSI(strings.Join(helpLines(), "\n"))
	assert.Contains(t, body, "← →", "collapse/expand must be discoverable now that h/l are gone")
}

// viewerModel opens a long file in the read-only viewer.
func viewerModel(t *testing.T, lines int) *Model {
	t.Helper()

	var b strings.Builder
	for i := range lines {
		b.WriteString("line " + strconv.Itoa(i) + "\n")
	}
	path := filepath.Join(t.TempDir(), "long.go")
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o600))

	m := newHelpModel(t)
	m.topH = 20
	m.fileView.SetSize(80, 20)
	m.loadFileQuietly(path)
	m.focused, m.leftMode = paneTree, modeView

	return m
}

// A 500-line Go file one keypress at a time was the worst instance of the gap.
func TestPaging_FileViewer(t *testing.T) {
	m := viewerModel(t, 500)
	page := m.fileView.VisibleRows()
	require.Positive(t, page)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, page, m.fileView.CurrentLine())

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Zero(t, m.fileView.CurrentLine())

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, m.fileView.LastLine(), m.fileView.CurrentLine())

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, m.fileView.CurrentLine())
}

// Paging must not disturb the keys the viewer already owned.
func TestPaging_ViewerKeysStillWork(t *testing.T) {
	m := viewerModel(t, 50)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.fileView.CurrentLine())

	_, _ = m.handleKey(keyRune('i'))
	assert.True(t, m.fileView.Editing())

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.fileView.Editing())
}

func TestPaging_OptionsPopup(t *testing.T) {
	m := newOptionsModel(t)
	m.optionsOpen = true
	last := len(m.optToggles) - 1

	_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, last, m.optCursor)

	_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, m.optCursor)

	_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Positive(t, m.optCursor)
	assert.LessOrEqual(t, m.optCursor, last, "paging never runs off the end")

	// Clamped rather than wrapping, at both ends.
	for range len(m.optToggles) {
		_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	assert.Equal(t, last, m.optCursor)
	for range len(m.optToggles) {
		_, _ = m.handleOptionsKey(tea.KeyMsg{Type: tea.KeyPgUp})
	}
	assert.Zero(t, m.optCursor)
}

// Every navigable pane now pages. Stated as one test so a pane added later has
// an obvious place to be listed — and an obvious reason to support it.
func TestPaging_EveryNavigablePaneSupportsIt(t *testing.T) {
	body := stripANSI(strings.Join(helpLines(), "\n"))

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
