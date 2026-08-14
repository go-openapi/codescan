// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
)

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Quitting is app policy, so it is decided here rather than in each mode that captures keys - which is what let
	// ctrl+q mean "quit" in the help overlay, "apply and close" in the options popup, and nothing at all in the search
	// input, while the keymap advertised it under "anywhere".
	if key.MsgBinding(msg).Quit() {
		return tea.Quit
	}

	// Modal overlays capture all keys until they dismiss themselves.
	//
	// An overlay only ever RECORDS what it wants; the apply steps below are where the model decides what that means, in
	// the same order overlays() ranks them.
	if o := m.activeOverlay(); o != nil {
		cmd := o.HandleKey(msg)
		if confirmed := m.applyConfirm(); confirmed != nil {
			return confirmed
		}
		if rescan := m.applyOptions(); rescan != nil {
			return rescan
		}

		return cmd
	}
	if m.search.Active() {
		return m.handleSearchKey(msg)
	}
	// The editor captures everything except a handful of app keys.
	if m.leftMode == modeView && m.focused == paneTree && m.fileView.Editing() {
		return m.handleEditKey(msg)
	}

	// Then the focused pane gets first refusal; whatever it declines falls through to the global bindings below.
	if cmd, handled := m.routePaneKey(msg); handled {
		return cmd
	}

	if cmd, handled := m.handleSearchControl(msg); handled {
		return cmd
	}

	if cmd, handled := m.handleSplitKey(msg); handled {
		return cmd
	}

	// Checked on the raw spelling, before the lowercasing switch below turns `V` into `v` and revalidates instead of
	// switching tab.
	if msg.String() == "V" {
		m.toggleDiagTab()

		return nil
	}

	switch key.MsgBinding(msg) {
	case key.Tab:
		m.focused = (m.focused + 1) % paneCount
		return m.syncEditFocus()
	case key.ShiftTab:
		m.focused = (m.focused + paneCount - 1) % paneCount
		return m.syncEditFocus()
	case key.CtrlJ:
		m.setSpecFormat("JSON")
		return nil
	case key.CtrlY:
		m.setSpecFormat("YAML")
		return nil
	case key.R:
		return m.startScan()
	case key.F5:
		return m.requestReload()
	case key.V:
		return m.startValidation()
	case key.H, key.Question:
		m.help.Open()
		return nil
	case key.M:
		m.runstats.Open()
		return nil
	case key.O:
		m.exitFollow()
		m.refs.Reset()
		m.options.Open()
		return nil
	case key.Enter:
		// Enter on a file (in browse mode) opens it in the editor.
		// Dirs fall through to the tree (expand or collapse).
		if m.focused == paneTree && m.leftMode == modeBrowse {
			if path, isDir, ok := m.tree.Selection(); ok && !isDir {
				return m.openFile(path)
			}
		}
		return m.updateFocused(msg)
	case key.G:
		// Jump the spec to the first node the selected source file produced (position-backed locate).
		if path, isDir, ok := m.tree.Selection(); ok && !isDir {
			return m.locateInSpec(path)
		}
		return nil
	case key.F:
		// Toggle spec-driven follow mode: as the spec scrolls, the source pane mirrors the node at the top of the viewport
		// (spec→source).
		if m.focused == paneSpec {
			m.toggleFollow(followSpec)
		}
		return nil
	case key.C:
		return m.copyFocused()
	case key.Esc:
		if m.follow != followOff {
			m.exitFollow()
			return nil
		}
		m.refs.Reset()
		m.spec.ClearSearch()
		return nil
	}

	return m.updateFocused(msg)
}

// handleSearchControl handles the case-sensitive search keys.
//
// / opens the search; n and N step through matches. They are handled apart from the rest
// because MsgBinding lowercases, which would make N indistinguishable from n.
//
// Returns handled=false for anything else.
func (m *Model) handleSearchControl(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "/":
		return m.enterSearch(), true
	case "n":
		if _, total := m.spec.MatchInfo(); total > 0 {
			m.spec.Step(+1)
			return nil, true
		}
	case "N":
		if _, total := m.spec.MatchInfo(); total > 0 {
			m.spec.Step(-1)
			return nil, true
		}
	}

	return nil, false
}

