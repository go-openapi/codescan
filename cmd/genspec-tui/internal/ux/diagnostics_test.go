// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for the diagnostics pane: selection, paging, follow, and how a diagnostic list renders.

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/diagnostics"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestDiag_MoveCursorClamps(t *testing.T) {
	m := testModel(t, diagSize(60, 8), withDiags(make([]grammar.Diagnostic, 3)...))

	m.moveDiagCursor(+1)
	assert.Equal(t, 1, m.diagCursor)
	m.moveDiagCursor(+5)
	assert.Equal(t, 2, m.diagCursor, "clamped at the last diagnostic")
	m.moveDiagCursor(-9)
	assert.Equal(t, 0, m.diagCursor, "clamped at the first")

	// No diagnostics: a no-op, no panic.
	empty := testModel(t, diagSize(60, 8))
	empty.moveDiagCursor(+1)
	assert.Equal(t, 0, empty.diagCursor)
}

func TestDiag_FollowModeTracksSelection(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.go")
	b := filepath.Join(dir, "b.go")
	require.NoError(t, os.WriteFile(a, []byte("package p\n\ntype X struct{}\n"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("package p\n\n\n\ntype Y struct{}\n"), 0o600))

	m := testModelIn(t, dir,
		panelSize(40, 12),
		diagSize(60, 8),
		focusedOn(paneDiag),
		withDiags(
			grammar.Warnf(pos(a, 3, 1), grammar.CodeInvalidNumber, "one"),
			grammar.Warnf(pos(b, 5, 1), grammar.CodeInvalidNumber, "two"),
		),
	)

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
	m := testModel(t,
		panelSize(40, 12),
		diagSize(60, 8),
		focusedOn(paneDiag),
		withDiags(make([]grammar.Diagnostic, 2)...),
	)
	m.follow = followDiag

	m.focused = paneSpec // tab/click away from the driver
	m.syncFollowIfActive()
	assert.Equal(t, followOff, m.follow, "leaving the driver pane exits follow")
}

func TestDiag_FollowNoPosition(t *testing.T) {
	m := testModel(t,
		panelSize(40, 10),
		diagSize(60, 8),
		focusedOn(paneDiag),
		withDiags(grammar.Diagnostic{Message: "no position"}), // zero Pos is invalid
	)

	m.toggleFollow(followDiag)
	assert.Equal(t, followDiag, m.follow)
	assert.Empty(t, m.currentFile, "nothing opened when the diagnostic has no source")
	assert.Equal(t, "(diagnostic carries no position)", m.followTarget,
		"a positionless diagnostic is a different miss from an unanchored spec node")
}

func TestRenderDiagnostics_SelectedLine(t *testing.T) {
	diags := []grammar.Diagnostic{
		grammar.Warnf(pos("/w/a.go", 1, 1), grammar.CodeInvalidNumber, "one"),
		grammar.Warnf(pos("/w/a.go", 2, 1), grammar.CodeInvalidNumber, "two"),
	}
	// tally on line 0, first diagnostic on line 1, second on line 2.
	_, line := diagnostics.Render("/w", nil, diags, 1, true)
	assert.Equal(t, 2, line)
}

// Spotted in a screenshot: the spec cursor and the selected diagnostic were both drawn with the strong bar, so two
// panes appeared to be driving at once.
//
// The rule everywhere else is focused = strong, unfocused = muted tint.
func TestDiag_SelectionDimsWhenUnfocused(t *testing.T) {
	diags := []grammar.Diagnostic{
		grammar.Warnf(pos("a.go", 3, 1), grammar.CodeInvalidNumber, "one"),
		grammar.Warnf(pos("b.go", 5, 1), grammar.CodeInvalidNumber, "two"),
	}

	focused, _ := diagnostics.Render("", nil, diags, 1, true)
	unfocused, _ := diagnostics.Render("", nil, diags, 1, false)

	assert.NotEqual(t, focused, unfocused,
		"the selected row must look different depending on whether the pane drives")
	assert.Equal(t, testutils.StripANSI(focused), testutils.StripANSI(unfocused),
		"...but only in styling — the text is the same either way")
}

// Diagnostics-pane navigation: paging, wheel, minimal scrolling, and Enter jumping to the source.
func TestDiagNav(t *testing.T) {
	t.Run("paging", func(t *testing.T) {
		m, _ := diagNavModel(t)
		page := m.diag.VisibleRows()
		require.Positive(t, page)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
		assert.Equal(t, page, m.diagCursor, "page down moves a viewport's worth")

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgUp})
		assert.Zero(t, m.diagCursor)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnd})
		assert.Equal(t, diagNavCount-1, m.diagCursor, "end selects the last diagnostic")

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyHome})
		assert.Zero(t, m.diagCursor)
	})

	// The wheel must move the selection, not just the view — otherwise you can scroll away and then `f` or Enter acts on
	// a diagnostic you cannot see.
	t.Run("wheel moves the selection", func(t *testing.T) {
		m, _ := diagNavModel(t)

		for range 3 {
			_ = m.handleMouse(tea.MouseMsg{
				Button: tea.MouseButtonWheelDown,
				Action: tea.MouseActionPress,
				X:      1,
				Y:      headerH + m.topH, // inside the diagnostics strip
			})
		}

		assert.Equal(t, 3, m.diagCursor, "the wheel carried the selection with it")
	})

	// Stepping the selection must not shift the whole list under the reader.
	t.Run("scrolls minimally", func(t *testing.T) {
		m, _ := diagNavModel(t)
		page := m.diag.VisibleRows()

		// Moving inside the visible window leaves the viewport alone.
		before := m.diag.TopLine()
		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, before, m.diag.TopLine(), "the target was already on screen")

		// Walking past the bottom edge scrolls by exactly one line at a time.
		for range page + 2 {
			_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		}
		assert.Positive(t, m.diag.TopLine(), "it did scroll once the selection left the window")
		assert.Less(t, m.diag.TopLine(), m.diagCursor+1, "and no further than needed")
	})

	// Enter is the one-shot counterpart to `f`: it goes to the source AND moves focus, where follow mode keeps the
	// diagnostics pane driving.
	t.Run("enter jumps to source", func(t *testing.T) {
		m, src := diagNavModel(t)
		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		require.Equal(t, 2, m.diagCursor)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Equal(t, src, m.currentFile, "the producing file is open")
		assert.Equal(t, modeView, m.leftMode)
		assert.Equal(t, paneTree, m.focused, "focus moved to the source, unlike follow mode")
		assert.Equal(t, 2, m.fileView.CurrentLine(), "on the diagnostic's line (3, 0-based 2)")
		assert.Contains(t, m.notice, "many.go:3")
	})

	t.Run("enter without a position", func(t *testing.T) {
		m := testModel(t,
			diagSize(80, 10),
			focusedOn(paneDiag),
			withDiags(grammar.Warnf(token.Position{}, grammar.CodeInvalidNumber, "no position")),
		)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Empty(t, m.currentFile, "nothing was opened")
		assert.Contains(t, m.notice, "no position")
	})

	t.Run("enter with no diagnostics", func(t *testing.T) {
		m := testModel(t, diagSize(80, 10), focusedOn(paneDiag))

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Empty(t, m.currentFile)
		assert.Empty(t, m.notice)
	})

	// The pane still declines keys it does not own, so the globals keep working.
	t.Run("unowned keys fall through", func(t *testing.T) {
		m, _ := diagNavModel(t)
		m.scan.JSON = `{"swagger":"2.0"}`
		m.refreshSpec()

		_ = m.handleKey(testutils.KeyRune('h'))
		assert.True(t, m.help.IsOpen(), "h still reaches the global help")

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		_ = m.handleKey(testutils.KeyRune('r'))
		assert.True(t, m.scan.Running, "r still rescans")
	})
}

