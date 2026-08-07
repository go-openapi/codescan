// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for the source pane: opening a file, the Go syntax runs it paints, and the diagnostic marks laid over them.

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A missing or unreadable file leaves an error message in the buffer.
//
// The spans from the PREVIOUS file must not colour that message by their columns.
func TestGoSpans_ReadErrorClearsSpans(t *testing.T) {
	m := goViewerModel(t, testutils.WriteTempGo(t, annotatedGo))
	require.Contains(t, m.fileView.View(false, false), "swagger:model")

	m.loadFileQuietly(filepath.Join(t.TempDir(), "gone.go"))

	body := m.fileView.View(false, false)
	assert.Contains(t, testutils.StripANSI(body), "error reading file")
	assert.NotContains(t, body, runOpenerFor(t, theme.SyntaxKey),
		"no annotation run survives into the error message")
}

// Go syntax runs in the source pane, including what an edit does to them.
func TestGoSyntax(t *testing.T) {
	// The annotation is what the pane exists for, so it must be told apart from the prose beside it all the way through to
	// the rendered output.
	t.Run("annotation stands out from prose", func(t *testing.T) {
		m := goViewerModel(t, testutils.WriteTempGo(t, annotatedGo))

		view := m.fileView.View(false, false)

		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User",
			"the annotation is highlighted as the payload")
		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxComment)+"// A user of the system.",
			"the prose next to it is not")
		assert.Contains(t, testutils.StripANSI(view), "type User struct {", "the text is unchanged")
	})

	// Leaving the editor returns to the highlighted viewer.
	//
	// Spans computed when the file was LOADED colour by the old columns, which is a plausible lie.
	t.Run("rehighlights on leaving edit mode", func(t *testing.T) {
		path := testutils.WriteTempGo(t, annotatedGo)
		m := goViewerModel(t, path)
		m.currentFile = path
		require.NotNil(t, m.fileView.StartEdit())

		// Prepend a line: every run below it is now one line off.
		m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("//")})
		m.fileView.Update(tea.KeyMsg{Type: tea.KeyEnter})
		require.True(t, m.fileView.Dirty())

		_ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})

		require.False(t, m.fileView.Editing(), "esc returns to the viewer")
		view := m.fileView.View(false, false)
		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User",
			"the annotation is coloured on the line it MOVED to")
	})

	// An edit that makes the file un-parseable must not blank the pane: the tokenizer is error tolerant precisely so
	// highlighting survives typing.
	t.Run("survives an unparseable edit", func(t *testing.T) {
		path := testutils.WriteTempGo(t, annotatedGo)
		m := goViewerModel(t, path)
		m.currentFile = path
		require.NotNil(t, m.fileView.StartEdit())

		m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("func f( {")})

		_ = m.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})

		assert.Contains(t, m.fileView.View(false, false),
			runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User")
	})

	// textarea rewrites tabs as spaces on the way in.
	//
	// Tokenizing the FILE rather than the buffer put every run three columns early PER LEADING TAB, which is enough to cut
	// a token in half: `int64` drew as a plain `int` and a green `64`, and `struct` came out in two colours.
	t.Run("tab indented lines colour on token boundaries", func(t *testing.T) {
		path := testutils.WriteTempGo(t, "package p\n\ntype T struct {\n\tID int64 `json:\"id\"`\n}\n")
		m := goViewerModel(t, path)

		view := m.fileView.View(false, false)

		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxString)+"`json:\"id\"`",
			"the string run starts at the backtick")
		assert.NotContains(t, view, runOpenerFor(t, theme.SyntaxString)+"int64",
			"and not one tab-expansion earlier, inside int64")
	})

	// Two tabs displace twice as far, so nesting is where this first became visible.
	t.Run("nested tabs stay on token boundaries", func(t *testing.T) {
		path := testutils.WriteTempGo(t,
			"package p\n\ntype T struct {\n\tItems []struct {\n\t\tPetID int64 `json:\"petId\"`\n\t}\n}\n")
		m := goViewerModel(t, path)

		view := m.fileView.View(false, false)

		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxString)+"`json:\"petId\"`")
		assert.Contains(t, view, runOpenerFor(t, theme.SyntaxKeyword)+"struct ",
			"`struct` is one run, not split across two colours")
	})
}

