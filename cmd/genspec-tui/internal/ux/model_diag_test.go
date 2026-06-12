// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestDiag_MoveCursorClamps(t *testing.T) {
	m := &Model{diag: panels.NewDiagnostics()}
	m.diag.SetSize(60, 8)
	m.diags = make([]grammar.Diagnostic, 3)

	m.moveDiagCursor(+1)
	assert.Equal(t, 1, m.diagCursor)
	m.moveDiagCursor(+5)
	assert.Equal(t, 2, m.diagCursor, "clamped at the last diagnostic")
	m.moveDiagCursor(-9)
	assert.Equal(t, 0, m.diagCursor, "clamped at the first")

	// No diagnostics: a no-op, no panic.
	empty := &Model{diag: panels.NewDiagnostics()}
	empty.moveDiagCursor(+1)
	assert.Equal(t, 0, empty.diagCursor)
}

func TestDiag_FollowModeTracksSelection(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.go")
	b := filepath.Join(dir, "b.go")
	require.NoError(t, os.WriteFile(a, []byte("package p\n\ntype X struct{}\n"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("package p\n\n\n\ntype Y struct{}\n"), 0o600))

	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics()}
	m.cfg.WorkDir = dir
	m.fileView.SetSize(40, 12)
	m.diag.SetSize(60, 8)
	m.diags = []grammar.Diagnostic{
		grammar.Warnf(pos(a, 3, 1), grammar.CodeInvalidNumber, "one"),
		grammar.Warnf(pos(b, 5, 1), grammar.CodeInvalidNumber, "two"),
	}
	m.focused = paneDiag

	// Entering follow mode mirrors the first diagnostic; the driver keeps focus.
	m.toggleFollow(followDiag)
	assert.Equal(t, followDiag, m.follow)
	assert.Equal(t, paneDiag, m.focused, "the diagnostics pane stays the driver")
	assert.Equal(t, a, m.currentFile)
	assert.Equal(t, modeView, m.leftMode)
	assert.Equal(t, 2, m.fileView.CurrentLine(), "line 3 → row 2")

	// Moving the selection auto-tracks the source pane (the Update loop re-syncs).
	m.moveDiagCursor(+1)
	m.syncFollowIfActive()
	assert.Equal(t, b, m.currentFile, "source follows to the second diagnostic's file")
	assert.Equal(t, 4, m.fileView.CurrentLine(), "line 5 → row 4")

	// A second `f` toggles off.
	m.toggleFollow(followDiag)
	assert.Equal(t, followOff, m.follow)
}

func TestDiag_FollowExitsOnFocusChange(t *testing.T) {
	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics()}
	m.fileView.SetSize(40, 12)
	m.diag.SetSize(60, 8)
	m.diags = make([]grammar.Diagnostic, 2)
	m.focused = paneDiag
	m.follow = followDiag

	m.focused = paneSpec // tab/click away from the driver
	m.syncFollowIfActive()
	assert.Equal(t, followOff, m.follow, "leaving the driver pane exits follow")
}

func TestDiag_FollowNoPosition(t *testing.T) {
	m := &Model{fileView: panels.NewFileView(), diag: panels.NewDiagnostics()}
	m.fileView.SetSize(40, 10)
	m.diag.SetSize(60, 8)
	m.diags = []grammar.Diagnostic{{Message: "no position"}} // zero Pos is invalid
	m.focused = paneDiag

	m.toggleFollow(followDiag)
	assert.Equal(t, followDiag, m.follow)
	assert.Empty(t, m.currentFile, "nothing opened when the diagnostic has no source")
	assert.Equal(t, "(no source)", m.followTarget)
}

func TestRenderDiagnostics_SelectedLine(t *testing.T) {
	diags := []grammar.Diagnostic{
		grammar.Warnf(pos("/w/a.go", 1, 1), grammar.CodeInvalidNumber, "one"),
		grammar.Warnf(pos("/w/a.go", 2, 1), grammar.CodeInvalidNumber, "two"),
	}
	// tally on line 0, first diagnostic on line 1, second on line 2.
	_, line := renderDiagnostics("/w", nil, diags, 1)
	assert.Equal(t, 2, line)
}
