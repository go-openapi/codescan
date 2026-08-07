// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// followMode is the cross-ref auto-follow state: off, or one pane driving the other(s).
//
// The driver keeps focus; every follower mirrors it on each cursor move (syncFollowIfActive runs after each
// key/scroll). `f` toggles it; any focus change or edit exits it.
//
// Most modes drive one follower because the driving pane is itself one of the two ends of the link. The diagnostics
// pane is not — it is a third place, naming a source position — so it drives both.
type followMode int

const (
	followOff    followMode = iota
	followSpec              // spec drives, the source pane follows
	followSource            // the source pane drives, the spec follows
	followDiag              // the diagnostics pane drives, BOTH other panes follow
	followValidation        // the validation tab drives, the spec pane follows
)

// jumpDiagToSource opens the selected diagnostic's source line and MOVES FOCUS there — the one-shot counterpart to
// `f` follow mode, matching `g` from the tree and Enter in the spec pane.
//
// Follow mode is for reading down a list; this is for stopping on one and going to work on it.
func (m *Model) jumpDiagToSource() tea.Cmd {
	if len(m.scan.Diags) == 0 {
		return nil
	}

	d := m.scan.Diags[m.diagCursor]
	if !d.Pos.IsValid() || d.Pos.Filename == "" {
		return m.notify("(diagnostic carries no position)")
	}

	m.loadFileQuietly(d.Pos.Filename)
	m.fileView.GotoLine(d.Pos.Line - 1)
	m.focused = paneTree
	return m.notify("→ %s:%d", relTo(m.cfg.WorkDir, d.Pos.Filename), d.Pos.Line)
}

// followSep joins the two halves of the diagnostic driver's status badge.
const followSep = "  ·  "

// driveDiagFollowers mirrors BOTH followers to the selected diagnostic, WITHOUT moving focus (the diag pane stays the
// driver).
//
// The diag driver feeds two followers where the others feed one. A diagnostic is *about* a place in the source, but
// what you generally want to see next is what that place turned into — and for a diagnostic that reports a shape rather
// than a syntax error, the spec side is the whole question.
//
// Returns the composed target for the status badge.
func (m *Model) driveDiagFollowers() string {
	if len(m.scan.Diags) == 0 {
		return "(no diagnostics)"
	}
	d := m.scan.Diags[m.diagCursor]
	if !d.Pos.IsValid() || d.Pos.Filename == "" {
		return "(diagnostic carries no position)"
	}

	// Reported as two independent halves rather than one verdict: a diagnostic whose line produced no spec node is the
	// ordinary case — a parse error means nothing was built from it — and collapsing that into a single "not found"
	// would hide that the source half resolved perfectly well.
	return m.driveDiagToSource(d) + followSep + m.driveDiagToSpec(d)
}

// driveDiagToSource mirrors the source follower to the diagnostic's own position.
//
// The position rides on the diagnostic itself, so no index lookup is needed — which is why this half cannot miss once
// the caller has checked the position is valid.
func (m *Model) driveDiagToSource(d grammar.Diagnostic) string {
	if m.currentFile != d.Pos.Filename {
		m.loadFileQuietly(d.Pos.Filename)
	}
	m.fileView.GotoLine(d.Pos.Line - 1) // follower centres on the target; not focused

	return fmt.Sprintf("%s:%d", relTo(m.cfg.WorkDir, d.Pos.Filename), d.Pos.Line)
}

// driveDiagToSpec mirrors the spec follower to the node the diagnostic's source line produced.
//
// This goes through the SAME nearest-anchor walk the source pane's own `f` uses, rather than anything diagnostic-
// specific: "what did this line turn into" has one answer, and it should not depend on which pane asked.
//
// On a miss the spec follower holds position and the reason is returned, matching how driveSpecToSource treats its own.
func (m *Model) driveDiagToSpec(d grammar.Diagnostic) string {
	t := resolveSourceToSpec(m.srcIndex, m.specIndex, d.Pos.Filename, d.Pos.Line)
	if !t.Found {
		return t.Miss
	}

	m.spec.JumpTo(t.Line) // follower centres on the produced node; not focused

	return t.Pointer
}