// Highlighting must leave every line of every file exactly as it was.
//
// Checked against real fixture sources rather than a hand-written snippet.
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

		plain := testutils.StripANSI(m.fileView.View(false, false))
		for line := range strings.SplitSeq(string(content), "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				require.Contains(t, plain, trimmed, "%s: highlighting altered the text", e.Name())
			}
		}
		checked++
	}
	require.Positive(t, checked, "the fixture directory must hold Go sources")
}

// Windows line endings are normalised on the way in, so every coordinate layer agrees on what a line is.
func TestCRLF(t *testing.T) {
	// A file with Windows line endings must load with the SAME line numbering the file has.
	//
	// The editor widget treats a lone CR as a line break, so without normalising,
	// the buffer gains a blank line after every real one and every coordinate below the first CR - anchors, marks,
	// follow targets - points a growing distance away from what it names.
	//
	// This reproduces on any platform: the endings are in the fixture, not the checkout.
	t.Run("line numbering survives windows endings", func(t *testing.T) {
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
	})

	// ...and the highlighting keyed on those lines still lands: the annotation is on line 2 of the file, so it must be on
	// line 2 of the pane.
	t.Run("highlighting lands on the right line", func(t *testing.T) {
		crlf := strings.ReplaceAll(annotatedGo, "\n", "\r\n")
		path := filepath.Join(t.TempDir(), "crlf.go")
		require.NoError(t, os.WriteFile(path, []byte(crlf), 0o600))

		m := goViewerModel(t, path)

		assert.Contains(t, m.fileView.View(false, false),
			runOpenerFor(t, theme.SyntaxKey)+"// swagger:model User")
	})

	// A diagnostic names a line in the FILE; with CRLF unhandled it marked a line that had drifted away from the one it
	// named.
	t.Run("diagnostics mark the line they name", func(t *testing.T) {
		crlf := strings.ReplaceAll(annotatedGo, "\n", "\r\n")
		path := filepath.Join(t.TempDir(), "crlf.go")
		require.NoError(t, os.WriteFile(path, []byte(crlf), 0o600))

		m := goViewerModel(t, path)
		m.currentFile = path
		// Line 6 is the `Name string` field, indented with a tab.
		m.scan.Diags = []grammar.Diagnostic{{
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
	})
}

// A file column and a displayed column are two conversions apart.
//
// go/token counts BYTES, and textarea substitutes four spaces per tab.
//
// Getting this wrong is how int64 once drew as int plus a green 64.
func TestBufferColumn(t *testing.T) {
	for _, tc := range []struct {
		name    string
		line    string
		byteCol int
		want    int
	}{
		{"first column is untouched", "package p", 1, 1},
		{"no tabs, no multi-byte: identity", "package p", 9, 9},
		{"one leading tab is four columns", "\t// in: formData", 5, 8},
		{"two leading tabs are eight", "\t\t// in: formData", 6, 12},
		{"a tab mid-line counts too", "a\tb", 3, 6},
		{"multi-byte runes are one column each", "// héllo x", 11, 10},
		{"past the end clamps to the line", "ab", 99, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, bufferColumn(tc.line, tc.byteCol))
		})
	}
}

func TestDiagKind_CoversEverySeverity(t *testing.T) {
	assert.Equal(t, theme.SyntaxDiagError, diagKind(grammar.SeverityError))
	assert.Equal(t, theme.SyntaxDiagWarn, diagKind(grammar.SeverityWarning))
	assert.Equal(t, theme.SyntaxDiagHint, diagKind(grammar.SeverityHint))
}

// classificationOnce caches the malformed-input corpus scan for the package.
//
// The same reason petstoreOnce caches the petstore one: a real packages.Load costs seconds under -race,
// and three of these tests need the same scan.
var (
	classificationOnce sync.Once      //nolint:gochecknoglobals // test-only scan cache
	classificationRes  scan.ResultMsg //nolint:gochecknoglobals // test-only scan cache
)

