// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package ux is the bubbletea front-end for genspec-tui: a single root Model
// composing a header line, three panels (source tree, spec, diagnostics), and
// a status/help line. Structure borrows from fredbi/git-janitor — one root
// model owning panel values, an enum-based key dispatch, mouse focus/scroll,
// and a recalcLayout that distributes the terminal size across panels.
package ux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/gadgets"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// headerH / statusH are the single-line chrome rows reserved top and bottom.
const (
	headerH = 1
	statusH = 1
)

// bufferTabWidth is how many spaces bubbles/textarea substitutes for a tab when
// a file is loaded into it. Not a preference of ours — a property of the widget
// that every file coordinate has to be translated through.
const bufferTabWidth = 4

// noticeTTL is how long a transient status notice (e.g. "copied to
// clipboard") stays on the status line before it clears.
const noticeTTL = 2 * time.Second

// debounceDelay coalesces a burst of file-change events (an editor save often
// fires several) into a single rescan.
const debounceDelay = 300 * time.Millisecond

// copyResultMsg is delivered after an async clipboard copy completes.
type copyResultMsg struct{ err error }

// clearNoticeMsg clears the transient status notice.
type clearNoticeMsg struct{}

// fsEventMsg signals that the watcher saw a relevant source change.
type fsEventMsg struct{}

// debounceMsg fires after the quiet period; gen guards against stale timers.
type debounceMsg struct{ gen int }

type pane int

const (
	paneTree pane = iota
	paneSpec
	paneDiag
	paneCount
)

// leftMode is what the left pane shows: the source tree or a file's content.
type leftMode int

const (
	modeBrowse leftMode = iota
	modeView
)

// followMode is the cross-ref auto-follow state: off, or one pane driving the
// other. The driver keeps focus; the follower mirrors on every cursor move
// (syncFollowIfActive runs after each key/scroll). `f` toggles it; any focus
// change or edit exits it.
type followMode int

const (
	followOff    followMode = iota
	followSpec              // spec drives, the source pane follows
	followSource            // the source pane drives, the spec follows
	followDiag              // the diagnostics pane drives, the source pane follows
)

// Cross-ref outcome descriptions. A link can fail for genuinely different
// reasons, and conflating them sends the user hunting for a bug that isn't
// there: a node with no anchored ancestor was never produced from code (design
// §3.8 — an InputSpec overlay node legitimately has no origin), whereas a node
// that resolved but isn't rendered is simply outside the active JSON/YAML view.
// Both are first-class answers, not errors, so every link helper returns one of
// these whether or not the follower moved.
const (
	noNodeDesc        = "(no node here)"
	noFileDesc        = "(no file open)"
	noAnchorDesc      = "no spec node anchored at or above this line"
	noProvenanceDesc  = "no provenance from the last scan"
	noSourceSuffix    = " · no source (not produced from code)"
	notRenderedSuffix = " · not rendered in this view"
)

// Option groups, in the order the overlay shows them. Seventeen flat rows is a
// wall; these are the same divisions CLAUDE.md already uses to describe the
// knobs, so the overlay and the docs agree on where a setting lives.
const (
	groupScope  = "discovery & scope"
	groupRefs   = "$ref & composition"
	groupNaming = "naming"
	groupDocs   = "docs & comments"
	groupTypes  = "types & extensions"

	// optionGroupCount is how many of the above appear, for sizing only.
	optionGroupCount = 5
)

// optDep records that a toggle only bites when another one holds a particular
// value. Several codescan knobs are modifiers rather than independent switches
// — PruneUnusedModels does nothing without ScanModels, EmitXGoType does nothing
// while SkipExtensions suppresses every x-go-* extension — and an overlay that
// let you tick them without saying so would quietly lie about what it did.
type optDep struct {
	ptr   *bool
	on    bool   // the value ptr must hold for the dependent toggle to matter
	label string // the dependency's name, for the explanatory suffix
}

// satisfied reports whether the dependency currently holds.
func (d *optDep) satisfied() bool { return d == nil || *d.ptr == d.on }

// note renders the "…but only when X" suffix shown while the dependency is unmet.
func (d *optDep) note() string {
	if d.on {
		return " (needs " + d.label + ")"
	}

	return " (moot: " + d.label + ")"
}

// optToggle binds an options-popup row to a boolean field of the scan config,
// with a short human description (the field names alone are cryptic) and the
// section it belongs to. Rows are stored flat and grouped at render time, so
// cursor movement stays a plain index and headers can never be landed on.
type optToggle struct {
	group string
	label string
	desc  string
	ptr   *bool
	dep   *optDep
}

// Model is the root bubbletea model.
type Model struct {
	cfg           codescan.Options
	width, height int
	ready         bool
	focused       pane
	notice        string

	scanning    bool
	spin        spinner.Model
	numPaths    int
	numDefs     int
	lastElapsed time.Duration

	searching   bool
	searchInput textinput.Model

	helpOpen   bool
	helpScroll int

	optionsOpen bool
	optCursor   int
	optDirty    bool
	optToggles  []optToggle

	specJSON   string
	specYAML   string
	specIndex  *index.SpecIndex   // rendered-line ↔ JSON-pointer map for the active format
	refIndex   *index.RefIndex    // $ref sites in the active render (find-references / go-to-definition)
	srcIndex   *index.SourceIndex // JSON-pointer ↔ Go source position (cross-ref linker)
	diags      []grammar.Diagnostic
	scanErr    error // hard error from the last codescan.Run, shown in the diag pane
	diagCursor int   // selected diagnostic, for diagnostic→source navigation

	watch       *watcher
	watchCh     <-chan struct{}
	debounceGen int

	// layout regions, recomputed by recalcLayout and reused for hit-testing.
	leftW, topH, diagH int

	leftMode      leftMode
	currentFile   string
	currentSource string // the open file as READ, whose coordinates diagnostics use

	follow       followMode
	followTarget string // human-readable resolved target, for the nav status badge

	// Find-references cycle state (F3 / shift+F3). Valid only for the CURRENT
	// render, so refreshSpec resets it.
	refAnchor string          // the definition pointer whose uses are being cycled
	refSites  []index.RefSite // its reference sites, ordered by rendered line
	refCursor int             // which site we are parked on
	refStatus string          // persistent status line while a cycle is active

	tree     panels.Tree
	fileView panels.FileView
	spec     panels.Spec
	diag     panels.Diagnostics
}

