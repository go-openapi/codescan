// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/diagnostics"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/validation"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/watcher"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// ScanState is what the last scan produced, plus whether another is in flight.
//
// It has exactly two writers - startScan, when one begins, and absorbScan, when its result lands. Everything else
// only reads it: the header and status line are projections of it, and the cross-ref layer renders from its JSON or
// YAML. Keeping it in one struct is what makes that one-way traffic visible.
type ScanState struct {
	Running bool
	Spin    spinner.Model
	Elapsed time.Duration

	NumPaths int
	NumDefs  int

	JSON string
	YAML string

	Diags []grammar.Diagnostic
	Err   error // hard error from the last codescan.Run, shown in the diag pane
}

// NewScanState builds the idle state, with the spinner the header shows while a scan runs.
func NewScanState() ScanState {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return ScanState{Spin: sp}
}

// Body is the rendered spec in the requested format.
func (s *ScanState) Body(yamlFmt bool) string {
	if yamlFmt {
		return s.YAML
	}

	return s.JSON
}

// SourceWatch is the live-reload plumbing.
//
// It holds the file watcher, the channel its events arrive on, and the debounce generation
// that lets a newer change discard a timer already in flight.
type SourceWatch struct {
	w   *watcher.Watcher
	ch  <-chan struct{}
	gen int
}

// NewSourceWatch starts a watcher on dir, best-effort.
//
// If it cannot initialize, live reload is simply unavailable and the user falls back to r (manual rescan) - so the
// error is deliberately swallowed rather than failing the whole TUI.
func NewSourceWatch(dir string) SourceWatch {
	w, err := watcher.New(dir)
	if err != nil {
		return SourceWatch{}
	}

	return SourceWatch{w: w, ch: w.Events()}
}

// Listening reports whether a watcher came up and events can be waited on.
func (s *SourceWatch) Listening() bool { return s.ch != nil }

// Events is the channel a change arrives on.
func (s *SourceWatch) Events() <-chan struct{} { return s.ch }

// Bump opens a new debounce window, invalidating any timer already running.
func (s *SourceWatch) Bump() int {
	s.gen++

	return s.gen
}

// Current reports whether gen is still the newest debounce window - false means a later change superseded it.
func (s *SourceWatch) Current(gen int) bool { return gen == s.gen }

// Close releases the watcher.
func (s *SourceWatch) Close() {
	if s.w != nil {
		_ = s.w.Close()
	}
}

// noticeTTL is how long a transient status notice stays on the status line before it clears.
//
// A notice is something like "copied to clipboard".
const noticeTTL = 2 * time.Second

// debounceDelay coalesces a burst of file-change events (an editor save often fires several) into a single rescan.
const debounceDelay = 300 * time.Millisecond

// copyResultMsg is delivered after an async clipboard copy completes.
type copyResultMsg struct{ err error }

// clearNoticeMsg clears the transient status notice.
type clearNoticeMsg struct{}

// fsEventMsg signals that the watcher saw a relevant source change.
type fsEventMsg struct{}

// debounceMsg fires after the quiet period; gen guards against stale timers.
type debounceMsg struct{ gen int }

// startScan marks a scan in flight and returns the scan command.
//
// The spinner starts only when one is not already running, which avoids stacking tick loops.
func (m *Model) startScan() tea.Cmd {
	already := m.scan.Running
	m.scan.Running = true
	run := scan.Run(m.cfg)
	if already {
		return run
	}

	return tea.Batch(run, m.scan.Spin.Tick)
}

// absorbScan takes in a finished scan.
//
// It replaces everything the previous one produced, re-derives the source index from the new provenance,
// and re-renders the three views that show any of it.
//
// The diagnostics cursor is clamped rather than reset - a scan that fixed one diagnostic should leave you next to the
// ones it did not.
func (m *Model) absorbScan(msg scan.ResultMsg) {
	m.scan.Running = false
	m.scan.JSON, m.scan.YAML = msg.JSON, msg.YAML
	m.scan.NumPaths, m.scan.NumDefs = msg.Paths, msg.Defs
	m.scan.Elapsed = msg.Elapsed
	m.scan.Diags = msg.Diags
	m.scan.Err = msg.Err

	m.diagCursor = min(max(m.diagCursor, 0), max(len(m.scan.Diags)-1, 0))
	m.srcIndex = index.BuildSourceIndex(msg.Provenance)
	m.retireValidation() // the findings judged the document this scan has just replaced

	m.applyScan()
}

// tickSpinner advances the scan spinner, and only while a scan is actually running.
func (m *Model) tickSpinner(msg spinner.TickMsg) tea.Cmd {
	if !m.scan.Running {
		return nil
	}

	var cmd tea.Cmd
	m.scan.Spin, cmd = m.scan.Spin.Update(msg)

	return cmd
}

// applyScan updates the spec and diagnostics panes from the latest scan.
func (m *Model) applyScan() {
	m.refreshSpec()
	m.refreshDiagnostics()
	m.refreshSource()
}

// refreshDiagnostics re-renders the diagnostics pane from whichever tab is active.
//
// It draws from the stored findings and that tab's selection cursor, scrolling the selected row into view.
//
// The pane shows any hard error from codescan.Run first, then every grammar.Diagnostic the build emitted (colored by
// severity, paths relative to the work dir, the selected row highlighted); a clean scan with no diagnostics shows the
// empty state.
func (m *Model) refreshDiagnostics() {
	focused := m.focused == paneDiag

	var (
		content string
		line    int
	)
	if m.diagTab == tabValidation {
		content, line = validation.Render(
			m.validation.Findings, m.validation.Ran, m.validation.Err, m.validation.Cursor, focused)
	} else {
		content, line = diagnostics.Render(m.cfg.WorkDir, m.scan.Err, m.scan.Diags, m.diagCursor, focused)
	}

	m.diag.SetTitle(m.diagTabTitle())
	m.diag.SetContent(content)
	if line >= 0 {
		m.diag.RevealLine(line)
	}
}

// notify shows a transient status message and returns the command that clears it.
//
// The two halves belong together: a notice that is set without also being scheduled to expire stays on the status line
// until something else happens to overwrite it. Pairing them here makes that structural rather than remembered.
func (m *Model) notify(format string, args ...any) tea.Cmd {
	m.notice = fmt.Sprintf(format, args...)

	return clearNoticeAfter(noticeTTL)
}

// clearNoticeAfter returns a command that emits clearNoticeMsg after d.
func clearNoticeAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearNoticeMsg{} })
}

// waitForFS blocks on the watcher channel and emits one fsEventMsg per change.
//
// It is re-issued after each event to form the listen loop; a closed channel ends the loop quietly.
func waitForFS(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return nil
		}
		return fsEventMsg{}
	}
}

// debounceCmd emits a debounceMsg for gen after the quiet period.
func debounceCmd(gen int) tea.Cmd {
	return tea.Tick(debounceDelay, func(time.Time) tea.Msg { return debounceMsg{gen: gen} })
}
