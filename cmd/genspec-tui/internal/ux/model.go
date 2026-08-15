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
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/reference"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/runstats"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
)

// Model is the root bubbletea model.
type Model struct {
	cfg *codescan.Options

	// What each scan captures about itself, fixed at launch. Not a runtime toggle: the heap sampling rate is set once
	// for the process, so a run profiled under one rate cannot be compared with a run profiled under another.
	profiling *scan.Profiling

	// Where the starting options came from, for the opening notice: a session that behaves unlike the command line
	// says so would otherwise leave the reader to discover the file themselves.
	configPath string
	configSet  int

	// Terminal geometry, and the regions recalcLayout carves out of it (kept for mouse hit-testing).
	width, height      int
	leftW, topH, diagH int
	ready              bool

	// Where the two dividers sit, as a percentage of the terminal. Adjustable at runtime; see recalcLayout for why
	// these are proportions rather than cell counts.
	leftPct, diagPct int

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
	help      help.Overlay
	options   options.Overlay
	confirm   confirm.Overlay
	reference reference.Overlay
	runstats  runstats.Overlay

	// What an accepted confirmation will do. The overlay records the answer; naming the action is the model's job.
	pendingConfirm confirmAction

	// The diagnostics pane's two views: the scan's own findings, and the spec validation v produces on demand.
	diagTab    diagTab
	validation ValidationState
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

// The split geometry: where the dividers start, how far a keypress moves them, and how far they may travel.
//
// The bounds are what stop a divider being driven onto a pane's edge - a pane you cannot see is a pane you cannot
// drag back, since the keys that would restore it are advertised in a status line the collapsed pane no longer has
// room to explain. They are asymmetric because the panes are: the diagnostics strip is a list you glance at, so it
// earns less of the screen than the two it sits under.
const (
	defaultLeftPct = 33 // the left pane's share of the width
	defaultDiagPct = 25 // the diagnostics strip's share of the height

	splitStep = 5

	minLeftPct, maxLeftPct = 15, 85
	minDiagPct, maxDiagPct = 10, 60
)

// Absolute floors, applied under the percentages so a small terminal still renders.
//
// Each is the least a pane can occupy and still show a border plus something inside it.
const (
	minLeftW = 1
	minTopH  = 3
	minDiagH = 5
)

// Startup is everything the command settled before the UI existed: what to scan, what to observe about the scan, and
// where those answers came from.
//
// One struct rather than a growing argument list, because these arrive together and are decided together - the
// configuration file presets the flags, the flags fill the options, and the options overlay takes over from there. A
// caller that only wants a scan leaves the rest zero.
type Startup struct {
	// Options is what the first scan runs with. Everything in it is a live setting afterwards: the options overlay
	// writes to this very struct, so it decides the session's starting point, not its limits - and the caller hands it
	// over rather than keeping a hand on it.
	//
	// Required: a session with nothing to scan is not one.
	Options *codescan.Options

	// Profiling says what each scan captures about itself. Fixed for the session - see [scan.Profiling].
	Profiling *scan.Profiling

	// ConfigPath is the configuration file that preset the flags, and ConfigSet the flags it decided. Carried only to
	// be reported: by the time the model exists the answers are already in Options, and what is worth saying is that
	// they did not all come from the command line.
	ConfigPath string
	ConfigSet  []string
}

// New builds the root model around a ready-made scan config; the source tree browses cfg.WorkDir.
//
// Taking the whole Options rather than a handful of arguments means a new CLI flag needs no signature change here
// - and the boolean knobs the overlay drives are the same struct the caller filled in.
//
// A file watcher is started best-effort - if it can't initialize, live reload is simply unavailable and the user
// falls back to r (manual rescan).
func New(start Startup) *Model {
	m := &Model{
		cfg:        start.Options,
		profiling:  start.Profiling,
		configPath: start.ConfigPath,
		configSet:  len(start.ConfigSet),
		focused:    paneTree,
		leftPct:    defaultLeftPct,
		diagPct:    defaultDiagPct,
		scan:       NewScanState(),
		search:     NewSearchBox(),
		watch:      NewSourceWatch(start.Options.WorkDir),
		tree:       panels.NewTree(start.Options.WorkDir),
		fileView:   panels.NewFileView(),
		spec:       panels.NewSpec(),
		diag:       panels.NewDiagnostics(),
		help:       help.New(),
		confirm:    confirm.New(),
		reference:  reference.New(),
		runstats:   runstats.New(),
	}
	// The options rows bind to the scan-config booleans by pointer, so the overlay writes straight into the config every
	// scan is started from. A scan is handed a copy of it (see [scan.Run]), which is what keeps a toggle from reaching
	// a run already in flight.
	m.options = options.New(m.cfg)

	return m
}

// Close releases the file watcher.
//
// Call after the program exits.
func (m *Model) Close() { m.watch.Close() }

// Init implements tea.Model.
//
// It kicks off the initial whole-scope scan and, when a watcher is available, begins listening for source changes.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.startScan()}
	if m.watch.Listening() {
		cmds = append(cmds, waitForFS(m.watch.Events()))
	}
	if cmd := m.announceConfig(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// Update implements [tea.Model].
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

	case validationMsg:
		return m, m.absorbValidation(msg)

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

// announceConfig says, once, that a file decided some of what this session started with.
//
// On the status line rather than in the diagnostics pane: it is a fact about the command, not about the code being
// scanned, and it expires like every other notice. Silent when there was no file, which is the ordinary case.
func (m *Model) announceConfig() tea.Cmd {
	if m.configPath == "" {
		return nil
	}
	if m.configSet == 0 {
		return m.notify("read %s (it set nothing)", m.configPath)
	}

	return m.notify("read %s (%d %s)", m.configPath, m.configSet, plural(m.configSet, "setting"))
}

// plural renders a count's noun, so a notice does not say "1 settings".
func plural(n int, noun string) string {
	if n == 1 {
		return noun
	}

	return noun + "s"
}

// copyFocused copies the focused panel's raw content to the clipboard.
//
// Asynchronously: the clipboard tool exec must not block the event loop.
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

// updateFocused forwards a message to the currently focused panel.
//
// The left pane is the tree or the file viewer, depending on leftMode.
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

// moveVSplit moves the vertical divider between the left pane and the spec, in whole steps, and relays out.
//
// Positive widens the left pane. The keys are spelled so that the divider travels in the arrow's own direction, which
// is the only mapping that stays right whichever pane you happen to be thinking about.
func (m *Model) moveVSplit(steps int) {
	m.leftPct = clampPct(m.leftPct+steps*splitStep, minLeftPct, maxLeftPct)
	m.recalcLayout()
}

// moveHSplit moves the horizontal divider above the diagnostics strip, in whole steps, and relays out.
//
// Positive grows the strip - the divider moving UP - so that here too the divider follows the arrow.
func (m *Model) moveHSplit(steps int) {
	m.diagPct = clampPct(m.diagPct+steps*splitStep, minDiagPct, maxDiagPct)
	m.recalcLayout()
}

// clampPct holds a divider inside its travel.
func clampPct(pct, lo, hi int) int { return min(max(pct, lo), hi) }

// recalcLayout distributes the terminal size across the chrome and the panes.
//
// A header line, a top row with the source tree beside the spec, a diagnostics strip, and a status line.
//
// The two dividers are placed by PERCENTAGE rather than by cell count, so resizing the terminal keeps the proportions
// the user chose instead of leaving one pane fixed while the other absorbs everything.
//
// The absolute floors below the percentages are what keeps a small terminal usable: a pane whose share rounds to
// nothing is still given the few rows or columns it needs to render its border and a line of content.
//
// The regions are stored for mouse hit-testing.
func (m *Model) recalcLayout() {
	m.diagH = max(m.height*m.diagPct/100, minDiagH)
	m.topH = max(m.height-headerH-statusH-m.diagH, minTopH)
	m.leftW = max(min(m.width*m.leftPct/100, m.width), minLeftW)
	rightW := max(m.width-m.leftW, 1)

	m.tree.SetSize(m.leftW, m.topH)
	m.fileView.SetSize(m.leftW, m.topH)
	m.spec.SetSize(rightW, m.topH)
	m.diag.SetSize(m.width, m.diagH)
	for _, o := range m.overlays() {
		o.SetSize(m.width, m.height)
	}
}
