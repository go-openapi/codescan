// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"
	"time"

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

// headerLine shows the app name, the (shortened) workdir, the active format, spec stats, and a scan spinner / ready
// indicator.
func (m *Model) headerLine() string {
	// The banner claims its columns before the work dir does, so it survives a narrow terminal — discoverability is the
	// whole point of it.
	wd := shortenPath(m.cfg.WorkDir, max(m.width-54, 12))
	stats := fmt.Sprintf("%d paths · %d defs", m.scan.NumPaths, m.scan.NumDefs)

	if cur, total := m.spec.MatchInfo(); total > 0 {
		stats += fmt.Sprintf("  ·  match %d/%d", cur, total)
	}

	// Placed right after the app name rather than at the end of the line: a long work dir must never be able to push the
	// one hint that reveals the others.
	left := theme.Accent().Render("genspec-tui") + " " + theme.Chip().Render(" h: help ")
	mid := theme.Status().Render(fmt.Sprintf("  ·  %s  ·  %s  ·  %s  ·  ", wd, m.spec.Format(), stats))

	tail := theme.Status().Render("ready")
	switch {
	case m.scan.Running:
		tail = m.scan.Spin.View() + theme.Status().Render("scanning")
	case m.scan.Elapsed > 0:
		tail = theme.Status().Render("ready (" + humanDuration(m.scan.Elapsed) + ")")
	}
	return left + mid + tail
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

func (m *Model) statusLine() string {
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
	if m.focused == paneDiag && len(m.scan.Diags) > 0 {
		return theme.Status().Render(fmt.Sprintf(
			"diagnostic %d/%d  ·  ↑↓/jk: select · f: follow mode · tab: focus · c: copy",
			m.diagCursor+1, len(m.scan.Diags)))
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
// — nav keeps working, it just stops pretending to be exact.
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
