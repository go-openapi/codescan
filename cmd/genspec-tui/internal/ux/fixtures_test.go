// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
)

// testModel assembles a Model for a test through one door, browsing a fresh temp dir.
//
// It goes through New rather than filling a struct literal, so a test can never be assembled differently from
// production - which is what a hand-rolled &Model{} does: it has to know which panels a code path will reach, and
// which of them are unusable when zero-valued (a zero textinput panics on Focus). That knowledge was spread across
// eighteen per-file fixtures, each rediscovering it.
//
// Options are applied in the order given, and add only what a test is actually about. Anything genuinely peculiar to
// one test stays in that test, where it is visible.
func testModel(t *testing.T, opts ...modelOpt) *Model {
	t.Helper()

	return testModelIn(t, t.TempDir(), opts...)
}

// testModelIn is testModel over an existing directory, for tests whose source files are already on disk.
func testModelIn(t *testing.T, dir string, opts ...modelOpt) *Model {
	t.Helper()

	m := New(Startup{Options: &codescan.Options{WorkDir: dir, Packages: []string{"./..."}}})
	t.Cleanup(m.Close)

	m.width, m.height = 80, 24
	m.ready = true
	m.recalcLayout()

	for _, opt := range opts {
		opt(t, m)
	}

	return m
}

// modelOpt installs one aspect of a test's starting state.
type modelOpt func(*testing.T, *Model)

// sized fits the terminal (and every panel with it) to w×h.
//
// Several tests turn on how much fits on screen - paging, clamping, whether the cursor row is inside the scrolled
// window - so the size is stated wherever it is load-bearing rather than inherited.
func sized(w, h int) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.width, m.height = w, h
		m.recalcLayout()
	}
}

// panelSize fits the spec and file panes to w×h directly, for tests that size the panes rather than the terminal.
func panelSize(w, h int) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.spec.SetSize(w, h)
		m.fileView.SetSize(w, h)
	}
}

// diagSize fits the diagnostics strip to w×h and records the height the pager reads.
func diagSize(w, h int) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.diag.SetSize(w, h)
		m.diagH = h
	}
}

// withRenders installs both renders of the spec and builds the indexes for the active one.
func withRenders(jsonBody, yamlBody string) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.scan.JSON, m.scan.YAML = jsonBody, yamlBody
		m.refreshSpec()
	}
}

// withSpecJSON installs a JSON render and builds the real indexes from the rendered bytes.
func withSpecJSON(body string) modelOpt { return withRenders(body, "") }

// withSpecContent installs a rendered body and a hand-written index over it.
//
// For tests that need a pointer at a KNOWN line without depending on how a real render lays the document out.
func withSpecContent(body string, line2ptr map[int]string) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.spec.SetContent(body)

		ptr2line := make(map[string]int, len(line2ptr))
		for line, ptr := range line2ptr {
			ptr2line[ptr] = line
		}
		m.specIndex = index.NewSpecIndex(line2ptr, ptr2line)
	}
}

// withProvenance installs the source index a scan would have emitted.
func withProvenance(provs ...scanner.Provenance) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.srcIndex = index.BuildSourceIndex(provs)
	}
}

// withDiags installs the diagnostics a scan would have emitted, and renders the pane from them.
func withDiags(diags ...grammar.Diagnostic) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.scan.Diags = diags
		m.refreshDiagnostics()
	}
}

// openFile loads a file into the read-only viewer, as opening it from the tree would.
func openFile(path string) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.loadFileQuietly(path)
	}
}

// viewing puts a name and body straight into the file pane, for tests whose "file" never has to exist on disk.
func viewing(name, body string) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.currentFile = name
		m.currentSource = body
		m.fileView.SetFile(name, body)
		m.leftMode = modeView
	}
}

// focusedOn gives a pane the keyboard.
func focusedOn(p pane) modelOpt {
	return func(_ *testing.T, m *Model) {
		m.focused = p
	}
}