// New builds the root model around a ready-made scan config; the source tree
// browses cfg.WorkDir. Taking the whole Options rather than a handful of
// arguments means a new CLI flag needs no signature change here — and the
// boolean knobs the overlay drives are the same struct the caller filled in.
//
// A file watcher is started best-effort — if it can't initialize, live reload
// is simply unavailable and the user falls back to `r` (manual rescan).
func New(cfg codescan.Options) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	si := textinput.New()
	si.Prompt = "/"
	si.Placeholder = "search spec"

	m := &Model{
		cfg:         cfg,
		focused:     paneTree,
		spin:        sp,
		searchInput: si,
		tree:        panels.NewTree(cfg.WorkDir),
		fileView:    panels.NewFileView(),
		spec:        panels.NewSpec(),
		diag:        panels.NewDiagnostics(),
	}
	// Options-popup rows bind to the scan-config booleans (pointers into
	// m.cfg stay valid — m is heap-allocated). EVERY exported bool on
	// codescan.Options belongs here; TestOptions_OverlayCoversEveryBoolKnob
	// fails when one is added without a row, which is how this list fell eleven
	// knobs behind the v0.36 feature streak before anyone noticed.
	// Aliased because two rows depend on them (and `scanModels` is already the
	// New parameter, so these need distinct names).
	depScanModels, depSkipExtensions := &m.cfg.ScanModels, &m.cfg.SkipExtensions
	m.optToggles = []optToggle{
		// Discovery & scope.
		{groupScope, "ScanModels", "also emit swagger:model definitions", depScanModels, nil},
		{
			groupScope, "PruneUnusedModels", "drop models nothing references", &m.cfg.PruneUnusedModels,
			&optDep{depScanModels, true, "ScanModels"},
		},
		{groupScope, "ExcludeDeps", "skip packages outside the module", &m.cfg.ExcludeDeps, nil},

		// $ref & composition.
		{groupRefs, "RefAliases", "$ref aliases instead of expanding", &m.cfg.RefAliases, nil},
		{groupRefs, "TransparentAliases", "aliases never become definitions", &m.cfg.TransparentAliases, nil},
		{groupRefs, "EmitRefSiblings", "description beside $ref, not allOf", &m.cfg.EmitRefSiblings, nil},
		{
			groupRefs, "SkipAllOfCompounding", "never wrap in allOf; drops validations",
			&m.cfg.SkipAllOfCompounding, nil,
		},
		{groupRefs, "DefaultAllOfForEmbeds", "plain embeds compose via allOf", &m.cfg.DefaultAllOfForEmbeds, nil},

		// Naming.
		{groupNaming, "EmitHierarchicalNames", "nest colliding names, not concats", &m.cfg.EmitHierarchicalNames, nil},
		{groupNaming, "SkipJSONifyInterfaceMethods", "interface methods keep their names", &m.cfg.SkipJSONifyInterfaceMethods, nil},

		// Docs & comments.
		{groupDocs, "SingleLineCommentAsDescription", "one-line doc is description, not title", &m.cfg.SingleLineCommentAsDescription, nil},
		{groupDocs, "AfterDeclComments", "annotations inside or after a decl", &m.cfg.AfterDeclComments, nil},
		{groupDocs, "CleanGoDoc", "strip godoc-only syntax from prose", &m.cfg.CleanGoDoc, nil},
		{groupDocs, "SkipEnumDescriptions", "enum names only on x-go-enum-desc", &m.cfg.SkipEnumDescriptions, nil},

		// Types & extensions.
		{groupTypes, "SetXNullableForPointers", "pointer fields get x-nullable", &m.cfg.SetXNullableForPointers, nil},
		{groupTypes, "SkipExtensions", "omit x-go-* vendor extensions", depSkipExtensions, nil},
		{
			groupTypes, "EmitXGoType", "stamp x-go-type on definitions", &m.cfg.EmitXGoType,
			&optDep{depSkipExtensions, false, "SkipExtensions"},
		},
	}
	if w, err := newWatcher(cfg.WorkDir); err == nil {
		m.watch = w
		m.watchCh = w.events
	}
	return m
}

// Close releases the file watcher. Call after the program exits.
func (m *Model) Close() {
	if m.watch != nil {
		_ = m.watch.Close()
	}
}

// Init implements tea.Model: kick off the initial whole-scope scan and, if a
// watcher is available, begin listening for source changes.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.startScan()}
	if m.watchCh != nil {
		cmds = append(cmds, waitForFS(m.watchCh))
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
		model, cmd := m.handleKey(msg)
		m.syncFollowIfActive() // re-mirror the follower after a driver move
		return model, cmd

	case tea.MouseMsg:
		model, cmd := m.handleMouse(msg)
		m.syncFollowIfActive()
		return model, cmd

	case spinner.TickMsg:
		if !m.scanning {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case scanResultMsg:
		m.scanning = false
		m.specJSON, m.specYAML = msg.json, msg.yaml
		m.numPaths, m.numDefs = msg.paths, msg.defs
		m.lastElapsed = msg.elapsed
		m.diags = msg.diags
		m.scanErr = msg.err
		m.diagCursor = clampInt(m.diagCursor, 0, max(len(m.diags)-1, 0))
		m.srcIndex = index.BuildSourceIndex(msg.provenance)
		m.applyScan()
		m.syncFollowIfActive() // refresh the follower against the rebuilt spec
		return m, nil

	case fsEventMsg:
		// A change arrived: start (restart) the debounce window and keep
		// listening for the next event.
		m.debounceGen++
		return m, tea.Batch(debounceCmd(m.debounceGen), waitForFS(m.watchCh))

	case debounceMsg:
		// Rescan only if no newer change arrived during the quiet period.
		if msg.gen == m.debounceGen {
			return m, m.startScan()
		}
		return m, nil

	case copyResultMsg:
		if msg.err != nil {
			m.notice = "clipboard error: " + msg.err.Error()
		} else {
			m.notice = "copied to clipboard"
		}
		return m, clearNoticeAfter(noticeTTL)

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

	if m.optionsOpen {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.optionsView())
	}
	if m.helpOpen {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.helpView())
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

// startScan marks a scan in flight and returns the scan command, starting the
// spinner only when one isn't already running (avoids stacking tick loops).
func (m *Model) startScan() tea.Cmd {
	already := m.scanning
	m.scanning = true
	scan := runScan(m.cfg)
	if already {
		return scan
	}
	return tea.Batch(scan, m.spin.Tick)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal/input modes capture all keys until dismissed.
	if m.optionsOpen {
		return m.handleOptionsKey(msg)
	}
	if m.helpOpen {
		return m.handleHelpKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	// The editor captures everything except a handful of app keys.
	if m.leftMode == modeView && m.focused == paneTree && m.fileView.Editing() {
		return m.handleEditKey(msg)
	}

	// Then the focused pane gets first refusal; whatever it declines falls
	// through to the global bindings below.
	if cmd, handled := m.routePaneKey(msg); handled {
		return m, cmd
	}

	if mdl, cmd, handled := m.handleSearchControl(msg); handled {
		return mdl, cmd
	}

	switch key.MsgBinding(msg) {
	case key.CtrlC, key.CtrlQ:
		return m, tea.Quit
	case key.Tab:
		m.focused = (m.focused + 1) % paneCount
		return m, m.syncEditFocus()
	case key.ShiftTab:
		m.focused = (m.focused + paneCount - 1) % paneCount
		return m, m.syncEditFocus()
	case key.CtrlJ:
		m.setSpecFormat("JSON")
		return m, nil
	case key.CtrlY:
		m.setSpecFormat("YAML")
		return m, nil
	case key.R:
		return m, m.startScan()
	case key.H, key.Question:
		m.openHelp()
		return m, nil
	case key.O:
		m.exitFollow()
		m.resetRefCycle()
		m.optionsOpen = true
		m.optDirty = false
		return m, nil
	case key.Enter:
		// Enter on a file (in browse mode) opens it in the editor.
		// Dirs fall through to the tree (expand/collapse).
		if m.focused == paneTree && m.leftMode == modeBrowse {
			if path, isDir, ok := m.tree.Selection(); ok && !isDir {
				return m, m.openFile(path)
			}
		}
		return m, m.updateFocused(msg)
	case key.G:
		// Jump the spec to the first node the selected source file produced
		// (position-backed locate).
		if path, isDir, ok := m.tree.Selection(); ok && !isDir {
			return m, m.locateInSpec(path)
		}
		return m, nil
	case key.F:
		// Toggle spec-driven follow mode: as the spec scrolls, the source pane
		// mirrors the node at the top of the viewport (spec→source).
		if m.focused == paneSpec {
			m.toggleFollow(followSpec)
		}
		return m, nil
	case key.C:
		return m, m.copyFocused()
	case key.Esc:
		if m.follow != followOff {
			m.exitFollow()
			return m, nil
		}
		m.resetRefCycle()
		m.spec.ClearSearch()
		return m, nil
	}

	return m, m.updateFocused(msg)
}

// handleSearchControl handles the case-sensitive search keys (`/` opens search,
// `n`/`N` step matches) that MsgBinding would lowercase. Returns handled=false
// for anything else.
func (m *Model) handleSearchControl(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "/":
		mdl, cmd := m.enterSearch()
		return mdl, cmd, true
	case "n":
		if _, total := m.spec.MatchInfo(); total > 0 {
			m.spec.Step(+1)
			return m, nil, true
		}
	case "N":
		if _, total := m.spec.MatchInfo(); total > 0 {
			m.spec.Step(-1)
			return m, nil, true
		}
	}
	return m, nil, false
}

// routePaneKey offers a key to the focused pane's own handler before the
// global bindings see it. Each handler reports handled=false for keys it does
// not own, so a pane shadows only what it genuinely needs — the alternative,
// swallowing everything, is what used to make `/`, `o` and `r` dead while a
// file was open.
func (m *Model) routePaneKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.leftMode == modeView && m.focused == paneTree {
		return m.handleViewerKey(msg)
	}
	if m.focused == paneDiag {
		return m.handleDiagNav(msg)
	}
	if cmd, handled := m.handleSpecNav(msg); handled {
		return cmd, true
	}

	return m.handleRefNav(msg)
}

