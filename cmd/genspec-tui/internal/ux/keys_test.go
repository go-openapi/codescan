// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for key dispatch: what the overlays capture, what falls through to the global bindings, and the paging keys
// every navigable pane owes the user.

package ux

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestHelp_OpensOnHAndQuestionMark(t *testing.T) {
	for _, k := range []rune{'h', '?'} {
		m := newHelpModel(t)

		_ = m.handleKey(testutils.KeyRune(k))

		assert.True(t, m.help.IsOpen(), "%q opens the help", k)
		assert.Contains(t, testutils.StripANSI(m.help.View()), "Key bindings")
	}
}

func TestHelp_Closes(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		testutils.KeyRune('h'),
		testutils.KeyRune('?'),
		{Type: tea.KeyEnter},
	} {
		m := newHelpModel(t)
		_ = m.handleKey(testutils.KeyRune('h'))
		require.True(t, m.help.IsOpen())

		_ = m.handleKey(msg)

		assert.False(t, m.help.IsOpen(), "%v closes the help", msg)
	}
}

// While the overlay covers the UI, acting on a key whose effect the user cannot see would be worse than ignoring it.
func TestHelp_SwallowsOtherKeys(t *testing.T) {
	m := newHelpModel(t)
	_ = m.handleKey(testutils.KeyRune('h'))

	for _, msg := range []tea.KeyMsg{testutils.KeyRune('r'), testutils.KeyRune('o'), testutils.KeyRune('/'), {Type: tea.KeyF3}} {
		_ = m.handleKey(msg)
	}

	assert.True(t, m.help.IsOpen(), "still open")
	assert.False(t, m.scan.Running, "r did not start a scan")
	assert.False(t, m.options.IsOpen(), "o did not open the options")
	assert.False(t, m.search.Active(), "/ did not open search")
}

// h is an ordinary character in the editor; opening help there would make the buffer unusable.
func TestHelp_DoesNotHijackTheEditor(t *testing.T) {
	m := newHelpModel(t)
	m.loadFileQuietly(testutils.WriteTempGo(t, "package p\n"))
	m.focused, m.leftMode = paneTree, modeView
	_ = m.fileView.StartEdit()
	require.True(t, m.fileView.Editing())

	_ = m.handleKey(testutils.KeyRune('h'))

	assert.False(t, m.help.IsOpen(), "the editor keeps plain h for typing")
	assert.Contains(t, m.fileView.Value(), "h", "and the character reached the buffer")
}

// ...but the read-only viewer passes it through, like the other global keys.
func TestHelp_OpensFromTheReadOnlyViewer(t *testing.T) {
	m := newHelpModel(t)
	m.loadFileQuietly(testutils.WriteTempGo(t, "package p\n"))
	m.focused, m.leftMode = paneTree, modeView
	require.False(t, m.fileView.Editing())

	_ = m.handleKey(testutils.KeyRune('h'))

	assert.True(t, m.help.IsOpen())
}

// Quitting is decided above the overlays, so it works from inside one.
func TestHelp_QuitsFromTheOverlay(t *testing.T) {
	m := newHelpModel(t)
	_ = m.handleKey(testutils.KeyRune('h'))
	require.True(t, m.help.IsOpen())

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlQ})

	require.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd(), "ctrl+q quits from inside the help overlay")
}

func TestHelp_BannerIsInTheHeader(t *testing.T) {
	m := newHelpModel(t)
	m.cfg.WorkDir = "/a/very/long/path/that/would/otherwise/crowd/the/header/line/out"

	header := testutils.StripANSI(m.headerLine())

	assert.Contains(t, header, "h: help",
		"the banner must survive a long work dir — it is what reveals every other key")
	assert.Less(t, strings.Index(header, "h: help"), strings.Index(header, "JSON"),
		"and sit early in the line, where nothing can push it off")
}

func TestOptions_OpensOnO(t *testing.T) {
	m := newOptionsModel(t)

	_ = m.handleKey(testutils.KeyRune('o'))

	assert.True(t, m.options.IsOpen())
}

func TestOptions_ToggleAndApply(t *testing.T) {
	m := newOptionsModel(t)
	_ = m.handleKey(testutils.KeyRune('o'))
	require.False(t, m.cfg.ScanModels)

	_ = m.handleKey(testutils.KeyRune(' '))
	assert.True(t, m.cfg.ScanModels, "space toggles the row under the cursor, straight into the scan config")

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, m.options.IsOpen())
	assert.NotNil(t, cmd, "a changed option triggers a rescan on close")
	assert.True(t, m.scan.Running, "and the rescan is under way")
}

