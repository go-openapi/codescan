// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// go/token counts BYTES and textarea substitutes four spaces per tab, so a file
// column and a displayed column are two conversions apart. Getting this wrong is
// how `int64` once drew as `int` plus a green `64`.
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

// classificationOnce caches the malformed-input corpus scan for the package, for
// the same reason petstoreOnce caches the petstore one: a real packages.Load
// costs seconds under -race, and three of these tests need the same scan.
var (
	classificationOnce sync.Once     //nolint:gochecknoglobals // test-only scan cache
	classificationRes  scanResultMsg //nolint:gochecknoglobals // test-only scan cache
)

// classificationScan is the cached scan of the fixture corpus that deliberately
// contains malformed input. Diagnostics are CLONED per caller: one of these
// tests rewrites their filenames, and the cache is shared.
func classificationScan(t *testing.T) scanResultMsg {
	t.Helper()

	classificationOnce.Do(func() {
		classificationRes = doScan(codescan.Options{
			WorkDir:  fixturesDir(t),
			Packages: []string{"./goparsing/classification/..."},
		})
	})
	require.NoError(t, classificationRes.err)
	require.NotEmpty(t, classificationRes.diags, "the corpus must produce diagnostics")

	res := classificationRes
	res.diags = slices.Clone(classificationRes.diags)

	return res
}

// classificationModel opens one of the corpus's files against that scan.
func classificationModel(t *testing.T, name string) (*Model, string) {
	t.Helper()

	dir := fixturesDir(t)
	res := classificationScan(t)

	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.cfg.WorkDir = dir
	m.spec.SetSize(100, 30)
	m.fileView.SetSize(90, 30)
	m.diags = res.diags
	m.specJSON = res.json

	path := filepath.Join(dir, "goparsing", "classification", name)
	m.loadFileQuietly(path)

	return m, path
}

// End to end against a real scan: the diagnostic the pane below lists must be
// drawn on the token it names, in the coordinates the pane draws in.
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

	// `// in: formData` is reported at the keyword; after translation it must
	// still be the keyword, not the tab that precedes it.
	source := strings.Split(m.currentSource, "\n")
	var checked int
	for _, d := range m.diags {
		if d.Pos.Filename != path || !strings.Contains(source[d.Pos.Line-1], "// in:") {
			continue
		}
		col := bufferColumn(source[d.Pos.Line-1], d.Pos.Column)
		assert.True(t, strings.HasPrefix(string([]rune(buffer[d.Pos.Line-1])[col-1:]), "in:"),
			"line %d landed on %q", d.Pos.Line, buffer[d.Pos.Line-1])
		checked++
	}
	require.Positive(t, checked, "the fixture must still contain a context-invalid `in:`")
}

// The mark has to survive all the way to the screen, over the lexical class the
// token would otherwise have had.
func TestE2E_DiagnosticStyleReachesTheRenderedPane(t *testing.T) {
	m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))

	marks := m.sourceMarks()
	require.NotEmpty(t, marks)
	m.fileView.GotoLine(marks[0].Line)

	view := m.fileView.View(false, false)

	assert.Contains(t, view, runOpenerFor(t, marks[0].Kind),
		"the diagnostic's style is drawn in the source pane")
}

// Diagnostics name their file. Marking a line number from another file would
// underline whatever happens to be there.
func TestDiagMarks_OtherFilesDoNotMarkThisOne(t *testing.T) {
	m, path := classificationModel(t, filepath.Join("operations", "noparams.go"))
	require.NotEmpty(t, m.sourceMarks())

	for i := range m.diags {
		m.diags[i].Pos.Filename = path + ".elsewhere"
	}

	assert.Empty(t, m.sourceMarks(), "another file's diagnostics stay in it")
}

// A rescan replaces the diagnostics; before this was wired the open file went on
// showing the previous scan's marks while the pane below listed the new ones.
func TestDiagMarks_RescanRefreshesTheOpenFile(t *testing.T) {
	path := writeTempGo(t, annotatedGo)
	m := goViewerModel(t, path)
	m.currentFile = path
	require.Empty(t, m.sourceMarks(), "a clean file starts unmarked")

	// Line 3 is `// A user of the system.`, prose — so an unmarked comment run.
	before := m.fileView.View(false, false)
	require.NotContains(t, before, runOpenerFor(t, theme.SyntaxDiagError))

	m.diags = []grammar.Diagnostic{{
		Pos:      token.Position{Filename: path, Line: 4, Column: 1},
		Severity: grammar.SeverityError,
		Code:     grammar.CodeUnexpectedToken,
		Message:  "invented for this test",
	}}
	m.refreshSource()

	assert.Contains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError),
		"the new scan's marks are on screen")
}

// ...and through the loop that actually delivers a rescan. Calling refreshSource
// directly proves the function works; only this proves it is WIRED, which is the
// half that was missing.
func TestDiagMarks_RescanThroughTheUpdateLoop(t *testing.T) {
	path := writeTempGo(t, annotatedGo)
	m := goViewerModel(t, path)
	m.currentFile = path
	m.spec.SetSize(80, 20)
	require.NotContains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError))

	_, _ = m.Update(scanResultMsg{
		json: "{}",
		diags: []grammar.Diagnostic{{
			Pos:      token.Position{Filename: path, Line: 4, Column: 1},
			Severity: grammar.SeverityError,
			Code:     grammar.CodeUnexpectedToken,
			Message:  "invented for this test",
		}},
	})

	assert.Contains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError),
		"a rescan must re-derive the open file, not only the panes below it")
}

// Opening a different file must not carry the previous one's marks across.
func TestDiagMarks_ClearedWhenTheFileChanges(t *testing.T) {
	m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))
	require.NotEmpty(t, m.sourceMarks())

	clean := writeTempGo(t, annotatedGo)
	m.loadFileQuietly(clean)

	assert.Empty(t, m.sourceMarks())
	assert.NotContains(t, m.fileView.View(false, false), runOpenerFor(t, theme.SyntaxDiagError))
}

// A read failure leaves an error message in the buffer; marks resolved against
// the file that is no longer loaded would colour that message by its columns.
func TestDiagMarks_ReadErrorDropsThem(t *testing.T) {
	m, _ := classificationModel(t, filepath.Join("operations", "noparams.go"))
	require.NotEmpty(t, m.sourceMarks())

	m.loadFileQuietly(filepath.Join(t.TempDir(), "gone.go"))

	assert.Empty(t, m.sourceMarks())
	assert.Contains(t, stripANSI(m.fileView.View(false, false)), "error reading file")
}

// Over every Go file the malformed-input corpus scans: no mark may land past the
// end of the line it is on, at any indentation.
func TestE2E_NoMarkLandsPastItsLine(t *testing.T) {
	dir := fixturesDir(t)
	res := classificationScan(t)

	files := map[string]bool{}
	for _, d := range res.diags {
		files[d.Pos.Filename] = true
	}

	var checked int
	for path := range files {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		m.cfg.WorkDir = dir
		m.fileView.SetSize(120, 30)
		m.diags = res.diags
		m.loadFileQuietly(path)

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