// handleSpecNav moves the spec pane's line cursor. The pane is navigable in its
// own right now, so scrolling and "where the user is" are the same thing —
// paging moves the cursor with the view rather than leaving it behind off
// screen, where F3 or Enter would act on a node nobody can see.
func (m *Model) handleSpecNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.focused != paneSpec {
		return nil, false
	}

	page := max(m.topH-3, 1) // the viewport's visible height
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.spec.MoveCursor(-1)
	case key.Down, key.J:
		m.spec.MoveCursor(+1)
	case key.PgUp:
		m.spec.MoveCursor(-page)
	case key.PgDown:
		m.spec.MoveCursor(+page)
	case key.Home:
		m.spec.SetCursor(0)
	case key.End:
		m.spec.SetCursor(m.spec.LastLine())
	default:
		return nil, false
	}

	return nil, true
}

// handleRefNav handles the Phase-D navigation keys, which belong to the spec
// pane: F3 / shift+F3 cycle the references of the node under the cursor, Enter
// follows the $ref under it. Returns handled=false for every other pane, so
// their own bindings (notably Enter opening a file in the tree) still apply.
func (m *Model) handleRefNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.focused != paneSpec {
		return nil, false
	}

	switch key.MsgBinding(msg) {
	case key.F3:
		return m.cycleRefs(+1), true
	case key.ShiftF3, key.ShiftF3Named:
		return m.cycleRefs(-1), true
	case key.Enter:
		return m.gotoDefinition(), true
	}

	return nil, false
}

// handleDiagNav handles diagnostics-pane selection and follow. Returns
// handled=false for keys it doesn't own, so global bindings still apply.
func (m *Model) handleDiagNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	page := m.diag.VisibleRows()
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.moveDiagCursor(-1)
	case key.Down, key.J:
		m.moveDiagCursor(+1)
	case key.PgUp:
		m.moveDiagCursor(-page)
	case key.PgDown:
		m.moveDiagCursor(+page)
	case key.Home:
		m.moveDiagCursor(-len(m.diags))
	case key.End:
		m.moveDiagCursor(len(m.diags))
	case key.F:
		// Toggle diagnostics-driven follow mode: as the selection moves, the
		// source pane mirrors the selected diagnostic's position.
		m.toggleFollow(followDiag)
	case key.Enter:
		return m.jumpDiagToSource(), true
	default:
		return nil, false
	}

	return nil, true
}

// jumpDiagToSource opens the selected diagnostic's source line and MOVES FOCUS
// there — the one-shot counterpart to `f` follow mode, matching `g` from the
// tree and Enter in the spec pane. Follow mode is for reading down a list;
// this is for stopping on one and going to work on it.
func (m *Model) jumpDiagToSource() tea.Cmd {
	if len(m.diags) == 0 {
		return nil
	}

	d := m.diags[m.diagCursor]
	if !d.Pos.IsValid() || d.Pos.Filename == "" {
		m.notice = "(diagnostic carries no position)"

		return clearNoticeAfter(noticeTTL)
	}

	m.loadFileQuietly(d.Pos.Filename)
	m.fileView.GotoLine(d.Pos.Line - 1)
	m.focused = paneTree
	m.notice = "→ " + fmt.Sprintf("%s:%d", relTo(m.cfg.WorkDir, d.Pos.Filename), d.Pos.Line)

	return clearNoticeAfter(noticeTTL)
}

// moveDiagCursor moves the diagnostics selection by delta (clamped) and
// re-renders the pane to highlight and scroll to it. In follow mode the Update
// loop re-mirrors the source pane afterward (syncFollowIfActive).
func (m *Model) moveDiagCursor(delta int) {
	if len(m.diags) == 0 {
		return
	}
	m.diagCursor = clampInt(m.diagCursor+delta, 0, len(m.diags)-1)
	m.refreshDiagnostics()
}

// driveDiagToSource mirrors the source follower to the selected diagnostic's
// position, WITHOUT moving focus (the diag pane stays the driver). Returns a
// human-readable target for the status badge; the position rides on the
// diagnostic itself, so no index lookup is needed.
func (m *Model) driveDiagToSource() string {
	if len(m.diags) == 0 {
		return "(no diagnostics)"
	}
	d := m.diags[m.diagCursor]
	if !d.Pos.IsValid() || d.Pos.Filename == "" {
		return "(diagnostic carries no position)"
	}
	if m.currentFile != d.Pos.Filename {
		m.loadFileQuietly(d.Pos.Filename)
	}
	m.fileView.GotoLine(d.Pos.Line - 1) // follower centres on the target; not focused
	return fmt.Sprintf("%s:%d", relTo(m.cfg.WorkDir, d.Pos.Filename), d.Pos.Line)
}