func TestOptions_CloseWithoutChangesDoesNotRescan(t *testing.T) {
	m := newOptionsModel(t)
	_ = m.handleKey(testutils.KeyRune('o'))

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, m.options.IsOpen())
	assert.Nil(t, cmd, "nothing changed, so nothing to re-run")
	assert.False(t, m.scan.Running)
}

// A change is applied ONCE.
//
// The dirty flag has to outlive the close for the model to see it, so something must then say it has been acted on
// - otherwise the next keystroke that reaches any overlay asks for the same rescan all over again.
func TestOptions_ChangeIsAppliedOnlyOnce(t *testing.T) {
	m := newOptionsModel(t)
	_ = m.handleKey(testutils.KeyRune('o'))
	_ = m.handleKey(testutils.KeyRune(' '))
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "precondition: closing with a change rescans")
	m.scan.Running = false // as if that scan had finished

	_ = m.handleKey(testutils.KeyRune('h'))
	cmd = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})

	assert.Nil(t, cmd, "scrolling the help must not re-trigger the options rescan")
	assert.False(t, m.scan.Running)
}

// The overlay covers the UI, so the keys behind it must not fire.
func TestOptions_SwallowsOtherKeys(t *testing.T) {
	m := newOptionsModel(t)
	_ = m.handleKey(testutils.KeyRune('o'))

	for _, msg := range []tea.KeyMsg{testutils.KeyRune('r'), testutils.KeyRune('/'), testutils.KeyRune('h')} {
		_ = m.handleKey(msg)
	}

	assert.True(t, m.options.IsOpen(), "still open")
	assert.False(t, m.scan.Running, "r did not start a scan")
	assert.False(t, m.search.Active(), "/ did not open search")
	assert.False(t, m.help.IsOpen(), "h did not open the help behind it")
}

// Quitting is decided above the overlays.
//
// So ctrl+q means the same thing here as everywhere else. It used to close the popup and rescan instead.
func TestOptions_QuitsFromTheOverlay(t *testing.T) {
	m := newOptionsModel(t)
	_ = m.handleKey(testutils.KeyRune('o'))
	require.True(t, m.options.IsOpen())

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlQ})

	require.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())
}

// Writing the keymap out is what exposed these: laid side by side, three panes were missing the paging the spec pane
// had, and `h` turned out to be taken.

// h is advertised in the header, so no pane may shadow it.
//
// Least of all the source tree, which is where the app starts.
//
// The tree's vim-style h/l aliases for collapse or expand gave way to the arrows, which always worked.
func TestPaging_TreeDoesNotShadowTheHelpKey(t *testing.T) {
	m := testModel(t, sized(100, 40), focusedOn(paneTree))
	m.leftMode = modeBrowse

	_ = m.handleKey(testutils.KeyRune('h'))

	assert.True(t, m.help.IsOpen(), "h opens the help in the pane the app starts in")
}

// A 500-line Go file one keypress at a time was the worst instance of the gap.
func TestPaging_FileViewer(t *testing.T) {
	m := viewerModel(t, 500)
	page := m.fileView.VisibleRows()
	require.Positive(t, page)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, page, m.fileView.CurrentLine())

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Zero(t, m.fileView.CurrentLine())

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, m.fileView.LastLine(), m.fileView.CurrentLine())

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, m.fileView.CurrentLine())
}

// Paging must not disturb the keys the viewer already owned.
func TestPaging_ViewerKeysStillWork(t *testing.T) {
	m := viewerModel(t, 50)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.fileView.CurrentLine())

	_ = m.handleKey(testutils.KeyRune('i'))
	assert.True(t, m.fileView.Editing())

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.fileView.Editing())
}

// The keymap's own record that every navigable pane pages lives with the keymap, in the help package.

func newHelpModel(t *testing.T) *Model {
	t.Helper()

	return testModel(t, sized(100, 40))
}

// What the rows are and how they render is the overlay's own business, and is tested there. What the model owes it is
// the wiring: `o` opens it, keys reach it, and a change is APPLIED - the overlay records the intent, the model decides
// a rescan is what carrying it out means.

func newOptionsModel(t *testing.T) *Model {
	t.Helper()
	// Tall enough that nothing scrolls unless a test wants it to.
	return testModel(t, sized(100, 40))
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
