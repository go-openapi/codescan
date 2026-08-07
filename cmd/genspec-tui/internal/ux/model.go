// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/confirm"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/gadgets"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/help"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/options"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
)

// Model is the root bubbletea model.
type Model struct {
	cfg codescan.Options

	// Terminal geometry, and the regions recalcLayout carves out of it (kept for mouse hit-testing).
	width, height      int
	leftW, topH, diagH int
	ready              bool

	// Mode: which pane owns the keyboard, and what the left one is showing.
	//
	// Every cluster consults these to decide whether a key is even theirs, which is what makes them the model's own
	// state rather than any one file's.
	focused  pane
	leftMode leftMode

	// The transient status message, and the diagnostics selection the status line reports.
	notice     string
	diagCursor int

	// What the last scan produced, written only by startScan and absorbScan.
	scan ScanState

	// The three cross-reference indexes.
	//
	// The first two are rebuilt per RENDER (refreshSpec) and the third per SCAN, which is why they are not one struct:
	// a format toggle invalidates the first two and leaves the third standing.
	specIndex *index.SpecIndex   // rendered-line ↔ JSON-pointer map for the active format
	refIndex  *index.RefIndex    // $ref sites in the active render (find-references / go-to-definition)
	srcIndex  *index.SourceIndex // JSON-pointer ↔ Go source position (cross-ref linker)

	// The open file, as loaded and as the diagnostics coordinates see it.
	currentFile   string
	currentSource string

	// Cross-ref auto-follow: which pane drives, and the resolved target for the status badge.
	follow       followMode
	followTarget string

	// Find-references cycle state (F3 / shift+F3); refreshSpec drops it with the render it was computed against.
	refs RefCycle

	// Live reload: the file watcher and its debounce window.
	watch SourceWatch

	// The spec-pane search prompt.
	search SearchBox

	tree     panels.Tree
	fileView panels.FileView
	spec     panels.Spec
	diag     panels.Diagnostics

	// Modal overlays, in the precedence the overlays method states.
	help    help.Overlay
	options options.Overlay
	confirm confirm.Overlay

	// What an accepted confirmation will do. The overlay records the answer; naming the action is the model's job.
	pendingConfirm confirmAction
}

// confirmAction names what a pending confirmation carries out when the user accepts it.
type confirmAction int

const (
	confirmNothing confirmAction = iota
	confirmReload
)

type pane int

const (
	paneTree pane = iota
	paneSpec
	paneDiag
	paneCount
)

// New builds the root model around a ready-made scan config; the source tree browses cfg.WorkDir.
//
// Taking the whole Options rather than a handful of arguments means a new CLI flag needs no signature change here —
// and the boolean knobs the overlay drives are the same struct the caller filled in.
//
// A file watcher is started best-effort — if it can't initialize, live reload is simply unavailable and the user
// falls back to `r` (manual rescan).
func New(cfg codescan.Options) *Model {
	m := &Model{
		cfg:      cfg,
		focused:  paneTree,
		scan:     NewScanState(),
		search:   NewSearchBox(),
		watch:    NewSourceWatch(cfg.WorkDir),
		tree:     panels.NewTree(cfg.WorkDir),
		fileView: panels.NewFileView(),
		spec:     panels.NewSpec(),
		diag:     panels.NewDiagnostics(),
		help:     help.New(),
		confirm:  confirm.New(),
	}
	// Built after the struct exists: the options rows bind to the scan-config booleans by pointer, and those pointers
	// have to be into m.cfg (valid because m is heap-allocated) rather than into the caller's copy.
	m.options = options.New(&m.cfg)

	return m
}

// Close releases the file watcher.
//
// Call after the program exits.
func (m *Model) Close() { m.watch.Close() }

// Init implements tea.Model: kick off the initial whole-scope scan and, if a watcher is available, begin listening for
// source changes.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.startScan()}
	if m.watch.Listening() {
		cmds = append(cmds, waitForFS(m.watch.Events()))
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		m.syncFollowIfActive() // re-mirror the follower after a driver move
		return m, cmd

	case tea.MouseMsg:
		cmd := m.handleMouse(msg)
		m.syncFollowIfActive()
		return m, cmd

	case spinner.TickMsg:
		return m, m.tickSpinner(msg)

	case scan.ResultMsg:
		m.absorbScan(msg)
		m.syncFollowIfActive() // refresh the follower against the rebuilt spec

		return m, nil

	case fsEventMsg:
		// A change arrived: start (restart) the debounce window and keep listening for the next event.
		return m, tea.Batch(debounceCmd(m.watch.Bump()), waitForFS(m.watch.Events()))

	case debounceMsg:
		// Rescan only if no newer change arrived during the quiet period.
		if m.watch.Current(msg.gen) {
			return m, m.startScan()
		}
		return m, nil

	case copyResultMsg:
		if msg.err != nil {
			return m, m.notify("clipboard error: %s", msg.err)
		}

		return m, m.notify("copied to clipboard")

	case clearNoticeMsg:
		m.notice = ""
		return m, nil
	}

	return m, m.updateFocused(msg)
}

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}

	if o := m.activeOverlay(); o != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, o.View())
	}

	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.leftView(m.focused == paneTree),
		m.spec.View(m.focused == paneSpec),
	)

	return m.headerLine() + "\n" +
		top + "\n" +
		m.diag.View(m.focused == paneDiag) + "\n" +
		m.statusLine()
}

// copyFocused copies the focused panel's raw content to the clipboard, asynchronously (the clipboard tool exec must not
// block the event loop).
//
// Returns nil when the focused panel has nothing to copy.
func (m *Model) copyFocused() tea.Cmd {
	text := m.focusedContent()
	if text == "" {
		return nil
	}

	return func() tea.Msg {
		return copyResultMsg{err: gadgets.CopyToClipboard(context.Background(), text)}
	}
}

func (m *Model) focusedContent() string {
	switch m.focused {
	case paneTree:
		if m.leftMode == modeView {
			return m.fileView.Content()
		}
		return m.tree.Content()
	case paneSpec:
		return m.spec.Content()
	case paneDiag:
		return m.diag.Content()
	}
	return ""
}

// updateFocused forwards a message to the currently focused panel (the left pane is the tree or the file viewer
// depending on leftMode).
func (m *Model) updateFocused(msg tea.Msg) tea.Cmd {
	switch m.focused {
	case paneTree:
		if m.leftMode == modeView {
			return m.fileView.Update(msg)
		}
		return m.tree.Update(msg)
	case paneSpec:
		return m.spec.Update(msg)
	case paneDiag:
		return m.diag.Update(msg)
	}
	return nil
}

// recalcLayout distributes the terminal size: a header line, a top row with the source tree (1/3 width) beside the
// spec, a diagnostics strip, and a status line.
//
// The regions are stored for mouse hit-testing.
func (m *Model) recalcLayout() {
	m.diagH = max(m.height/4, 5)
	m.topH = max(m.height-headerH-statusH-m.diagH, 3)
	m.leftW = max(min(m.width/3, m.width), 1)
	rightW := max(m.width-m.leftW, 1)

	m.tree.SetSize(m.leftW, m.topH)
	m.fileView.SetSize(m.leftW, m.topH)
	m.spec.SetSize(rightW, m.topH)
	m.diag.SetSize(m.width, m.diagH)
	for _, o := range m.overlays() {
		o.SetSize(m.width, m.height)
	}
}