// handleViewerKey drives the read-only file viewer: move the highlighted nav
// line, follow it to the spec node it produced (`f`), enter the editor (`i`/
// Enter), or leave back to the tree (Esc).
func (m *Model) handleViewerKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.fileView.NavUp()
		return nil, true
	case key.Down, key.J:
		m.fileView.NavDown()
		return nil, true
	case key.PgUp:
		m.fileView.ScrollBy(-m.fileView.VisibleRows())
		return nil, true
	case key.PgDown:
		m.fileView.ScrollBy(m.fileView.VisibleRows())
		return nil, true
	case key.Home:
		m.fileView.ScrollBy(-m.fileView.LastLine())
		return nil, true
	case key.End:
		m.fileView.ScrollBy(m.fileView.LastLine())
		return nil, true
	case key.F:
		// Toggle source-driven follow mode: as the nav line moves, the spec
		// mirrors the node it produced (source→spec).
		m.toggleFollow(followSource)
		return nil, true
	case key.I, key.Enter:
		return m.fileView.StartEdit(), true
	case key.Esc:
		if m.follow != followOff {
			m.exitFollow()
			return nil, true
		}
		m.leftMode = modeBrowse
		return nil, true
	}

	// Everything else falls through to the global bindings. The viewer shadows
	// only the keys it genuinely owns: swallowing the rest used to disable `/`,
	// `o`, `r`, `g` and the format toggle for as long as a file was open, which
	// is exactly when you most want to rescan or flip JSON↔YAML.
	return nil, false
}

// handleEditKey routes keys to the file editor while it is focused. A few app
// keys still work: Esc returns to the read-only viewer, Ctrl-S saves, Ctrl-F
// follows the cursor line to the spec, Ctrl-Q quits, Tab moves focus. Everything
// else edits the buffer.
func (m *Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fileView.StopEdit()
		// The buffer may have moved under the runs computed when it was loaded,
		// and stale spans colour by the OLD columns — re-derive from what is
		// now there rather than show a plausible lie.
		m.refreshSource()
		return m, nil
	case "ctrl+f":
		// One-shot jump from the cursor's source line to the spec node it
		// produced, focusing the spec. ctrl+f rather than f because the editor
		// owns plain f for typing; follow mode proper runs from the read-only
		// viewer.
		desc, moved := m.linkSourceToSpec()
		if moved {
			m.fileView.Blur()
			m.focused = paneSpec
			m.notice = "→ " + desc
		} else {
			m.notice = desc
		}
		return m, clearNoticeAfter(noticeTTL)
	case "ctrl+s":
		return m, m.saveFile()
	case "ctrl+q":
		return m, tea.Quit
	case "tab":
		m.focused = (m.focused + 1) % paneCount
		return m, m.syncEditFocus()
	case "shift+tab":
		m.focused = (m.focused + paneCount - 1) % paneCount
		return m, m.syncEditFocus()
	}
	return m, m.fileView.Update(msg)
}

// loadFileQuietly loads path into the read-only viewer and switches the left
// pane to view mode WITHOUT changing focus — used by spec-driven follow, where
// the spec keeps focus while the source pane mirrors. A read error is shown in
// the buffer.
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
// The editor widget treats a CR as a line break of its own, so a file with
// Windows endings loads as twice as many lines with a blank between each. Line
// numbers below the first CR are then wrong in the buffer while right in the
// file — and the line number is the coordinate the whole cross-reference layer
// is keyed on: provenance anchors, follow mode, go-to-definition, diagnostic
// marks. Normalising on the way in gives the file, the buffer, the indexes and
// the marks one shared notion of what a line is.
//
// Saving therefore writes LF. That is the same bargain the widget already
// imposes on tabs, and it is recorded with it in the module README.
func normalizeNewlines(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}

	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// refreshSource re-derives everything the source pane knows about the open file:
// its lexical runs, the diagnostics landing in it, and the provenance anchors in
// its gutter.
//
// It runs on load, on leaving the editor, and after every rescan. The last is
// what this function exists for — a rescan replaces both the anchors and the
// diagnostics, and before it was wired the open file kept showing the previous
// scan's marks while the pane below it listed the new ones.
func (m *Model) refreshSource() {
	if m.currentFile == "" {
		return
	}
	// Tokenize the BUFFER, not the bytes read: textarea rewrites tabs as
	// spaces on the way in, so a span computed from the file would sit three
	// columns early per leading tab and cut a token in half.
	spans := goSpans(m.currentFile, []byte(m.fileView.Value()))
	m.fileView.SetSpans(index.MarkDiagnostics(spans, m.sourceMarks()))
	m.fileView.SetAnchors(m.srcIndex.AnchorLines(m.currentFile))
}

// bufferColumn converts a 1-based BYTE column in a file line — what go/token
// reports — into the 1-based RUNE column of the same character as displayed.
//
// Two conversions in one, and both are needed: multi-byte runes make a byte
// column drift from a rune column, and textarea substitutes four spaces for
// every tab, so leading indentation is wider on screen than in the file.
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

// goSpans returns the syntax runs for a Go source file, and nil for anything
// else — the tree happily opens go.mod, a golden JSON fixture or a README, and
// a Go tokenizer has nothing true to say about those.
func goSpans(path string, content []byte) map[int][]theme.Span {
	if filepath.Ext(path) != ".go" {
		return nil
	}

	return index.BuildGoHighlight(content).All()
}

