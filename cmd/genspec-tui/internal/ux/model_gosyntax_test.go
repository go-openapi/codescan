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
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const annotatedGo = `package main

// swagger:model User
// A user of the system.
type User struct {
	Name string ` + "`json:\"name\"`" + `
}
`

// goViewerModel opens path in a model whose source pane is sized to render.
func goViewerModel(t *testing.T, path string) *Model {
	t.Helper()
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.fileView.SetSize(60, 20)
	m.loadFileQuietly(path)

	return m
}

// The tree opens whatever the user points at. A Go tokenizer has nothing true
// to say about go.mod or a golden JSON fixture, so those stay plain.
func TestGoSpans_OnlyGoFiles(t *testing.T) {
	assert.NotEmpty(t, goSpans("x.go", []byte(annotatedGo)))

	for _, name := range []string{"go.mod", "golden.json", "README.md", "Makefile"} {
		assert.Nil(t, goSpans(name, []byte(annotatedGo)), name)
	}
}

// A missing or unreadable file leaves an error message in the buffer, and the
// spans from the PREVIOUS file must not colour it by their columns.
func TestGoSpans_ReadErrorClearsSpans(t *testing.T) {
	m := goViewerModel(t, writeTempGo(t, annotatedGo))
	require.Contains(t, m.fileView.View(false, false), "swagger:model")

	m.loadFileQuietly(filepath.Join(t.TempDir(), "gone.go"))

	body := m.fileView.View(false, false)
	assert.Contains(t, stripANSI(body), "error reading file")
	assert.NotContains(t, body, runOpenerFor(t, theme.SyntaxKey),
		"no annotation run survives into the error message")
}

// The annotation is what the pane exists for, so it must be told apart from the
// prose beside it all the way through to the rendered output.
func TestGoSyntax_AnnotationStandsOutFromProse(t *testing.T) {
	m := goViewerModel(t, writeTempGo(t, annotatedGo))

	view := m.fileView.View(false, false)

	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User",
		"the annotation is highlighted as the payload")
	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxComment)+"// A user of the system.",
		"the prose next to it is not")
	assert.Contains(t, stripANSI(view), "type User struct {", "the text is unchanged")
}

// Leaving the editor returns to the highlighted viewer. Spans computed when the
// file was LOADED colour by the old columns, which is a plausible lie.
func TestGoSyntax_RehighlightsOnLeavingEditMode(t *testing.T) {
	path := writeTempGo(t, annotatedGo)
	m := goViewerModel(t, path)
	m.currentFile = path
	require.NotNil(t, m.fileView.StartEdit())

	// Prepend a line: every run below it is now one line off.
	m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("//")})
	m.fileView.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, m.fileView.Dirty())

	_, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})

	require.False(t, m.fileView.Editing(), "esc returns to the viewer")
	view := m.fileView.View(false, false)
	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User",
		"the annotation is coloured on the line it MOVED to")
}

// An edit that makes the file un-parseable must not blank the pane: the
// tokenizer is error tolerant precisely so highlighting survives typing.
func TestGoSyntax_SurvivesAnUnparseableEdit(t *testing.T) {
	path := writeTempGo(t, annotatedGo)
	m := goViewerModel(t, path)
	m.currentFile = path
	require.NotNil(t, m.fileView.StartEdit())

	m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("func f( {")})

	_, _ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.Contains(t, m.fileView.View(false, false),
		runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User")
}

// textarea rewrites tabs as spaces on the way in. Tokenizing the FILE rather
// than the buffer put every run three columns early PER LEADING TAB, which is
// enough to cut a token in half: `int64` drew as a plain `int` and a green `64`,
// and `struct` came out in two colours.
func TestGoSyntax_TabIndentedLinesColourOnTokenBoundaries(t *testing.T) {
	path := writeTempGo(t, "package p\n\ntype T struct {\n\tID int64 `json:\"id\"`\n}\n")
	m := goViewerModel(t, path)

	view := m.fileView.View(false, false)

	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxString)+"`json:\"id\"`",
		"the string run starts at the backtick")
	assert.NotContains(t, view, runOpenerFor(t, theme.SyntaxString)+"int64",
		"and not one tab-expansion earlier, inside int64")
}

// Two tabs displace twice as far, so nesting is where this first became visible.
func TestGoSyntax_NestedTabsStayOnTokenBoundaries(t *testing.T) {
	path := writeTempGo(t,
		"package p\n\ntype T struct {\n\tItems []struct {\n\t\tPetID int64 `json:\"petId\"`\n\t}\n}\n")
	m := goViewerModel(t, path)

	view := m.fileView.View(false, false)

	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxString)+"`json:\"petId\"`")
	assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKeyword)+"struct ",
		"`struct` is one run, not split across two colours")
}