// TestDoScanCollectsDiagnostics is the end-to-end wiring proof: a malformed numeric validation surfaces the parser's
// CodeInvalidNumber through Options.OnDiagnostic into scan.ResultMsg.Diags, without failing the scan
// (diagnostics never abort the build).
func TestDoScanCollectsDiagnostics(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod":   "module diagfixture\n\ngo 1.25\n",
		"types.go": badMaximumFixture,
	})

	res := scan.Do(codescan.Options{
		WorkDir:    dir,
		Packages:   []string{"."},
		ScanModels: true,
	})

	if res.Err != nil {
		t.Fatalf("scan should not hard-fail on soft diagnostics: %v", res.Err)
	}
	if len(res.Diags) == 0 {
		t.Fatal("expected at least one diagnostic from the malformed fixture")
	}

	found := false
	for _, d := range res.Diags {
		if d.Code == grammar.CodeInvalidNumber {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a %s diagnostic; got %v", grammar.CodeInvalidNumber, codes(res.Diags))
	}
}

// The diagnostics pane was the last one where the view and the selection could drift apart: the wheel scrolled the
// viewport without moving the cursor, and every cursor step re-centred the whole list.
// These pin the corrected behaviour, matching the spec pane and the source viewer.

const diagNavCount = 30

// badMaximumFixture is the minimal diagnostic trigger: a swagger:model whose field carries a non-numeric `maximum:`.
//
// The parser drops the keyword from the spec and emits grammar.CodeInvalidNumber — exactly the soft-diagnostic shape
// the pane is built to surface.
// Kept inline so the test owns its input and the lean TUI module needs no root test-fixture dependency.
const badMaximumFixture = `package diagfixture

// BadMaximum has an invalid maximum: value.
//
// swagger:model BadMaximum
type BadMaximum struct {
	// Count holds an arbitrary count.
	//
	// maximum: notanumber
	Count int ` + "`json:\"count\"`" + `
}
`

// diagNavModel builds a model with many diagnostics over two files, so paging and file-switching both have somewhere to
// go.
func diagNavModel(t *testing.T) (*Model, string) {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(dir, "many.go")
	require.NoError(t, os.WriteFile(src, []byte(strings.Repeat("package p\n", diagNavCount+2)), 0o600))

	diags := make([]grammar.Diagnostic, 0, diagNavCount)
	for i := range diagNavCount {
		pos := token.Position{Filename: src, Line: i + 1, Column: 1}
		diags = append(diags, grammar.Warnf(pos, grammar.CodeInvalidNumber, "diag %d", i))
	}

	m := testModelIn(t, dir,
		panelSize(80, 10),
		diagSize(80, 10),
		focusedOn(paneDiag),
		withDiags(diags...),
	)

	return m, src
}

func codes(diags []grammar.Diagnostic) []grammar.Code {
	out := make([]grammar.Code, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Code)
	}
	return out
}

// writeModule materializes files (relative path → content) under a fresh temp dir and returns it. codescan scans it
// as a standalone module (it forces GOWORK=off), so no go.sum or workspace entry is needed for a stdlib-only tree.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

func pos(file string, line, col int) token.Position {
	return token.Position{Filename: file, Line: line, Column: col}
}
