// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// bufferTabWidth is how many spaces bubbles/textarea substitutes for a tab when a file is loaded into it.
//
// Not a preference of ours — a property of the widget that every file coordinate has to be translated through.
const bufferTabWidth = 4

// leftMode is what the left pane shows: the source tree or a file's content.
type leftMode int

const (
	modeBrowse leftMode = iota
	modeView
)

// loadFileQuietly loads path into the read-only viewer and switches the left pane to view mode WITHOUT changing focus
// — used by spec-driven follow, where the spec keeps focus while the source pane mirrors.
//
// A read error is shown in the buffer.
func (m *Model) loadFileQuietly(path string) {
	m.currentFile = path
	content, err := os.ReadFile(path)
	if err != nil {
		m.currentSource = ""
		m.fileView.SetFile(filepath.Base(path), "error reading file: "+err.Error())
		m.fileView.SetSpans(nil)
		m.fileView.SetAnchors(nil)
		m.leftMode = modeView

		return
	}

	m.currentSource = normalizeNewlines(string(content))
	m.fileView.SetFile(relTo(m.cfg.WorkDir, path), m.currentSource)
	m.refreshSource()
	m.leftMode = modeView
}

// normalizeNewlines converts CRLF and lone CR to LF.
//
// The editor widget treats a CR as a line break of its own, so a file with Windows endings loads as twice as many lines
// with a blank between each.
//
// Line numbers below the first CR are then wrong in the buffer while right in the file — and the line number is the
// coordinate the whole cross-reference layer is keyed on: provenance anchors, follow mode, go-to-definition, diagnostic
// marks.
// Normalising on the way in gives the file, the buffer, the indexes and the marks one shared notion of what a line is.
//
// Saving therefore writes LF.
// That is the same bargain the widget already imposes on tabs, and it is recorded with it in the module README.
func normalizeNewlines(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}

	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// refreshSource re-derives everything the source pane knows about the open file: its lexical runs, the diagnostics
// landing in it, and the provenance anchors in its gutter.
//
// It runs on load, on leaving the editor, and after every rescan.
// The last is what this function exists for — a rescan replaces both the anchors and the diagnostics, and before it
// was wired the open file kept showing the previous scan's marks while the pane below it listed the new ones.
func (m *Model) refreshSource() {
	if m.currentFile == "" {
		return
	}
	// Tokenize the BUFFER, not the bytes read: textarea rewrites tabs as spaces on the way in, so a span computed from the
	// file would sit three columns early per leading tab and cut a token in half.
	spans := goSpans(m.currentFile, []byte(m.fileView.Value()))
	m.fileView.SetSpans(index.MarkDiagnostics(spans, m.sourceMarks()))
	m.fileView.SetAnchors(m.srcIndex.AnchorLines(m.currentFile))
}

// bufferColumn converts a 1-based BYTE column in a file line — what go/token reports — into the 1-based RUNE column
// of the same character as displayed.
//
// Two conversions in one, and both are needed: multi-byte runes make a byte column drift from a rune column, and
// textarea substitutes four spaces for every tab, so leading indentation is wider on screen than in the file.
func bufferColumn(fileLine string, byteCol int) int {
	col := 1
	for i, r := range fileLine {
		if i >= byteCol-1 {
			break
		}
		if r == '\t' {
			col += bufferTabWidth

			continue
		}
		col++
	}

	return col
}

// diagKind maps a severity onto the class that paints it.
func diagKind(severity grammar.Severity) theme.SyntaxKind {
	switch severity {
	case grammar.SeverityError:
		return theme.SyntaxDiagError
	case grammar.SeverityWarning:
		return theme.SyntaxDiagWarn
	case grammar.SeverityHint:
		return theme.SyntaxDiagHint
	default:
		return theme.SyntaxDiagHint
	}
}

// goSpans returns the syntax runs for a Go source file, and nil for anything else — the tree happily opens go.mod, a
// golden JSON fixture or a README, and a Go tokenizer has nothing true to say about those.
func goSpans(path string, content []byte) map[int][]theme.Span {
	if filepath.Ext(path) != ".go" {
		return nil
	}

	return index.BuildGoHighlight(content).All()
}

// sourceMarks locates the last scan's diagnostics for the open file in the coordinates the pane draws in.
func (m *Model) sourceMarks() []index.DiagMark {
	if m.currentSource == "" {
		return nil
	}

	lines := strings.Split(m.currentSource, "\n")
	marks := make([]index.DiagMark, 0, len(m.scan.Diags))
	for _, d := range m.scan.Diags {
		if d.Pos.Filename != m.currentFile || d.Pos.Line < 1 || d.Pos.Line > len(lines) {
			continue
		}
		marks = append(marks, index.DiagMark{
			Line: d.Pos.Line - 1,
			Col:  bufferColumn(lines[d.Pos.Line-1], d.Pos.Column),
			Kind: diagKind(d.Severity),
		})
	}

	return marks
}

// openFile loads path into the read-only viewer and focuses it.
//
// The viewer is navigable immediately; `i` enters the editor.
func (m *Model) openFile(path string) tea.Cmd {
	m.loadFileQuietly(path)
	m.focused = paneTree
	return nil
}

// requestReload re-reads the open file from disk, asking first when that would throw away unsaved edits.
//
// The auto-reload this replaces fired on every watcher event and silently clobbered whatever was in the buffer. A
// manual reload is the useful half of that — the file changed underneath you, or you want your edits gone — and the
// guard is what makes it safe to offer.
func (m *Model) requestReload() tea.Cmd {
	if m.currentFile == "" {
		return m.notify("%s", noFileDesc)
	}
	if !m.fileView.Dirty() {
		return m.reloadFile()
	}

	m.pendingConfirm = confirmReload
	m.confirm.Ask("Discard unsaved edits to " + relTo(m.cfg.WorkDir, m.currentFile) + "?")

	return nil
}

// reloadFile re-reads the open file from disk, keeping the reader roughly where they were.
//
// The nav line is restored by NUMBER, which is the only anchor available: unlike a rescan — where the spec cursor is
// restored by node — nothing indexes a line of Go source to something stable across an external edit. So the line is
// honest about being approximate rather than pretending to track content.
//
// Reload always lands in the READ-ONLY viewer, even when it interrupted an edit: the buffer has just been replaced with
// something the user did not type, and dropping them back into the editor over it invites re-editing the very content
// they chose to discard.
func (m *Model) reloadFile() tea.Cmd {
	path := m.currentFile
	if path == "" {
		return m.notify("%s", noFileDesc)
	}
	line := m.fileView.CurrentLine()

	m.loadFileQuietly(path) // SetFile marks the buffer clean and leaves edit mode
	m.fileView.GotoLine(line)
	m.fileView.Blur() // the textarea is no longer the active mode, so it must stop capturing input

	return m.notify("reloaded %s", relTo(m.cfg.WorkDir, path))
}

// saveFile writes the editor buffer back to disk.
//
// The watcher then triggers a rescan, so the spec reflects the edit.
func (m *Model) saveFile() tea.Cmd {
	if m.currentFile == "" {
		return nil
	}
	if err := os.WriteFile(m.currentFile, []byte(m.fileView.Value()), 0o644); err != nil { //nolint:gosec // user's own source tree
		return m.notify("save failed: %s", err)
	}
	m.fileView.MarkClean()

	return m.notify("saved %s", relTo(m.cfg.WorkDir, m.currentFile))
}

// relTo renders path relative to base when possible, else the base name.
func relTo(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(path)
}