// handleSplitKey moves a pane divider, each key travelling in its own arrow's direction.
//
// Reachable from every mode INCLUDING the editor: wanting more room for the pane you are typing in is exactly when you
// reach for a resize, and a textarea swallows whatever it is not explicitly denied.
//
// Returns handled=false for anything that is not a split key.
func (m *Model) handleSplitKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch key.MsgBinding(msg) {
	case key.CtrlRight:
		m.moveVSplit(+1)
	case key.CtrlLeft:
		m.moveVSplit(-1)
	case key.CtrlUp:
		m.moveHSplit(+1)
	case key.CtrlDown:
		m.moveHSplit(-1)
	default:
		return nil, false
	}

	return nil, true
}

// routePaneKey offers a key to the focused pane's own handler before the global bindings see it.
//
// Each handler reports handled=false for keys it does not own, so a pane shadows only what it genuinely needs - the
// alternative, swallowing everything, is what used to make /, o and r dead while a file was open.
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

// handleSpecNav moves the spec pane's line cursor.
//
// The pane is navigable in its own right now, so scrolling and "where the user is" are the same thing - paging moves
// the cursor with the view rather than leaving it behind off screen, where F3 or Enter would act on a node nobody can
// see.
func (m *Model) handleSpecNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.focused != paneSpec {
		return nil, false
	}

	page := max(m.topH-3, 1) // the viewport's visible height
	delta, ok := key.Nav(key.MsgBinding(msg), page, m.spec.LastLine())
	if !ok {
		return nil, false
	}
	m.spec.MoveCursor(delta)

	return nil, true
}

// handleRefNav handles the reference-navigation keys, which belong to the spec pane.
//
// F3 and shift+F3 cycle the references of the node under the cursor; Enter follows the $ref under it.
//
// Returns handled=false for every other pane, so their own bindings (notably Enter opening a file in the tree) still
// apply.
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

// handleDiagNav handles diagnostics-pane selection and follow.
//
// Returns handled=false for keys it doesn't own, so global bindings still apply.
func (m *Model) handleDiagNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.diagTab == tabValidation {
		return m.handleValidationNav(msg)
	}

	if delta, ok := key.Nav(key.MsgBinding(msg), m.diag.VisibleRows(), len(m.scan.Diags)); ok {
		m.moveDiagCursor(delta)

		return nil, true
	}

	switch key.MsgBinding(msg) {
	case key.F:
		// Toggle diagnostics-driven follow mode: as the selection moves, the source pane mirrors the selected diagnostic's
		// position.
		m.toggleFollow(followDiag)
	case key.Enter:
		return m.jumpDiagToSource(), true
	default:
		return nil, false
	}

	return nil, true
}

// handleValidationNav is handleDiagNav for the validation tab.
//
// The same keys mean the same things, pointed at the other end of the link: a validation finding names a node in the
// DOCUMENT, so f and Enter drive the spec pane where the scan tab's drive the source.
func (m *Model) handleValidationNav(msg tea.KeyMsg) (tea.Cmd, bool) {
	if delta, ok := key.Nav(key.MsgBinding(msg), m.diag.VisibleRows(), len(m.validation.Findings)); ok {
		m.moveValidationCursor(delta)

		return nil, true
	}

	switch key.MsgBinding(msg) {
	case key.F:
		m.toggleFollow(followValidation)
	case key.Enter:
		return m.jumpValidationToSpec(), true
	default:
		return nil, false
	}

	return nil, true
}

// moveDiagCursor moves the diagnostics selection by delta, clamped.
//
// The pane re-renders, so the selection is highlighted and scrolled into view.
//
// In follow mode the Update loop re-mirrors the source pane afterward (syncFollowIfActive).
func (m *Model) moveDiagCursor(delta int) {
	if len(m.scan.Diags) == 0 {
		return
	}
	m.diagCursor = min(max(m.diagCursor+delta, 0), len(m.scan.Diags)-1)
	m.refreshDiagnostics()
}

