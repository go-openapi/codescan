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

// optToggle binds an options-popup row to a boolean field of the scan config,
// with a short human description (the field names alone are cryptic).
type optToggle struct {
	label string
	desc  string
	ptr   *bool
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

	optionsOpen bool
	optCursor   int
	optDirty    bool
	optToggles  []optToggle

	specJSON   string
	specYAML   string
	specIndex  *SpecIndex   // rendered-line ↔ JSON-pointer map for the active format
	srcIndex   *SourceIndex // JSON-pointer ↔ Go source position (cross-ref linker)
	diags      []grammar.Diagnostic
	scanErr    error // hard error from the last codescan.Run, shown in the diag pane
	diagCursor int   // selected diagnostic, for diagnostic→source navigation

	watch       *watcher
	watchCh     <-chan struct{}
	debounceGen int

	// layout regions, recomputed by recalcLayout and reused for hit-testing.
	leftW, topH, diagH int

	leftMode    leftMode
	currentFile string

	follow       followMode
	followTarget string // human-readable resolved target, for the nav status badge

	tree     panels.Tree
	fileView panels.FileView
	spec     panels.Spec
	diag     panels.Diagnostics
}

// New builds the root model. workdir/packages/scanModels map directly onto
// codescan.Options; the source tree browses workdir. A file watcher is started
// best-effort — if it can't initialize, live reload is simply unavailable and
// the user falls back to `r` (manual rescan).
func New(workdir string, packages []string, scanModels bool) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	si := textinput.New()
	si.Prompt = "/"
	si.Placeholder = "search spec"

	m := &Model{
		cfg:         codescan.Options{WorkDir: workdir, Packages: packages, ScanModels: scanModels},
		focused:     paneTree,
		spin:        sp,
		searchInput: si,
		tree:        panels.NewTree(workdir),
		fileView:    panels.NewFileView(),
		spec:        panels.NewSpec(),
		diag:        panels.NewDiagnostics(),
	}
	// Options-popup rows bind to the scan-config booleans (pointers into
	// m.cfg stay valid — m is heap-allocated).
	m.optToggles = []optToggle{
		{"ScanModels", "also emit definitions for swagger:model types", &m.cfg.ScanModels},
		{"SkipExtensions", "omit x-go-* vendor extensions from the spec", &m.cfg.SkipExtensions},
		{"SetXNullableForPointers", "mark pointer fields as x-nullable: true", &m.cfg.SetXNullableForPointers},
		{"RefAliases", "emit $ref for type aliases instead of expanding them", &m.cfg.RefAliases},
		{"TransparentAliases", "treat aliases as transparent (never define them)", &m.cfg.TransparentAliases},
		{"DescWithRef", "keep a field description beside its $ref (allOf wrap)", &m.cfg.DescWithRef},
	}
	if w, err := newWatcher(workdir); err == nil {
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
		m.srcIndex = BuildSourceIndex(msg.provenance)
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

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal/input modes capture all keys until dismissed.
	if m.optionsOpen {
		return m.handleOptionsKey(msg)
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	// The open file pane: the read-only viewer navigates; the editor captures
	// input (except a few app keys).
	if m.leftMode == modeView && m.focused == paneTree {
		if m.fileView.Editing() {
			return m.handleEditKey(msg)
		}
		return m.handleViewerKey(msg)
	}

	// Diagnostics pane: select a diagnostic and follow it to source. Unhandled
	// keys fall through to the global bindings below.
	if m.focused == paneDiag {
		if cmd, handled := m.handleDiagNav(msg); handled {
			return m, cmd
		}
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
		m.spec.SetFormat("JSON")
		m.refreshSpec()
		return m, nil
	case key.CtrlY:
		m.spec.SetFormat("YAML")
		m.refreshSpec()
		return m, nil
	case key.R:
		return m, m.startScan()
	case key.O:
		m.exitFollow()
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
		m.spec.ClearSearch()
		m.spec.ClearHighlight()
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

// handleDiagNav handles diagnostics-pane selection and follow. Returns
// handled=false for keys it doesn't own, so global bindings still apply.
func (m *Model) handleDiagNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.moveDiagCursor(-1)
		return nil, true
	case key.Down, key.J:
		m.moveDiagCursor(+1)
		return nil, true
	case key.F:
		// Toggle diagnostics-driven follow mode: as the selection moves, the
		// source pane mirrors the selected diagnostic's position.
		m.toggleFollow(followDiag)
		return nil, true
	}
	return nil, false
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
		return "(no source)"
	}
	if m.currentFile != d.Pos.Filename {
		m.loadFileQuietly(d.Pos.Filename)
	}
	m.fileView.GotoLine(d.Pos.Line - 1) // follower scrolls; not focused
	return fmt.Sprintf("%s:%d", relTo(m.cfg.WorkDir, d.Pos.Filename), d.Pos.Line)
}

