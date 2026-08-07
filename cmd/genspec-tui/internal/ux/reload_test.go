// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// reloadFixture writes a file and opens it in the viewer.
//
// It hands back the path so a test can rewrite it behind the TUI's back,
// which is the whole situation reload exists for.
type reloadFixture struct {
	m    *Model
	path string
}

func newReloadFixture(t *testing.T, body string) reloadFixture {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "user.go")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	m := testModelIn(t, dir, panelSize(40, 12), openFile(path), focusedOn(paneTree))
	require.Equal(t, modeView, m.leftMode)

	return reloadFixture{m: m, path: path}
}

// rewrite replaces the file on disk, as an external editor or a git checkout would.
func (f reloadFixture) rewrite(t *testing.T, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(f.path, []byte(body), 0o600))
}

// pressF5 sends the reload key through the real dispatch rather than calling the helper, so the binding is covered too.
func (f reloadFixture) pressF5() tea.Cmd {
	return f.m.handleKey(tea.KeyMsg{Type: tea.KeyF5})
}

const (
	reloadBefore = "package p\n\ntype X struct{}\n"
	reloadAfter  = "package p\n\ntype X struct{}\n\ntype Y struct{}\n"
)

// TestReload_CleanBufferSkipsTheGuard pins that a reload with nothing to lose just happens.
//
// The confirmation is there to protect unsaved work; asking when there is none would be a prompt that trains people to
// dismiss prompts.
func TestReload_CleanBufferSkipsTheGuard(t *testing.T) {
	f := newReloadFixture(t, reloadBefore)
	f.rewrite(t, reloadAfter)

	_ = f.pressF5()

	assert.False(t, f.m.confirm.IsOpen(), "a clean buffer is not worth a question")
	assert.Contains(t, f.m.fileView.Value(), "type Y struct{}", "the new content is in the buffer")
	assert.False(t, f.m.fileView.Dirty(), "the reloaded buffer is the new baseline")
}

// TestReload_DirtyBufferAsksFirst pins the guard: the file on disk must not reach the buffer until the user says so.
func TestReload_DirtyBufferAsksFirst(t *testing.T) {
	f := newReloadFixture(t, reloadBefore)
	f.m.fileView.StartEdit()
	_ = f.m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	require.True(t, f.m.fileView.Dirty())
	f.rewrite(t, reloadAfter)

	_ = f.pressF5()

	require.True(t, f.m.confirm.IsOpen(), "unsaved edits must be confirmed away")
	assert.NotContains(t, f.m.fileView.Value(), "type Y struct{}",
		"nothing is reloaded while the question is still on screen")
	assert.Contains(t, f.m.fileView.Value(), "zzz", "the edit is still there")
}

// TestReload_DeclineChangesNothing pins that no is a true no-op.
//
// The edit survives, and the action does not stay armed for the next keypress.
func TestReload_DeclineChangesNothing(t *testing.T) {
	f := newReloadFixture(t, reloadBefore)
	f.m.fileView.StartEdit()
	_ = f.m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	f.rewrite(t, reloadAfter)
	_ = f.pressF5()
	require.True(t, f.m.confirm.IsOpen())

	_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	assert.False(t, f.m.confirm.IsOpen(), "answering closes the question")
	assert.Contains(t, f.m.fileView.Value(), "zzz", "the edit survives a decline")
	assert.NotContains(t, f.m.fileView.Value(), "type Y struct{}")
	assert.Equal(t, confirmNothing, f.m.pendingConfirm, "a declined action must not stay armed")
}

// TestReload_AcceptDiscardsAndReloads pins the other branch of the guard.
func TestReload_AcceptDiscardsAndReloads(t *testing.T) {
	f := newReloadFixture(t, reloadBefore)
	f.m.fileView.StartEdit()
	_ = f.m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	f.rewrite(t, reloadAfter)
	_ = f.pressF5()
	require.True(t, f.m.confirm.IsOpen())

	_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.False(t, f.m.confirm.IsOpen())
	assert.Contains(t, f.m.fileView.Value(), "type Y struct{}", "disk won")
	assert.NotContains(t, f.m.fileView.Value(), "zzz", "the edit was discarded")
	assert.False(t, f.m.fileView.Dirty(), "the reloaded buffer is the new baseline")
	assert.False(t, f.m.fileView.Editing(), "reload lands in the read-only viewer, not back in the editor")
	assert.Equal(t, confirmNothing, f.m.pendingConfirm)
}

// TestReload_KeepsThePlace pins that reload does not send the reader back to line 1.
//
// SetFile resets the nav line to the top, which is right when OPENING a file and wrong when re-reading the one already
// on screen.
func TestReload_KeepsThePlace(t *testing.T) {
	body := "package p\n\n// 3\n// 4\n// 5\n// 6\n// 7\n// 8\n"
	f := newReloadFixture(t, body)
	f.m.fileView.GotoLine(6) // 0-based: the "// 7" line
	require.Equal(t, 6, f.m.fileView.CurrentLine())
	f.rewrite(t, body+"// 9\n")

	_ = f.pressF5()

	assert.Equal(t, 6, f.m.fileView.CurrentLine(), "the reader stays where they were reading")
}

// TestReload_NoFileOpen pins that the key is harmless with nothing to reload.
func TestReload_NoFileOpen(t *testing.T) {
	m := testModel(t, panelSize(40, 12))
	require.Empty(t, m.currentFile)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF5})

	assert.False(t, m.confirm.IsOpen(), "nothing to ask about")
}

// TestReload_FromInsideTheEditor pins that F5 reaches the reload while the editor is capturing keys.
//
// That is the state the guard exists for, so a binding that only worked from the viewer would miss its own use case.
func TestReload_FromInsideTheEditor(t *testing.T) {
	f := newReloadFixture(t, reloadBefore)
	f.m.fileView.StartEdit()
	_ = f.m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zzz")})
	require.True(t, f.m.fileView.Editing())

	_ = f.pressF5()

	assert.True(t, f.m.confirm.IsOpen(), "F5 is not swallowed by the textarea")
}
