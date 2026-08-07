// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Overlay is a modal layer.
//
// It covers the base UI, captures every key until it dismisses itself,
// and renders a self-contained box the model draws over everything else.
//
// Like the panels, an overlay is a concrete type the model owns and drives - it is deliberately NOT a tea.Model. An
// overlay shaped as a tea.Model has to be handed the root model just to hand it back, and ends up deciding app policy
// (quitting) on the root's behalf. The panels never needed that, and neither do these.
//
// Whatever an overlay wants to happen when it closes, it records; the model decides what applying it means.
type Overlay interface {
	SetSize(w, h int)
	IsOpen() bool
	HandleKey(msg tea.KeyMsg) tea.Cmd
	View() string
}

// overlays lists every overlay in precedence order: the first one open captures input and covers the UI.
//
// One list, consulted by the key dispatch, the view and the layout alike - the precedence used to be spelled out
// separately in each, as parallel lists that merely happened to agree.
//
// The confirmation comes first: it is the only one holding an action back, and a question that could be buried under
// another modal would be a question the user cannot answer.
func (m *Model) overlays() []Overlay {
	return []Overlay{&m.confirm, &m.options, &m.help, &m.reference}
}

// activeOverlay returns the overlay currently covering the UI, or nil when none is.
func (m *Model) activeOverlay() Overlay {
	for _, o := range m.overlays() {
		if o.IsOpen() {
			return o
		}
	}

	return nil
}

// applyConfirm carries out the pending action when the confirmation overlay has just been answered yes.
//
// The action is cleared either way: a declined question is finished with, and leaving it armed would let the NEXT
// confirmation inherit it.
func (m *Model) applyConfirm() tea.Cmd {
	accepted, answered := m.confirm.TakeAnswer()
	if !answered {
		return nil
	}

	action := m.pendingConfirm
	m.pendingConfirm = confirmNothing
	if !accepted {
		return nil
	}

	switch action {
	case confirmReload:
		return m.reloadFile()
	case confirmNothing:
		return nil
	}

	return nil
}

// applyOptions returns the rescan command when the options overlay has just been dismissed with a changed toggle.
//
// The overlay records what the user asked for; deciding that a rescan is how it takes effect is the model's job, and
// only the model knows how to start one (the spinner rides along with it).
func (m *Model) applyOptions() tea.Cmd {
	if m.options.IsOpen() || !m.options.Dirty() {
		return nil
	}
	m.options.MarkApplied()

	return m.startScan()
}