// refreshSpec renders the spec pane from the stored JSON/YAML per the active format toggle, and rebuilds the
// line↔pointer index for the active format (the spec-side half of the cross-ref linker).
func (m *Model) refreshSpec() {
	yamlFmt := m.spec.Format() == "YAML"
	body := m.scan.Body(yamlFmt)

	// Remember the NODE under the cursor before anything is rebuilt.
	// Every re-render renumbers lines — a rescan that gains a definition above you shifts everything below it — so
	// carrying the line number across would silently move the user to a different node.
	// This is the hot path: every save fires it, and live-reload is the tool's whole point.
	anchor, anchored := "", false
	if m.specIndex != nil {
		anchor, anchored = m.specIndex.PointerAt(m.spec.CursorLine())
	}

	// Both indexes and the find-references cycle are per-render: a rescan or a format toggle invalidates every line number
	// they hold.
	m.refs.Reset()
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
// When the node itself is gone — you deleted the type that produced it — the walk falls back to its nearest
// surviving ancestor, so you land in the right neighbourhood rather than somewhere arbitrary.
// When nothing on its path survives, the clamped line SetContent already chose stands; there is nothing more honest to
// say.
//
// It scrolls MINIMALLY rather than centring: after a rescan the node has usually moved by a line or two and is still on
// screen, and yanking the viewport on every save would be worse than the drift it fixes.
// An explicit format switch recentres afterwards (setSpecFormat), that being a deliberate change of view.
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

// setSpecFormat switches the spec render between JSON and YAML. refreshSpec keeps the cursor on the same NODE across
// the re-render; this additionally recentres it, because switching format is a deliberate change of view and the node
// has usually moved far enough that a minimal scroll would leave it pinned against an edge.
//
// A no-op when the format is already active.
func (m *Model) setSpecFormat(format string) {
	if m.spec.Format() == format {
		return
	}

	m.spec.SetFormat(format)
	m.refreshSpec()
	m.spec.JumpTo(m.spec.CursorLine())
}

// cycleRefs steps through the places the node under the spec cursor is referenced (multi-candidate case): dir +1 for
// the next site, -1 for the previous, wrapping.
//
// A cycle continues only while the cursor is still parked on the site the last step put it on.
// Move it and the next F3 re-anchors on the node you are now on — which is what makes "F3 repeatedly" walk one
// definition's uses rather than chasing the definition of whatever it last landed on.
func (m *Model) cycleRefs(dir int) tea.Cmd {
	if m.refs.ParkedOn(m.spec.CursorLine()) {
		m.refs.Step(dir)
	} else {
		// Drop the old cycle FIRST: if the new node has no references we must not leave "ref 1/3 of /definitions/User" on
		// screen while the user is looking at something else.
		m.refs.Reset()
		if cmd, ok := m.startRefCycle(dir); !ok {
			return cmd
		}
	}

	m.spec.JumpTo(m.refs.Site().Line)
	m.refs.Status = m.refs.Describe()

	return nil
}

// startRefCycle begins a new cycle anchored on the node under the spec cursor, entering at the first site for a forward
// step and the last for a backward one.
//
// On failure it reports ok=false along with the notice command explaining why, rather than leaving the caller to
// remember that a message is waiting to be cleared.
func (m *Model) startRefCycle(dir int) (tea.Cmd, bool) {
	ptr, ok := m.specIndex.PointerAt(m.spec.CursorLine())
	if !ok {
		return m.notify("%s", noNodeDesc), false
	}
	anchor, sites := m.refIndex.RefsNear(ptr)
	if len(sites) == 0 {
		return m.notify("nothing references %s", ptr), false
	}

	m.refs.Start(anchor, sites, dir)

	return nil, true
}

// gotoDefinition follows the $ref under the spec cursor to the node it points at.
//
// This is the inverse of cycleRefs.
// Only local (`#/…`) refs are followable: the TUI renders one spec and is not a $ref resolver, so an external target
// is reported honestly rather than guessed at.
func (m *Model) gotoDefinition() tea.Cmd {
	t := resolveRefToSpec(m.refIndex, m.specIndex, m.spec.CursorLine())
	if !t.Found {
		return m.notify("%s", t.Miss)
	}

	m.refs.Reset()
	m.spec.JumpTo(t.Line)

	return m.notify("→ %s", t.Pointer)
}

// rebuildGutters recomputes the link markers for both panes: the discoverability layer that says which lines actually
// lead somewhere, now that both indexes exist.
//
// The spec side is driven from the SOURCE index rather than by walking the spec: only pointers with an anchor of their
// OWN get a dot.
// Marking everything that merely resolves through an ancestor would dot nearly every line — true, since
// nearest-ancestor resolution almost always finds something, and useless for the same reason.
//
// A dot therefore means "following this lands exactly here".
func (m *Model) rebuildGutters() {
	m.spec.SetGutter(m.specGutter())
	m.fileView.SetAnchors(m.srcIndex.AnchorLines(m.currentFile))
}

// specGutter maps rendered lines to their marker: an anchored node, or a followable $ref.
//
// Returns nil when there is nothing to mark, which keeps the gutter column off entirely.
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
	// A $ref line is navigable via Enter even though the $ref member itself is never anchored, so it wins the column where
	// the two would collide.
	for _, line := range m.refIndex.LocalRefLines() {
		g[line] = panels.GutterRef
	}
	if len(g) == 0 {
		return nil
	}

	return g
}