// handleViewerKey drives the read-only file viewer.
//
// It moves the highlighted nav line, follows it to the spec node it produced (f), enters the editor (i or Enter),
// or leaves back to the tree (Esc).
func (m *Model) handleViewerKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	// Checked on the RAW spelling and before the nav keys, because MsgBinding lowercases: `K` would otherwise arrive as
	// `k` and move the cursor up. Same reason `N` is handled apart from `n` for search.
	//
	// `K` is vim's own "look up what is under the cursor", and what LSP clients bind hover to - so it collides with `k`
	// for exactly the reason it is the discoverable choice here.
	if msg.String() == "K" {
		return m.showAnnotationReference(), true
	}

	if delta, ok := key.Nav(key.MsgBinding(msg), m.fileView.VisibleRows(), m.fileView.LastLine()); ok {
		m.fileView.ScrollBy(delta)

		return nil, true
	}

	switch key.MsgBinding(msg) {
	case key.F:
		// Toggle source-driven follow mode: as the nav line moves, the spec mirrors the node it produced (source→spec).
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

	// Everything else falls through to the global bindings.
	// The viewer shadows only the keys it genuinely owns: swallowing the rest used to disable `/`, `o`, `r`, `g` and the
	// format toggle for as long as a file was open, which is exactly when you most want to rescan or flip JSON↔YAML.
	return nil, false
}

// handleEditKey routes keys to the file editor while it is focused.
//
// A few app keys still work: Esc returns to the read-only viewer, Ctrl-S saves, Ctrl-F follows the cursor line to the
// spec, F5 reloads from disk, Ctrl-Q quits, Tab moves focus.
// Everything else edits the buffer.
func (m *Model) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	if cmd, handled := m.handleSplitKey(msg); handled {
		return cmd
	}

	switch msg.String() {
	case "f5":
		// Reachable from inside the editor on purpose: an unsaved buffer is exactly the state the reload guard exists
		// for, and requiring Esc first would mean leaving the mode to ask about the mode.
		return m.requestReload()
	case "esc":
		m.fileView.StopEdit()
		// The buffer may have moved under the runs computed when it was loaded, and stale spans colour by the OLD columns
		// - re-derive from what is now there rather than show a plausible lie.
		m.refreshSource()
		return nil
	case "ctrl+f":
		// One-shot jump from the cursor's source line to the spec node it produced, focusing the spec. ctrl+f rather than f
		// because the editor owns plain f for typing; follow mode proper runs from the read-only viewer.
		desc, moved := m.linkSourceToSpec()
		if !moved {
			return m.notify("%s", desc)
		}
		m.fileView.Blur()
		m.focused = paneSpec

		return m.notify("→ %s", desc)
	case "ctrl+s":
		return m.saveFile()
	case "tab":
		m.focused = (m.focused + 1) % paneCount
		return m.syncEditFocus()
	case "shift+tab":
		m.focused = (m.focused + paneCount - 1) % paneCount
		return m.syncEditFocus()
	}
	return m.fileView.Update(msg)
}

// syncEditFocus focuses the editor only when the left pane is focused in view mode AND editing.
//
// The read-only viewer needs no textarea focus.
//
// Blurs it otherwise so a backgrounded editor doesn't keep capturing input.
func (m *Model) syncEditFocus() tea.Cmd {
	if m.leftMode == modeView && m.focused == paneTree && m.fileView.Editing() {
		return m.fileView.Focus()
	}
	m.fileView.Blur()
	return nil
}

// enterSearch opens the search prompt over the status line, focusing the spec.
func (m *Model) enterSearch() tea.Cmd {
	m.exitFollow()
	m.refs.Reset()
	m.focused = paneSpec

	return m.search.Open()
}

// handleSearchKey routes keys to the search input; Enter runs the search, Esc cancels and clears highlights.
func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		q := m.search.Query()
		m.search.Close()
		if q == "" {
			m.spec.ClearSearch()

			return nil
		}
		if n := m.spec.Search(q); n == 0 {
			return m.notify("no matches: %s", q)
		}

		return nil
	case tea.KeyEsc:
		m.search.Close()
		m.spec.ClearSearch()

		return nil
	}

	return m.search.Update(msg)
}