// Against real fixture sources rather than a hand-written snippet: highlighting
// must leave every line of every file exactly as it was.
func TestE2E_GoSyntaxLeavesTheSourceIntact(t *testing.T) {
	dir := filepath.Join(fixturesDir(t), "goparsing", "petstore", "models")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var checked int
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		m := goViewerModel(t, path)
		m.fileView.SetSize(400, len(strings.Split(string(content), "\n"))+4)

		plain := stripANSI(m.fileView.View(false, false))
		for line := range strings.SplitSeq(string(content), "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				require.Contains(t, plain, trimmed, "%s: highlighting altered the text", e.Name())
			}
		}
		checked++
	}
	require.Positive(t, checked, "the fixture directory must hold Go sources")
}

// runOpenerFor is the SGR prefix a syntax class renders with — assertions match
// on it because a run also carries whatever follows it up to the next run.
func runOpenerFor(t *testing.T, kind theme.SyntaxKind) string {
	t.Helper()
	rendered := theme.Syntax(kind).Render("x")
	prefix, _, found := strings.Cut(rendered, "x")
	require.True(t, found)
	require.NotEmpty(t, prefix, "colour is off — see TestMain")

	return prefix
}

// A file with Windows line endings must load with the SAME line numbering the
// file has. The editor widget treats a lone CR as a line break, so without
// normalising, the buffer gains a blank line after every real one and every
// coordinate below the first CR — anchors, marks, follow targets — points a
// growing distance away from what it names.
//
// This reproduces on any platform: the endings are in the fixture, not the
// checkout.
func TestCRLF_LineNumberingSurvivesWindowsEndings(t *testing.T) {
	crlf := strings.ReplaceAll(annotatedGo, "\n", "\r\n")
	path := filepath.Join(t.TempDir(), "crlf.go")
	require.NoError(t, os.WriteFile(path, []byte(crlf), 0o600))

	m := goViewerModel(t, path)

	// Modulo the widget's tab expansion, which is separate and documented.
	wantLines := strings.Split(strings.ReplaceAll(annotatedGo, "\t", "    "), "\n")
	gotLines := strings.Split(m.fileView.Value(), "\n")
	require.Equal(t, len(wantLines), len(gotLines), "the buffer gained or lost lines")
	assert.Equal(t, wantLines, gotLines, "line for line, the buffer is the file")

	assert.NotContains(t, m.fileView.Value(), "\r", "no carriage return survives into the buffer")
	assert.NotContains(t, m.currentSource, "\r", "nor into the coordinates diagnostics resolve against")
}

// ...and the highlighting keyed on those lines still lands: the annotation is
// on line 2 of the file, so it must be on line 2 of the pane.
func TestCRLF_HighlightingLandsOnTheRightLine(t *testing.T) {
	crlf := strings.ReplaceAll(annotatedGo, "\n", "\r\n")
	path := filepath.Join(t.TempDir(), "crlf.go")
	require.NoError(t, os.WriteFile(path, []byte(crlf), 0o600))

	m := goViewerModel(t, path)

	assert.Contains(t, m.fileView.View(false, false),
		runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User")
}

// A diagnostic names a line in the FILE; with CRLF unhandled it marked a line
// that had drifted away from the one it named.
func TestCRLF_DiagnosticsMarkTheLineTheyName(t *testing.T) {
	crlf := strings.ReplaceAll(annotatedGo, "\n", "\r\n")
	path := filepath.Join(t.TempDir(), "crlf.go")
	require.NoError(t, os.WriteFile(path, []byte(crlf), 0o600))

	m := goViewerModel(t, path)
	m.currentFile = path
	// Line 6 is the `Name string` field, indented with a tab.
	m.diags = []grammar.Diagnostic{{
		Pos:      token.Position{Filename: path, Line: 6, Column: 2},
		Severity: grammar.SeverityWarning,
		Code:     grammar.CodeContextInvalid,
		Message:  "invented for this test",
	}}
	m.refreshSource()

	marks := m.sourceMarks()
	require.Len(t, marks, 1)
	assert.Equal(t, 5, marks[0].Line, "0-based line 5 is the file's line 6")

	buffer := strings.Split(m.fileView.Value(), "\n")
	require.Less(t, marks[0].Line, len(buffer))
	assert.Contains(t, buffer[marks[0].Line], "Name string", "the mark is on the line it names")
}