// locateInSpec jumps the spec pane to the first node produced by the given source file (position-backed, via the
// SourceIndex), highlighting it and focusing the spec.
//
// The exact replacement for the retired name-matching linker.
func (m *Model) locateInSpec(path string) tea.Cmd {
	t := resolveFileToSpec(m.srcIndex, m.specIndex, path)
	if !t.Found {
		return m.notify("%s", t.Miss)
	}

	m.spec.JumpTo(t.Line)
	m.focused = paneSpec

	return m.notify("→ %s", t.Pointer)
}

// toggleFollow turns the given follow mode on (driving from the current pane) or off if it is already active, doing an
// immediate first sync on entry.
func (m *Model) toggleFollow(mode followMode) {
	if m.follow == mode {
		m.exitFollow()
		return
	}
	m.refs.Reset() // follow drives the viewport; the cycle's lines go stale
	m.follow = mode
	m.syncFollowIfActive()
}

// exitFollow leaves follow mode and drops the spec follower highlight (the source nav line is the viewer's own cursor,
// so it stays).
func (m *Model) exitFollow() {
	if m.follow == followOff {
		return
	}
	m.follow = followOff
	m.followTarget = ""
}

// syncFollowIfActive re-mirrors the follower pane from the driver's current position.
//
// Runs after every key/scroll.
// A focus change away from the driver (or starting to edit) exits follow mode rather than mirroring stale state.
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
		// The description is meaningful on both outcomes — show it either way rather than flattening every miss to one
		// opaque message.
		m.followTarget, _ = m.linkSourceToSpec()
	case followDiag:
		if m.focused != paneDiag {
			m.exitFollow()
			return
		}
		m.followTarget = m.driveDiagFollowers()
	case followValidation:
		if m.focused != paneDiag || m.diagTab != tabValidation {
			m.exitFollow()
			return
		}
		m.followTarget = m.driveValidationToSpec()
	case followOff:
	}
}

// driveSpecToSource mirrors the source follower to the spec node at the top of the viewport, WITHOUT moving focus or
// the spec scroll (the user drives it).
//
// Returns a human-readable target for the status badge.
func (m *Model) driveSpecToSource() string {
	t := resolveSpecToSource(m.specIndex, m.srcIndex, m.spec.CursorLine())
	if !t.Found {
		// Hold the follower where it is rather than jumping to the wrong place; the miss names which case this is.
		return t.Miss
	}

	if m.currentFile != t.Pos.Filename {
		m.loadFileQuietly(t.Pos.Filename)
	}
	m.fileView.GotoLine(t.Pos.Line - 1) // follower centres on the target; not focused

	return fmt.Sprintf("%s → %s:%d", t.Pointer, relTo(m.cfg.WorkDir, t.Pos.Filename), t.Pos.Line)
}

// linkSourceToSpec highlights (and scrolls to) the spec node produced by the file viewer's current line.
//
// No focus change.
// The description is ALWAYS meaningful — callers show it whether or not the follower moved, because "this line
// produced nothing", "nothing was anchored at all" and "the node exists but isn't rendered here" are three different
// answers the user needs to tell apart.
//
// The bool reports only whether the follower actually moved.
func (m *Model) linkSourceToSpec() (string, bool) {
	line := m.fileView.CurrentLine() + 1 // pane rows are 0-based; source lines 1-based
	t := resolveSourceToSpec(m.srcIndex, m.specIndex, m.currentFile, line)
	if !t.Found {
		return t.Miss, false
	}

	m.spec.JumpTo(t.Line) // the follower centres on the produced node

	return t.Pointer, true
}
