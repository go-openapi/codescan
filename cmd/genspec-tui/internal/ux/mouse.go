// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleMouse focuses the pane under a left-click and scrolls the pane under the wheel — no Tab required.
func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	p, ok := m.paneAt(msg.X, msg.Y)
	if !ok {
		return nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.scrollPane(p, msg, -1)
	case tea.MouseButtonWheelDown:
		return m.scrollPane(p, msg, +1)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			m.focused = p
			return m.syncEditFocus()
		}
	}
	return nil
}

// scrollPane scrolls the given pane: the tree moves its cursor; the viewport panes handle the wheel event natively.
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

// paneAt maps terminal coordinates to a pane, using the regions recalcLayout stored.
//
// Returns false for the header/status chrome rows.
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