// End to end against a real scan.
//
// The diagnostic the pane below lists must be drawn on the token it names, in the coordinates the pane draws in.
func TestE2E_DiagnosticMarksTheOffendingKeyword(t *testing.T) {
	m, path := classificationModel(t, filepath.Join("operations", "noparams.go"))

	marks := m.sourceMarks()
	require.NotEmpty(t, marks, "this fixture is the malformed-input corpus")

	buffer := strings.Split(m.fileView.Value(), "\n")
	for _, mark := range marks {
		require.Less(t, mark.Line, len(buffer))
		runes := []rune(buffer[mark.Line])
		require.LessOrEqual(t, mark.Col, len(runes)+1,
			"line %d: a mark past the end of the line it is on", mark.Line+1)
	}

	// `// maximum: 3` is a schema keyword under a prose-only classifier body, so it is reported at the keyword; after
	// translation the mark must still be the keyword, not the tab that precedes it.
	//
	// The subject used to be the `// in: formData` on the same field, which warned beside `swagger:file`.
	// That warning was spurious - `in:` is a field directive the parameters builder reads out of band, and the line it
	// fired on is the canonical file-upload idiom - so the subject moved to a keyword that is genuinely invalid there.
	// The tab-indent property under test is unchanged.
	source := strings.Split(m.currentSource, "\n")
	var checked int
	for _, d := range m.scan.Diags {
		if d.Pos.Filename != path || !strings.Contains(source[d.Pos.Line-1], "// maximum:") {
			continue
		}
		col := bufferColumn(source[d.Pos.Line-1], d.Pos.Column)
		assert.True(t, strings.HasPrefix(string([]rune(buffer[d.Pos.Line-1])[col-1:]), "maximum:"),
			"line %d landed on %q", d.Pos.Line, buffer[d.Pos.Line-1])
		checked++
	}
	require.Positive(t, checked, "the fixture must still contain a context-invalid keyword")
}

// The mark has to survive all the way to the screen, over the lexical class the token would otherwise have had.
func TestE2E_DiagnosticStyleReachesTheRenderedPane(t *testing.T) {
	m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))

	marks := m.sourceMarks()
	require.NotEmpty(t, marks)
	m.fileView.GotoLine(marks[0].Line)

	view := m.fileView.View(false, false)

	assert.Contains(t, view, runOpenerFor(t, marks[0].Kind),
		"the diagnostic's style is drawn in the source pane")
}

// Diagnostic marks over the open file, and what a rescan does to them.
func TestDiagMarks(t *testing.T) {
	// Diagnostics name their file.
	//
	// Marking a line number from another file would underline whatever happens to be there.
	t.Run("other files do not mark this one", func(t *testing.T) {
		m, path := classificationModel(t, filepath.Join("operations", "noparams.go"))
		require.NotEmpty(t, m.sourceMarks())

		for i := range m.scan.Diags {
			m.scan.Diags[i].Pos.Filename = path + ".elsewhere"
		}

		assert.Empty(t, m.sourceMarks(), "another file's diagnostics stay in it")
	})

	// A rescan replaces the diagnostics; before this was wired the open file went on showing the previous scan's marks
	// while the pane below listed the new ones.
	t.Run("rescan refreshes the open file", func(t *testing.T) {
		path := testutils.WriteTempGo(t, annotatedGo)
		m := goViewerModel(t, path)
		m.currentFile = path
		require.Empty(t, m.sourceMarks(), "a clean file starts unmarked")

		// Line 3 is `// A user of the system.`, prose - so an unmarked comment run.
		before := m.fileView.View(false, false)
		require.NotContains(t, before, runOpenerFor(t, theme.SyntaxDiagError))

		m.scan.Diags = []grammar.Diagnostic{{
			Pos:      token.Position{Filename: path, Line: 4, Column: 1},
			Severity: grammar.SeverityError,
			Code:     grammar.CodeUnexpectedToken,
			Message:  "invented for this test",
		}}
		m.refreshSource()

		assert.Contains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError),
			"the new scan's marks are on screen")
	})

	// ...and through the loop that actually delivers a rescan.
	//
	// Calling refreshSource directly proves the function works; only this proves it is WIRED, which is the half that was
	// missing.
	t.Run("rescan through the update loop", func(t *testing.T) {
		path := testutils.WriteTempGo(t, annotatedGo)
		m := goViewerModel(t, path)
		m.currentFile = path
		m.spec.SetSize(80, 20)
		require.NotContains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError))

		_, _ = m.Update(scan.ResultMsg{
			JSON: "{}",
			Diags: []grammar.Diagnostic{{
				Pos:      token.Position{Filename: path, Line: 4, Column: 1},
				Severity: grammar.SeverityError,
				Code:     grammar.CodeUnexpectedToken,
				Message:  "invented for this test",
			}},
		})

		assert.Contains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError),
			"a rescan must re-derive the open file, not only the panes below it")
	})

	// Opening a different file must not carry the previous one's marks across.
	t.Run("cleared when the file changes", func(t *testing.T) {
		m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))
		require.NotEmpty(t, m.sourceMarks())

		clean := testutils.WriteTempGo(t, annotatedGo)
		m.loadFileQuietly(clean)

		assert.Empty(t, m.sourceMarks())
		assert.NotContains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError))
	})

	// A read failure leaves an error message in the buffer; marks resolved against the file that is no longer loaded would
	// colour that message by its columns.
	t.Run("read error drops them", func(t *testing.T) {
		m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))
		require.NotEmpty(t, m.sourceMarks())

		m.loadFileQuietly(filepath.Join(t.TempDir(), "gone.go"))

		assert.Empty(t, m.sourceMarks())
		assert.Contains(t, testutils.StripANSI(m.fileView.View(false, false)), "error reading file")
	})
}

