// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// headerH / statusH are the single-line chrome rows reserved top and bottom.
const (
	headerH = 1
	statusH = 1
)

// leftView renders whichever the left pane currently shows.
//
// The file viewer highlights its nav line when focused or when it is the active follower in spec-driven follow mode
// (where the spec keeps focus).
func (m *Model) leftView(focused bool) string {
	if m.leftMode == modeView {
		// The source pane is the active follower in spec- and diag-driven follow.
		navActive := focused || m.follow == followSpec || m.follow == followDiag
		return m.fileView.View(focused, navActive)
	}
	return m.tree.View(focused)
}

// headerLine renders the top chrome line.
//
// It shows the app name, the shortened work dir, the active format, the spec stats, and a scan spinner
// or a ready indicator.
//
// The work dir is the only elastic field, so it is given whatever the others leave rather than a fixed allowance. It
// used to have a constant one, which every other field could outgrow: the stats widen with the spec, the tail gains a
// duration once a scan has finished, and the match counter appears only while a search is active. Past that constant,
// the right-hand end of the line simply ran off the screen - and the fields that vanished were the ones reporting
// whether the scan had even finished.
func (m *Model) headerLine() string {
	stats := fmt.Sprintf("%d paths · %d defs", m.scan.NumPaths, m.scan.NumDefs)
	if cur, total := m.spec.MatchInfo(); total > 0 {
		stats += fmt.Sprintf("  ·  match %d/%d", cur, total)
	}

	// Placed right after the app name rather than at the end of the line: a long work dir must never be able to push the
	// one hint that reveals the others.
	left := theme.Accent().Render("genspec-tui") + " " + theme.Chip().Render(" h: help ")

	tail := theme.Status().Render("ready")
	switch {
	case m.scan.Running:
		tail = m.scan.Spin.View() + theme.Status().Render("scanning")
	case m.scan.Elapsed > 0:
		tail = theme.Status().Render("ready (" + humanDuration(m.scan.Elapsed) + ")")
	}

	// Measured, not guessed. Assembled around the work dir rather than through a format string, so nothing in the data
	// can be read as a verb.
	beforeWD, afterWD := "  ·  ", "  ·  "+m.spec.Format()+"  ·  "+stats+"  ·  "
	spent := lipgloss.Width(left) + lipgloss.Width(beforeWD) + lipgloss.Width(afterWD) + lipgloss.Width(tail)
	// No floor of its own: on a terminal too narrow for everything, the path yields first and shortenPath still keeps it
	// legible at its own minimum. Anything that reports scan STATE is worth more columns than the directory it ran in.
	wd := shortenPath(m.cfg.WorkDir, m.width-spent)

	line := left + theme.Status().Render(beforeWD+wd+afterWD) + tail

	// Last resort, for a terminal too narrow even for the inelastic fields: clip. Overflow wraps onto a second row,
	// which pushes every pane down one and misaligns the mouse hit-testing against what is drawn.
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}

// humanDuration renders d compactly: "947ms", "3s", "1m 3s" (minute form drops a zero-second remainder, e.g. "2m").
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

// statusLine renders the bottom line, clipped to the terminal.
//
// Same reasoning as the header: several of its variants embed something unbounded - a JSON pointer, a follow target, a
// file path - and a status line that wrapped would push the layout it sits under off by a row.
func (m *Model) statusLine() string {
	return lipgloss.NewStyle().MaxWidth(m.width).Render(m.statusContent())
}

func (m *Model) statusContent() string {
	if m.search.Active() {
		return m.search.View()
	}
	if m.follow != followOff {
		return m.followBadge()
	}
	if m.notice != "" {
		return theme.Status().Render(m.notice)
	}
	if m.refs.Status != "" {
		return theme.Accent().Render(" REFS ") +
			theme.Status().Render("  "+m.refs.Status+"   ·   F3/shift+F3: next/prev   ·   enter: go to definition   ·   esc: clear")
	}
	if m.focused == paneTree && m.leftMode == modeView {
		if m.fileView.Editing() {
			return theme.Status().Render(
				"editing · ctrl+f: jump → spec · esc: stop editing · ctrl+s: save · ctrl+q: quit")
		}
		return theme.Status().Render(
			"viewing · ↑↓/jk: line · f: follow mode · i: edit · esc: tree · tab: focus · c: copy")
	}
	if m.focused == paneDiag {
		if counter := m.diagCounter(); counter != "" {
			return theme.Status().Render(counter +
				"  ·  ↑↓/jk: select · f: follow mode · tab: focus · c: copy")
		}
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

// diagCounter names the diagnostics pane's selection, for whichever tab is showing.
//
// It reads the tab the way refreshDiagnostics does, because the two must agree: reporting the scan's cursor under a
// validation list has it count a selection the user cannot see, and the totals differ too.
//
// Returns "" when the showing tab has nothing to count, so the caller falls through to the general hint.
func (m *Model) diagCounter() string {
	if m.diagTab == tabValidation {
		if n := len(m.validation.Findings); n > 0 {
			return fmt.Sprintf("finding %d/%d", m.validation.Cursor+1, n)
		}

		return ""
	}

	if n := len(m.scan.Diags); n > 0 {
		return fmt.Sprintf("diagnostic %d/%d", m.diagCursor+1, n)
	}

	return ""
}

// followBadge renders the auto-follow status line: which pane drives, the resolved target, and how to exit.
//
// The accent label makes the mode obvious.
func (m *Model) followBadge() string {
	label := "SPEC ▸ SOURCE"
	switch m.follow {
	case followSource:
		label = "SOURCE ▸ SPEC"
	case followDiag:
		label = "DIAG ▸ SOURCE + SPEC"
	case followValidation:
		label = "VALIDATION ▸ SPEC"
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

// stale reports whether the cross-ref positions are out of date with respect to what the user is looking at.
//
// Provenance is a snapshot of the LAST scan, so an unsaved edit shifts every anchor below it in that file: the follower
// can land N lines off until Ctrl-S → watcher → rescan refreshes the index.
//
// The design leaves the choice open between suppressing reverse-nav and badging it; a badge is the non-destructive read
// - nav keeps working, it just stops pretending to be exact.
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
