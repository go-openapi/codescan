// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The diagnostics pane was the last one where the view and the selection could
// drift apart: the wheel scrolled the viewport without moving the cursor, and
// every cursor step re-centred the whole list. These pin the corrected
// behaviour, matching the spec pane and the source viewer.

const diagNavCount = 30

// diagNavModel builds a model with many diagnostics over two files, so paging
// and file-switching both have somewhere to go.
func diagNavModel(t *testing.T) (*Model, string) {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(dir, "many.go")
	require.NoError(t, os.WriteFile(src, []byte(strings.Repeat("package p\n", diagNavCount+2)), 0o600))

	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics(), spec: panels.NewSpec()}
	m.cfg.WorkDir = dir
	m.diagH = 10
	m.diag.SetSize(80, 10)
	m.fileView.SetSize(80, 10)
	m.focused = paneDiag

	m.diags = make([]grammar.Diagnostic, 0, diagNavCount)
	for i := range diagNavCount {
		pos := token.Position{Filename: src, Line: i + 1, Column: 1}
		m.diags = append(m.diags, grammar.Warnf(pos, grammar.CodeInvalidNumber, "diag %d", i))
	}
	m.refreshDiagnostics()

	return m, src
}

func TestDiagNav_Paging(t *testing.T) {
	m, _ := diagNavModel(t)
	page := m.diag.VisibleRows()
	require.Positive(t, page)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, page, m.diagCursor, "page down moves a viewport's worth")

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Zero(t, m.diagCursor)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, diagNavCount-1, m.diagCursor, "end selects the last diagnostic")

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyHome})
	assert.Zero(t, m.diagCursor)
}

// The wheel must move the selection, not just the view — otherwise you can
// scroll away and then `f` or Enter acts on a diagnostic you cannot see.
func TestDiagNav_WheelMovesTheSelection(t *testing.T) {
	m, _ := diagNavModel(t)

	for range 3 {
		_, _ = m.handleMouse(tea.MouseMsg{
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
			X:      1,
			Y:      headerH + m.topH, // inside the diagnostics strip
		})
	}

	assert.Equal(t, 3, m.diagCursor, "the wheel carried the selection with it")
}

// Stepping the selection must not shift the whole list under the reader.
func TestDiagNav_ScrollsMinimally(t *testing.T) {
	m, _ := diagNavModel(t)
	page := m.diag.VisibleRows()

	// Moving inside the visible window leaves the viewport alone.
	before := m.diag.TopLine()
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, before, m.diag.TopLine(), "the target was already on screen")

	// Walking past the bottom edge scrolls by exactly one line at a time.
	for range page + 2 {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Positive(t, m.diag.TopLine(), "it did scroll once the selection left the window")
	assert.Less(t, m.diag.TopLine(), m.diagCursor+1, "and no further than needed")
}

// Enter is the one-shot counterpart to `f`: it goes to the source AND moves
// focus, where follow mode keeps the diagnostics pane driving.
func TestDiagNav_EnterJumpsToSource(t *testing.T) {
	m, src := diagNavModel(t)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, 2, m.diagCursor)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, src, m.currentFile, "the producing file is open")
	assert.Equal(t, modeView, m.leftMode)
	assert.Equal(t, paneTree, m.focused, "focus moved to the source, unlike follow mode")
	assert.Equal(t, 2, m.fileView.CurrentLine(), "on the diagnostic's line (3, 0-based 2)")
	assert.Contains(t, m.notice, "many.go:3")
}

func TestDiagNav_EnterWithoutAPosition(t *testing.T) {
	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics()}
	m.diagH = 10
	m.diag.SetSize(80, 10)
	m.focused = paneDiag
	m.diags = []grammar.Diagnostic{
		grammar.Warnf(token.Position{}, grammar.CodeInvalidNumber, "no position"),
	}
	m.refreshDiagnostics()

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Empty(t, m.currentFile, "nothing was opened")
	assert.Contains(t, m.notice, "no position")
}

func TestDiagNav_EnterWithNoDiagnostics(t *testing.T) {
	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics()}
	m.diag.SetSize(80, 10)
	m.focused = paneDiag

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Empty(t, m.currentFile)
	assert.Empty(t, m.notice)
}

// The pane still declines keys it does not own, so the globals keep working.
func TestDiagNav_UnownedKeysFallThrough(t *testing.T) {
	m, _ := diagNavModel(t)
	m.specJSON = `{"swagger":"2.0"}`
	m.refreshSpec()

	_, _ = m.handleKey(keyRune('h'))
	assert.True(t, m.helpOpen, "h still reaches the global help")

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	_, _ = m.handleKey(keyRune('r'))
	assert.True(t, m.scanning, "r still rescans")
}