// No mark may land past the end of the line it is on, at any indentation.
//
// Checked over every Go file the malformed-input corpus scans.
func TestE2E_NoMarkLandsPastItsLine(t *testing.T) {
	dir := fixturesDir(t)
	res := classificationScan(t)

	files := map[string]bool{}
	for _, d := range res.Diags {
		files[d.Pos.Filename] = true
	}

	var checked int
	for path := range files {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		m := testModelIn(t, dir, panelSize(120, 30), withDiags(res.Diags...), openFile(path))

		buffer := strings.Split(m.fileView.Value(), "\n")
		for _, mark := range m.sourceMarks() {
			require.Less(t, mark.Line, len(buffer), path)
			require.LessOrEqual(t, mark.Col, len([]rune(buffer[mark.Line]))+1,
				"%s:%d: mark at column %d, line is %q",
				filepath.Base(path), mark.Line+1, mark.Col, buffer[mark.Line])
			checked++
		}
	}
	require.Positive(t, checked, "the corpus must produce marks to check")
}

// classificationModel opens one of the corpus's files against that scan.
func classificationModel(t *testing.T, name string) (*Model, string) {
	t.Helper()

	dir := fixturesDir(t)
	res := classificationScan(t)

	path := filepath.Join(dir, "goparsing", "classification", name)
	m := testModelIn(t, dir,
		panelSize(100, 30),
		withDiags(res.Diags...),
		openFile(path),
	)
	m.scan.JSON = res.JSON

	return m, path
}

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
	return testModel(t, panelSize(60, 20), openFile(path))
}

// The tree opens whatever the user points at.
//
// A Go tokenizer has nothing true to say about go.mod or a golden JSON fixture, so those stay plain.
func TestGoSpans_OnlyGoFiles(t *testing.T) {
	assert.NotEmpty(t, goSpans("x.go", []byte(annotatedGo)))

	for _, name := range []string{"go.mod", "golden.json", "README.md", "Makefile"} {
		assert.Nil(t, goSpans(name, []byte(annotatedGo)), name)
	}
}

// runOpenerFor is the SGR prefix a syntax class renders with.
//
// Assertions match on it because a run also carries whatever follows it, up to the next run.
func runOpenerFor(t *testing.T, kind theme.SyntaxKind) string {
	t.Helper()
	rendered := theme.Syntax(kind).Render("x")
	prefix, _, found := strings.Cut(rendered, "x")
	require.True(t, found)
	require.NotEmpty(t, prefix, "colour is off — see TestMain")

	return prefix
}

// classificationScan is the cached scan of the fixture corpus that deliberately contains malformed input.
//
// Diagnostics are CLONED per caller: one of these tests rewrites their filenames, and the cache is shared.
func classificationScan(t *testing.T) scan.ResultMsg {
	t.Helper()

	classificationOnce.Do(func() {
		classificationRes = scan.Do(codescan.Options{
			WorkDir:  fixturesDir(t),
			Packages: []string{"./goparsing/classification/..."},
		})
	})
	require.NoError(t, classificationRes.Err)
	require.NotEmpty(t, classificationRes.Diags, "the corpus must produce diagnostics")

	res := classificationRes
	res.Diags = slices.Clone(classificationRes.Diags)

	return res
}