// handleViewerKey drives the read-only file viewer: move the highlighted nav
// line, follow it to the spec node it produced (`f`), enter the editor (`i`/
// Enter), or leave back to the tree (Esc).
func (m *Model) handleViewerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		m.fileView.NavUp()
		return m, nil
	case key.Down, key.J:
		m.fileView.NavDown()
		return m, nil
	case key.F:
		// Toggle source-driven follow mode: as the nav line moves, the spec
		// mirrors the node it produced (source→spec).
		m.toggleFollow(followSource)
		return m, nil
	case key.I, key.Enter:
		return m, m.fileView.StartEdit()
	case key.Esc:
		if m.follow != followOff {
			m.exitFollow()
			return m, nil
		}
		m.leftMode = modeBrowse
		return m, nil
	case key.Tab:
		m.focused = (m.focused + 1) % paneCount
		return m, m.syncEditFocus()
	case key.ShiftTab:
		m.focused = (m.focused + paneCount - 1) % paneCount
		return m, m.syncEditFocus()
	case key.C:
		return m, m.copyFocused()
	case key.CtrlQ, key.CtrlC:
		return m, tea.Quit
	}
	return m, nil
}

// handleEditKey routes keys to the file editor while it is focused. A few app
// keys still work: Esc returns to the read-only viewer, Ctrl-S saves, Ctrl-F
// follows the cursor line to the spec, Ctrl-Q quits, Tab moves focus. Everything
// else edits the buffer.
func (m *Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fileView.StopEdit()
		return m, nil
	case "ctrl+f":
		// One-shot jump from the cursor's source line to the spec node it
		// produced, focusing the spec. ctrl+f rather than f because the editor
		// owns plain f for typing; follow mode proper runs from the read-only
		// viewer.
		if desc, ok := m.linkSourceToSpec(); ok {
			m.fileView.Blur()
			m.focused = paneSpec
			m.notice = "→ " + desc
		} else {
			m.notice = "no spec node anchored at or above this line"
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
	if content, err := os.ReadFile(path); err != nil {
		m.fileView.SetFile(filepath.Base(path), "error reading file: "+err.Error())
	} else {
		m.fileView.SetFile(relTo(m.cfg.WorkDir, path), string(content))
	}
	m.leftMode = modeView
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

// relTo renders path relative to base when possible, else the base name.
func relTo(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(path)
}

// handleOptionsKey drives the scanner-options modal: move the cursor, toggle a
// boolean with space/enter, and apply-on-close (rescan only if something
// changed).
func (m *Model) handleOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		if m.optCursor > 0 {
			m.optCursor--
		}
	case key.Down, key.J:
		if m.optCursor < len(m.optToggles)-1 {
			m.optCursor++
		}
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
		return m.spec.Update(msg)
	case paneDiag:
		return m.diag.Update(msg)
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
}

// refreshDiagnostics re-renders the diagnostics pane from the stored diagnostics
// and the selection cursor, scrolling the selected diagnostic into view. The
// pane shows any hard error from codescan.Run first, then every
// grammar.Diagnostic the build emitted (colored by severity, paths relative to
// the work dir, the selected row highlighted); a clean scan with no diagnostics
// shows the empty state.
func (m *Model) refreshDiagnostics() {
	content, line := renderDiagnostics(m.cfg.WorkDir, m.scanErr, m.diags, m.diagCursor)
	m.diag.SetContent(content)
	if line >= 0 {
		m.diag.ScrollToLine(line)
	}
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

// refreshSpec renders the spec pane from the stored JSON/YAML per the active
// format toggle, and rebuilds the line↔pointer index for the active format
// (the spec-side half of the cross-ref linker, design §4 / build LX-spec-0).
func (m *Model) refreshSpec() {
	yamlFmt := m.spec.Format() == "YAML"
	body := m.specJSON
	if yamlFmt {
		body = m.specYAML
	}
	if body == "" {
		m.specIndex = nil
		m.spec.SetContent("(no spec generated yet)")
		return
	}
	if yamlFmt {
		m.specIndex = BuildYAMLIndex([]byte(body))
	} else {
		m.specIndex = BuildJSONIndex([]byte(body))
	}
	m.spec.SetContent(body)
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
	m.spec.HighlightLine(specLine)
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
	m.spec.ClearHighlight()
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
		if desc, ok := m.linkSourceToSpec(); ok {
			m.followTarget = desc
		} else {
			m.followTarget = "(no spec node)"
		}
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
	ptr, ok := m.specIndex.PointerAt(m.spec.TopLine())
	if !ok {
		return "(no node)"
	}
	if specLine, found := m.specIndex.LineForPointer(ptr); found {
		m.spec.MarkLine(specLine) // mark the mapped node, no scroll
	}
	pos, ok := m.srcIndex.PositionFor(ptr)
	if !ok {
		return ptr + " (no source)"
	}
	if m.currentFile != pos.Filename {
		m.loadFileQuietly(pos.Filename)
	}
	m.fileView.GotoLine(pos.Line - 1) // follower scrolls; not focused
	return fmt.Sprintf("%s → %s:%d", ptr, relTo(m.cfg.WorkDir, pos.Filename), pos.Line)
}

// linkSourceToSpec highlights (and scrolls to) the spec node produced by the
// file viewer's current line. No focus change. Returns a status description and
// whether the node was found in the current spec render.
func (m *Model) linkSourceToSpec() (string, bool) {
	if m.currentFile == "" {
		return "", false
	}
	line := m.fileView.CurrentLine() + 1 // pane rows are 0-based; source lines 1-based
	ptr, ok := m.srcIndex.PointerAt(m.currentFile, line)
	if !ok {
		return "", false
	}
	specLine, ok := m.specIndex.LineForPointer(ptr)
	if !ok {
		return ptr + " (not in view)", false
	}
	m.spec.HighlightLine(specLine) // follower scrolls + highlights
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

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}

	if m.optionsOpen {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.optionsView())
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

// optionsView renders the scanner-options modal: a bordered list of boolean
// toggles with checkboxes and a cursor caret.
func (m *Model) optionsView() string {
	labelW := 0
	for _, t := range m.optToggles {
		labelW = max(labelW, len(t.label))
	}

	var b strings.Builder
	b.WriteString(theme.Accent().Render("Scanner options"))
	b.WriteString("\n\n")

	for i, t := range m.optToggles {
		caret := "  "
		if i == m.optCursor {
			caret = "▸ "
		}
		box := "[ ]"
		if *t.ptr {
			box = "[x]"
		}
		label := fmt.Sprintf("%-*s", labelW, t.label)
		if i == m.optCursor {
			// highlight the whole row, description included
			b.WriteString(theme.Selected().Render(fmt.Sprintf("%s%s %s  %s", caret, box, label, t.desc)))
		} else {
			b.WriteString(fmt.Sprintf("%s%s %s  ", caret, box, label))
			b.WriteString(theme.Status().Render(t.desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.Status().Render("space: toggle · esc/o: apply & close"))

	return theme.Modal().Render(b.String())
}

// headerLine shows the app name, the (shortened) workdir, the active format,
// spec stats, and a scan spinner / ready indicator.
func (m *Model) headerLine() string {
	wd := shortenPath(m.cfg.WorkDir, max(m.width-44, 12))
	stats := fmt.Sprintf("%d paths · %d defs", m.numPaths, m.numDefs)

	if cur, total := m.spec.MatchInfo(); total > 0 {
		stats += fmt.Sprintf("  ·  match %d/%d", cur, total)
	}

	left := theme.Accent().Render("genspec-tui")
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
		if ptr, ok := m.specIndex.PointerAt(m.spec.TopLine()); ok {
			return theme.Status().Render("node " + ptr + "  ·  f: follow mode · /: search · tab: focus · c: copy")
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
	return theme.Accent().Render(" "+label+" ") +
		theme.Status().Render("  "+target+"   ·   esc / f: exit follow")
}

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