// sourceMarks locates the last scan's diagnostics for the open file in the
// coordinates the pane draws in.
func (m *Model) sourceMarks() []index.DiagMark {
	if m.currentSource == "" {
		return nil
	}

	lines := strings.Split(m.currentSource, "\n")
	marks := make([]index.DiagMark, 0, len(m.diags))
	for _, d := range m.diags {
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

// openFile loads path into the read-only viewer and focuses it. The viewer is
// navigable immediately; `i` enters the editor.
func (m *Model) openFile(path string) tea.Cmd {
	m.loadFileQuietly(path)
	m.focused = paneTree
	return nil
}

// saveFile writes the editor buffer back to disk. The watcher then triggers a
// rescan, so the spec reflects the edit.
func (m *Model) saveFile() tea.Cmd {
	if m.currentFile == "" {
		return nil
	}
	if err := os.WriteFile(m.currentFile, []byte(m.fileView.Value()), 0o644); err != nil { //nolint:gosec // user's own source tree
		m.notice = "save failed: " + err.Error()
		return clearNoticeAfter(noticeTTL)
	}
	m.fileView.MarkClean()
	m.notice = "saved " + relTo(m.cfg.WorkDir, m.currentFile)
	return clearNoticeAfter(noticeTTL)
}

// relTo renders path relative to base when possible, else the base name.
func relTo(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(path)
}

// syncEditFocus focuses the editor only when the left pane is focused in view
// mode AND editing; the read-only viewer needs no textarea focus. Blurs it
// otherwise so a backgrounded editor doesn't keep capturing input.
func (m *Model) syncEditFocus() tea.Cmd {
	if m.leftMode == modeView && m.focused == paneTree && m.fileView.Editing() {
		return m.fileView.Focus()
	}
	m.fileView.Blur()
	return nil
}

// openHelp shows the keymap overlay, always from the top: it is opened to look
// something up, not resumed.
func (m *Model) openHelp() {
	m.helpOpen = true
	m.helpScroll = 0
}

// handleHelpKey drives the help overlay: scroll, or dismiss. It swallows
// everything else — the overlay covers the UI, so acting on a key the user
// cannot see the effect of would be worse than ignoring it.
func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.scrollHelp(-1)
	case key.Down, key.J:
		m.scrollHelp(+1)
	case key.PgUp:
		m.scrollHelp(-m.helpVisibleRows())
	case key.PgDown:
		m.scrollHelp(+m.helpVisibleRows())
	case key.Home:
		m.helpScroll = 0
	case key.End:
		m.scrollHelp(len(helpLines()))
	case key.Esc, key.H, key.Question, key.Enter:
		m.helpOpen = false
	case key.CtrlQ, key.CtrlC:
		return m, tea.Quit
	}

	return m, nil
}

// handleOptionsKey drives the scanner-options modal: move the cursor, toggle a
// boolean with space/enter, and apply-on-close (rescan only if something
// changed).
func (m *Model) handleOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	last := len(m.optToggles) - 1
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.optCursor = clampInt(m.optCursor-1, 0, last)
	case key.Down, key.J:
		m.optCursor = clampInt(m.optCursor+1, 0, last)
	case key.PgUp:
		m.optCursor = clampInt(m.optCursor-m.optionsVisibleRows(), 0, last)
	case key.PgDown:
		m.optCursor = clampInt(m.optCursor+m.optionsVisibleRows(), 0, last)
	case key.Home:
		m.optCursor = 0
	case key.End:
		m.optCursor = last
	case key.Space, key.Enter:
		t := m.optToggles[m.optCursor]
		*t.ptr = !*t.ptr
		m.optDirty = true
	case key.Esc, key.O, key.CtrlQ, key.CtrlC:
		m.optionsOpen = false
		if m.optDirty {
			return m, m.startScan()
		}
	}
	return m, nil
}

// enterSearch opens the search input over the status line, focusing the spec.
func (m *Model) enterSearch() (tea.Model, tea.Cmd) {
	m.exitFollow()
	m.resetRefCycle()
	m.searching = true
	m.focused = paneSpec
	m.searchInput.SetValue("")
	return m, m.searchInput.Focus()
}

// handleSearchKey routes keys to the search input; Enter runs the search, Esc
// cancels and clears highlights.
func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searching = false
		m.searchInput.Blur()
		q := m.searchInput.Value()
		if q == "" {
			m.spec.ClearSearch()
			return m, nil
		}
		if n := m.spec.Search(q); n == 0 {
			m.notice = "no matches: " + q
			return m, clearNoticeAfter(noticeTTL)
		}
		return m, nil
	case tea.KeyEsc:
		m.searching = false
		m.searchInput.Blur()
		m.spec.ClearSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// handleMouse focuses the pane under a left-click and scrolls the pane under
// the wheel — no Tab required.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	p, ok := m.paneAt(msg.X, msg.Y)
	if !ok {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m, m.scrollPane(p, msg, -1)
	case tea.MouseButtonWheelDown:
		return m, m.scrollPane(p, msg, +1)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			m.focused = p
			return m, m.syncEditFocus()
		}
	}
	return m, nil
}

// scrollPane scrolls the given pane: the tree moves its cursor; the viewport
// panes handle the wheel event natively.
func (m *Model) scrollPane(p pane, msg tea.MouseMsg, delta int) tea.Cmd {
	switch p {
	case paneTree:
		if m.leftMode == modeView {
			if m.fileView.Editing() {
				return m.fileView.Update(msg) // textarea handles its own scroll
			}
			m.fileView.ScrollBy(delta) // read-only viewer moves its nav line
			return nil
		}
		m.tree.ScrollBy(delta)
		return nil
	case paneSpec:
		m.spec.MoveCursor(delta) // the cursor leads; the view follows it
		return nil
	case paneDiag:
		m.moveDiagCursor(delta) // the selection leads; the view follows it
		return nil
	}
	return nil
}

// paneAt maps terminal coordinates to a pane, using the regions recalcLayout
// stored. Returns false for the header/status chrome rows.
func (m *Model) paneAt(x, y int) (pane, bool) {
	topStart := headerH
	topEnd := topStart + m.topH
	switch {
	case y >= topStart && y < topEnd:
		if x < m.leftW {
			return paneTree, true
		}
		return paneSpec, true
	case y >= topEnd && y < topEnd+m.diagH:
		return paneDiag, true
	}
	return 0, false
}

// applyScan updates the spec and diagnostics panes from the latest scan.
func (m *Model) applyScan() {
	m.refreshSpec()
	m.refreshDiagnostics()
	m.refreshSource()
}

// clampInt clamps v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// refreshDiagnostics re-renders the diagnostics pane from the stored diagnostics
// and the selection cursor, scrolling the selected diagnostic into view. The
// pane shows any hard error from codescan.Run first, then every
// grammar.Diagnostic the build emitted (colored by severity, paths relative to
// the work dir, the selected row highlighted); a clean scan with no diagnostics
// shows the empty state.
func (m *Model) refreshDiagnostics() {
	content, line := renderDiagnostics(m.cfg.WorkDir, m.scanErr, m.diags, m.diagCursor, m.focused == paneDiag)
	m.diag.SetContent(content)
	if line >= 0 {
		m.diag.RevealLine(line)
	}
}

// refreshSpec renders the spec pane from the stored JSON/YAML per the active
// format toggle, and rebuilds the line↔pointer index for the active format
// (the spec-side half of the cross-ref linker, design §4 / build LX-spec-0).
func (m *Model) refreshSpec() {
	yamlFmt := m.spec.Format() == "YAML"
	body := m.specJSON
	if yamlFmt {
		body = m.specYAML
	}

	// Remember the NODE under the cursor before anything is rebuilt. Every
	// re-render renumbers lines — a rescan that gains a definition above you
	// shifts everything below it — so carrying the line number across would
	// silently move the user to a different node. This is the hot path: every
	// save fires it, and live-reload is the tool's whole point.
	anchor, anchored := "", false
	if m.specIndex != nil {
		anchor, anchored = m.specIndex.PointerAt(m.spec.CursorLine())
	}

	// Both indexes and the find-references cycle are per-render: a rescan or a
	// format toggle invalidates every line number they hold.
	m.resetRefCycle()
	if body == "" {
		m.specIndex, m.refIndex = nil, nil
		m.spec.SetSpans(nil)
		m.spec.SetContent("(no spec generated yet)")
		m.rebuildGutters()
		return
	}
	built := index.BuildJSONIndex([]byte(body))
	if yamlFmt {
		built = index.BuildYAMLIndex([]byte(body))
	}
	m.specIndex, m.refIndex = built.Spec, built.Refs
	m.spec.SetSpans(built.Highlight.All())
	m.spec.SetContent(body)
	if anchored {
		m.restoreCursorTo(anchor)
	}
	m.rebuildGutters()
}

// restoreCursorTo puts the cursor back on ptr in the freshly built index.
//
// When the node itself is gone — you deleted the type that produced it — the
// walk falls back to its nearest surviving ancestor, so you land in the right
// neighbourhood rather than somewhere arbitrary. When nothing on its path
// survives, the clamped line SetContent already chose stands; there is nothing
// more honest to say.
//
// It scrolls MINIMALLY rather than centring: after a rescan the node has usually
// moved by a line or two and is still on screen, and yanking the viewport on
// every save would be worse than the drift it fixes. An explicit format switch
// recentres afterwards (setSpecFormat), that being a deliberate change of view.
func (m *Model) restoreCursorTo(ptr string) {
	for ptr != "" {
		if line, ok := m.specIndex.LineForPointer(ptr); ok {
			m.spec.SetCursor(line)
			return
		}
		i := strings.LastIndexByte(ptr, '/')
		if i < 0 {
			return
		}
		ptr = ptr[:i]
	}
}

// setSpecFormat switches the spec render between JSON and YAML. refreshSpec
// keeps the cursor on the same NODE across the re-render; this additionally
// recentres it, because switching format is a deliberate change of view and the
// node has usually moved far enough that a minimal scroll would leave it pinned
// against an edge. A no-op when the format is already active.
func (m *Model) setSpecFormat(format string) {
	if m.spec.Format() == format {
		return
	}

	m.spec.SetFormat(format)
	m.refreshSpec()
	m.spec.JumpTo(m.spec.CursorLine())
}

// cycleRefs steps through the places the node under the spec cursor is
// referenced (design §6.4 multi-candidate case): dir +1 for the next site, -1
// for the previous, wrapping.
//
// A cycle continues only while the cursor is still parked on the site the last
// step put it on. Move it and the next F3 re-anchors on the node you are now
// on — which is what makes "F3 repeatedly" walk one definition's uses rather
// than chasing the definition of whatever it last landed on.
func (m *Model) cycleRefs(dir int) tea.Cmd {
	onCurrentSite := m.refAnchor != "" && m.refCursor < len(m.refSites) &&
		m.spec.CursorLine() == m.refSites[m.refCursor].Line
	if onCurrentSite {
		m.refCursor = (m.refCursor + dir + len(m.refSites)) % len(m.refSites)
	} else {
		// Drop the old cycle FIRST: if the new node has no references we must
		// not leave "ref 1/3 of /definitions/User" on screen while the user is
		// looking at something else.
		m.resetRefCycle()
		if !m.startRefCycle(dir) {
			return clearNoticeAfter(noticeTTL)
		}
	}

	site := m.refSites[m.refCursor]
	m.spec.JumpTo(site.Line)
	m.refStatus = fmt.Sprintf("ref %d/%d of %s → %s",
		m.refCursor+1, len(m.refSites), m.refAnchor, site.Pointer)

	return nil
}

// startRefCycle begins a new cycle anchored on the node under the spec cursor,
// entering at the first site for a forward step and the last for a backward
// one. Reports false (with a notice explaining why) when there is nothing to
// cycle.
func (m *Model) startRefCycle(dir int) bool {
	ptr, ok := m.specIndex.PointerAt(m.spec.CursorLine())
	if !ok {
		m.notice = noNodeDesc

		return false
	}
	anchor, sites := m.refIndex.RefsNear(ptr)
	if len(sites) == 0 {
		m.notice = "nothing references " + ptr

		return false
	}

	m.refAnchor, m.refSites = anchor, sites
	m.refCursor = 0
	if dir < 0 {
		m.refCursor = len(sites) - 1
	}

	return true
}

// resetRefCycle drops the find-references state. Called whenever the render it
// was computed against is replaced (rescan, format toggle) or the user moves on.
func (m *Model) resetRefCycle() {
	m.refAnchor, m.refSites, m.refCursor = "", nil, 0
	m.refStatus = ""
}

// gotoDefinition follows the $ref under the spec cursor to the node it points
// at — the inverse of cycleRefs. Only local (`#/…`) refs are followable: the TUI
// renders one spec and is not a $ref resolver, so an external target is
// reported honestly rather than guessed at.
func (m *Model) gotoDefinition() tea.Cmd {
	site, ok := m.refIndex.RefAt(m.spec.CursorLine())
	if !ok {
		m.notice = "no $ref on this line"

		return clearNoticeAfter(noticeTTL)
	}
	if !site.Target.Local {
		m.notice = "external ref, not in this spec: " + site.Target.Raw

		return clearNoticeAfter(noticeTTL)
	}
	line, ok := m.specIndex.LineForPointer(site.Target.Pointer)
	if !ok {
		m.notice = site.Target.Pointer + notRenderedSuffix

		return clearNoticeAfter(noticeTTL)
	}

	m.resetRefCycle()
	m.spec.JumpTo(line)
	m.notice = "→ " + site.Target.Pointer

	return clearNoticeAfter(noticeTTL)
}

// rebuildGutters recomputes the link markers for both panes (design §6.5): the
// discoverability layer that says which lines actually lead somewhere, now that
// both indexes exist.
//
// The spec side is driven from the SOURCE index rather than by walking the spec:
// only pointers with an anchor of their OWN get a dot. Marking everything that
// merely resolves through an ancestor would dot nearly every line — true, since
// nearest-ancestor resolution almost always finds something, and useless for the
// same reason. A dot therefore means "following this lands exactly here".
func (m *Model) rebuildGutters() {
	m.spec.SetGutter(m.specGutter())
	m.fileView.SetAnchors(m.srcIndex.AnchorLines(m.currentFile))
}

// specGutter maps rendered lines to their marker: an anchored node, or a
// followable $ref. Returns nil when there is nothing to mark, which keeps the
// gutter column off entirely.
func (m *Model) specGutter() map[int]rune {
	if m.specIndex == nil {
		return nil
	}

	g := make(map[int]rune)
	for _, ptr := range m.srcIndex.AnchoredPointers() {
		if line, ok := m.specIndex.LineForPointer(ptr); ok {
			g[line] = panels.GutterAnchor
		}
	}
	// A $ref line is navigable via Enter even though the $ref member itself is
	// never anchored, so it wins the column where the two would collide.
	for _, line := range m.refIndex.LocalRefLines() {
		g[line] = panels.GutterRef
	}
	if len(g) == 0 {
		return nil
	}

	return g
}

// locateInSpec jumps the spec pane to the first node produced by the given
// source file (position-backed, via the SourceIndex), highlighting it and
// focusing the spec. The exact replacement for the retired name-matching linker.
func (m *Model) locateInSpec(path string) tea.Cmd {
	ptr, ok := m.srcIndex.FirstAnchor(path)
	if !ok {
		m.notice = "no spec node produced by " + filepath.Base(path)
		return clearNoticeAfter(noticeTTL)
	}
	specLine, ok := m.specIndex.LineForPointer(ptr)
	if !ok {
		m.notice = "node not in the current spec view: " + ptr
		return clearNoticeAfter(noticeTTL)
	}
	m.spec.JumpTo(specLine)
	m.focused = paneSpec
	m.notice = "→ " + ptr
	return clearNoticeAfter(noticeTTL)
}

// toggleFollow turns the given follow mode on (driving from the current pane)
// or off if it is already active, doing an immediate first sync on entry.
func (m *Model) toggleFollow(mode followMode) {
	if m.follow == mode {
		m.exitFollow()
		return
	}
	m.resetRefCycle() // follow drives the viewport; the cycle's lines go stale
	m.follow = mode
	m.syncFollowIfActive()
}

// exitFollow leaves follow mode and drops the spec follower highlight (the
// source nav line is the viewer's own cursor, so it stays).
func (m *Model) exitFollow() {
	if m.follow == followOff {
		return
	}
	m.follow = followOff
	m.followTarget = ""
}

// syncFollowIfActive re-mirrors the follower pane from the driver's current
// position. Runs after every key/scroll. A focus change away from the driver
// (or starting to edit) exits follow mode rather than mirroring stale state.
func (m *Model) syncFollowIfActive() {
	switch m.follow {
	case followSpec:
		if m.focused != paneSpec {
			m.exitFollow()
			return
		}
		m.followTarget = m.driveSpecToSource()
	case followSource:
		if m.focused != paneTree || m.leftMode != modeView || m.fileView.Editing() {
			m.exitFollow()
			return
		}
		// The description is meaningful on both outcomes — show it either way
		// rather than flattening every miss to one opaque message.
		m.followTarget, _ = m.linkSourceToSpec()
	case followDiag:
		if m.focused != paneDiag {
			m.exitFollow()
			return
		}
		m.followTarget = m.driveDiagToSource()
	case followOff:
	}
}

// driveSpecToSource mirrors the source follower to the spec node at the top of
// the viewport, WITHOUT moving focus or the spec scroll (the user drives it).
// Returns a human-readable target for the status badge.
func (m *Model) driveSpecToSource() string {
	ptr, ok := m.specIndex.PointerAt(m.spec.CursorLine())
	if !ok {
		return noNodeDesc
	}
	pos, ok := m.srcIndex.PositionFor(ptr)
	if !ok {
		// Hold the follower where it is rather than jumping somewhere wrong
		// (design §6.4), and name which of the two misses this is.
		if m.srcIndex.Len() == 0 {
			return ptr + " · " + noProvenanceDesc
		}
		return ptr + noSourceSuffix
	}
	if m.currentFile != pos.Filename {
		m.loadFileQuietly(pos.Filename)
	}
	m.fileView.GotoLine(pos.Line - 1) // follower centres on the target; not focused
	return fmt.Sprintf("%s → %s:%d", ptr, relTo(m.cfg.WorkDir, pos.Filename), pos.Line)
}

// linkSourceToSpec highlights (and scrolls to) the spec node produced by the
// file viewer's current line. No focus change. The description is ALWAYS
// meaningful — callers show it whether or not the follower moved, because
// "this line produced nothing", "nothing was anchored at all" and "the node
// exists but isn't rendered here" are three different answers the user needs
// to tell apart. The bool reports only whether the follower actually moved.
func (m *Model) linkSourceToSpec() (string, bool) {
	if m.currentFile == "" {
		return noFileDesc, false
	}
	line := m.fileView.CurrentLine() + 1 // pane rows are 0-based; source lines 1-based
	ptr, ok := m.srcIndex.PointerAt(m.currentFile, line)
	if !ok {
		if m.srcIndex.Len() == 0 {
			return noProvenanceDesc, false
		}
		return noAnchorDesc, false
	}
	specLine, ok := m.specIndex.LineForPointer(ptr)
	if !ok {
		return ptr + notRenderedSuffix, false
	}
	m.spec.JumpTo(specLine) // the follower centres on the produced node
	return ptr, true
}

// copyFocused copies the focused panel's raw content to the clipboard,
// asynchronously (the clipboard tool exec must not block the event loop).
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

// clearNoticeAfter returns a command that emits clearNoticeMsg after d.
func clearNoticeAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearNoticeMsg{} })
}

// waitForFS blocks on the watcher channel and emits one fsEventMsg per change.
// It is re-issued after each event to form the listen loop; a closed channel
// ends the loop quietly.
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

// updateFocused forwards a message to the currently focused panel (the left
// pane is the tree or the file viewer depending on leftMode).
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

// recalcLayout distributes the terminal size: a header line, a top row with the
// source tree (1/3 width) beside the spec, a diagnostics strip, and a status
// line. The regions are stored for mouse hit-testing.
func (m *Model) recalcLayout() {
	m.diagH = max(m.height/4, 5)
	m.topH = max(m.height-headerH-statusH-m.diagH, 3)
	m.leftW = max(min(m.width/3, m.width), 1)
	rightW := max(m.width-m.leftW, 1)

	m.tree.SetSize(m.leftW, m.topH)
	m.fileView.SetSize(m.leftW, m.topH)
	m.spec.SetSize(rightW, m.topH)
	m.diag.SetSize(m.width, m.diagH)
}

// leftView renders whichever the left pane currently shows. The file viewer
// highlights its nav line when focused or when it is the active follower in
// spec-driven follow mode (where the spec keeps focus).
func (m *Model) leftView(focused bool) string {
	if m.leftMode == modeView {
		// The source pane is the active follower in spec- and diag-driven follow.
		navActive := focused || m.follow == followSpec || m.follow == followDiag
		return m.fileView.View(focused, navActive)
	}
	return m.tree.View(focused)
}

// optionsView renders the scanner-options modal: a bordered list of boolean
// toggles with checkboxes and a cursor caret.
func (m *Model) optionsView() string {
	lines, cursorLine := m.optionsLines()
	lines = windowAround(lines, cursorLine, m.optionsVisibleRows())

	var b strings.Builder
	b.WriteString(theme.Accent().Render("Scanner options"))
	fmt.Fprintf(&b, "  %s\n\n", theme.Status().Render(fmt.Sprintf("(%d)", len(m.optToggles))))
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render("↑↓/jk: move · space: toggle · esc/o: apply & close"))

	return theme.Modal().Render(b.String())
}

// optionsVisibleRows is how many rendered rows fit between the modal's chrome
// (border, padding, title, footer). The list outgrew a fixed layout at
// seventeen knobs plus headers, so it scrolls rather than overflowing a short
// terminal.
func (m *Model) optionsVisibleRows() int {
	const chrome = 10 // border 2 + padding 2 + title 2 + footer 2, with slack

	return max(m.height-chrome, 3)
}

// optionsLines renders the grouped rows and reports which rendered line the
// cursor sits on. Group headers are emitted as the group changes, so they are
// never navigable — the cursor indexes m.optToggles, not these lines.
func (m *Model) optionsLines() ([]string, int) {
	labelW := 0
	for _, t := range m.optToggles {
		labelW = max(labelW, len(t.label))
	}

	lines := make([]string, 0, len(m.optToggles)+2*optionGroupCount)
	cursorLine, lastGroup := 0, ""
	for i, t := range m.optToggles {
		if t.group != lastGroup {
			if lastGroup != "" {
				lines = append(lines, "")
			}
			lines = append(lines, theme.Accent().Render(t.group))
			lastGroup = t.group
		}
		if i == m.optCursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.optionRow(t, i == m.optCursor, labelW))
	}

	return lines, cursorLine
}

// optionRow renders one toggle. A row whose dependency is unmet is dimmed and
// says why: ticking it would otherwise look like it had done something.
func (m *Model) optionRow(t optToggle, selected bool, labelW int) string {
	box := "[ ]"
	if *t.ptr {
		box = "[x]"
	}
	caret := "  "
	if selected {
		caret = "▸ "
	}

	desc := t.desc
	inert := !t.dep.satisfied()
	if inert {
		desc += t.dep.note()
	}

	head := fmt.Sprintf("%s%s %-*s  ", caret, box, labelW, t.label)
	switch {
	case selected:
		// Highlight the whole row, description included.
		return theme.Selected().Render(head + desc)
	case inert:
		return theme.Status().Render(head + desc)
	default:
		return head + theme.Status().Render(desc)
	}
}

// windowAround returns at most size lines, scrolled so that cursor is visible
// and as far from the edges as the list allows.
func windowAround(lines []string, cursor, size int) []string {
	if len(lines) <= size {
		return lines
	}
	top := clampInt(cursor-size/2, 0, len(lines)-size)

	return lines[top : top+size]
}

// headerLine shows the app name, the (shortened) workdir, the active format,
// spec stats, and a scan spinner / ready indicator.
func (m *Model) headerLine() string {
	// The banner claims its columns before the work dir does, so it survives a
	// narrow terminal — discoverability is the whole point of it.
	wd := shortenPath(m.cfg.WorkDir, max(m.width-54, 12))
	stats := fmt.Sprintf("%d paths · %d defs", m.numPaths, m.numDefs)

	if cur, total := m.spec.MatchInfo(); total > 0 {
		stats += fmt.Sprintf("  ·  match %d/%d", cur, total)
	}

	// Placed right after the app name rather than at the end of the line: a long
	// work dir must never be able to push the one hint that reveals the others.
	left := theme.Accent().Render("genspec-tui") + " " + theme.Chip().Render(" h: help ")
	mid := theme.Status().Render(fmt.Sprintf("  ·  %s  ·  %s  ·  %s  ·  ", wd, m.spec.Format(), stats))

	tail := theme.Status().Render("ready")
	switch {
	case m.scanning:
		tail = m.spin.View() + theme.Status().Render("scanning")
	case m.lastElapsed > 0:
		tail = theme.Status().Render("ready (" + humanDuration(m.lastElapsed) + ")")
	}
	return left + mid + tail
}

// humanDuration renders d compactly: "947ms", "3s", "1m 3s" (minute form drops
// a zero-second remainder, e.g. "2m").
func humanDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	default:
		d = d.Round(time.Second)
		mins := int(d / time.Minute)
		secs := int((d % time.Minute) / time.Second)
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
}

func (m *Model) statusLine() string {
	if m.searching {
		return m.searchInput.View()
	}
	if m.follow != followOff {
		return m.followBadge()
	}
	if m.notice != "" {
		return theme.Status().Render(m.notice)
	}
	if m.refStatus != "" {
		return theme.Accent().Render(" REFS ") +
			theme.Status().Render("  "+m.refStatus+"   ·   F3/shift+F3: next/prev   ·   enter: go to definition   ·   esc: clear")
	}
	if m.focused == paneTree && m.leftMode == modeView {
		if m.fileView.Editing() {
			return theme.Status().Render(
				"editing · ctrl+f: jump → spec · esc: stop editing · ctrl+s: save · ctrl+q: quit")
		}
		return theme.Status().Render(
			"viewing · ↑↓/jk: line · f: follow mode · i: edit · esc: tree · tab: focus · c: copy")
	}
	if m.focused == paneDiag && len(m.diags) > 0 {
		return theme.Status().Render(fmt.Sprintf(
			"diagnostic %d/%d  ·  ↑↓/jk: select · f: follow mode · tab: focus · c: copy",
			m.diagCursor+1, len(m.diags)))
	}
	if m.focused == paneSpec && m.specIndex.Len() > 0 {
		if ptr, ok := m.specIndex.PointerAt(m.spec.CursorLine()); ok {
			hint := "f: follow · F3: find refs · enter: go to definition · /: search · tab: focus · c: copy"
			if _, isRef := m.refIndex.RefAt(m.spec.CursorLine()); !isRef {
				// Nothing to follow from here; don't advertise it.
				hint = "f: follow · F3: find refs · /: search · tab: focus · c: copy"
			}
			return theme.Status().Render("node " + ptr + "  ·  " + hint)
		}
	}
	return theme.Status().Render(
		"tab/click: focus · enter: open file · g: locate · /: search · n/N: next/prev · o: options · c: copy · r: rescan · ctrl+q: quit")
}

// followBadge renders the auto-follow status line: which pane drives, the
// resolved target, and how to exit. The accent label makes the mode obvious.
func (m *Model) followBadge() string {
	label := "SPEC ▸ SOURCE"
	switch m.follow {
	case followSource:
		label = "SOURCE ▸ SPEC"
	case followDiag:
		label = "DIAG ▸ SOURCE"
	case followSpec, followOff:
	}
	target := m.followTarget
	if target == "" {
		target = "(move the cursor)"
	}
	badge := theme.Accent().Render(" " + label + " ")
	if m.stale() {
		badge += theme.Stale().Render(" STALE ")
	}
	return badge + theme.Status().Render("  "+target+"   ·   esc / f: exit follow")
}

// stale reports whether the cross-ref positions are out of date with respect to
// what the user is looking at. Provenance is a snapshot of the LAST scan, so an
// unsaved edit shifts every anchor below it in that file: the follower can land
// N lines off until Ctrl-S → watcher → rescan refreshes the index. Design §6.4
// leaves the choice open between suppressing reverse-nav and badging it; a badge
// is the non-destructive read — nav keeps working, it just stops pretending to
// be exact.
func (m *Model) stale() bool { return m.fileView.Dirty() }

// shortenPath trims a path from the left with an ellipsis so it fits maxLen.
func shortenPath(p string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	r := []rune(p)
	if len(r) <= maxLen {
		return p
	}
	return "…" + string(r[len(r)-maxLen+1:])
}
