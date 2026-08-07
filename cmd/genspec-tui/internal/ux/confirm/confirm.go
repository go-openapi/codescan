// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package confirm is the yes/no modal: a question that must be answered before something irreversible happens.
//
// It follows the same contract as the other overlays — a concrete type the root model owns and drives, never a
// tea.Model. The overlay asks the question and records the answer; what the answer MEANS is the model's to decide, so
// nothing here knows what is being confirmed.
package confirm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Overlay is the yes/no modal.
//
// The zero value is a closed overlay; the root model opens it with Ask and composites its View over the base UI.
type Overlay struct {
	width, height int

	isOpen   bool
	prompt   string
	answered bool
	accepted bool
}

// New builds a closed confirmation overlay.
func New() Overlay {
	return Overlay{}
}

// SetSize fits the overlay to outer dimensions w×h.
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Ask opens the overlay on a question, discarding any answer not yet collected.
//
// The prompt should be phrased so that "yes" is the destructive reading — the overlay makes no attempt to guess which
// way round a question runs.
func (o *Overlay) Ask(prompt string) {
	o.isOpen = true
	o.prompt = prompt
	o.answered = false
	o.accepted = false
}

// TakeAnswer collects the answer exactly once, reporting ok=false while the question is unanswered or already
// collected.
//
// Consuming rather than merely reading is what keeps a single "yes" from firing an action on every subsequent keypress
// — the same reason the options overlay records that it has been applied.
func (o *Overlay) TakeAnswer() (accepted, ok bool) {
	if !o.answered {
		return false, false
	}
	o.answered = false

	return o.accepted, true
}

// HandleKey drives the overlay: `y` accepts, `n` / `esc` / `enter` decline, everything else is swallowed.
//
// The asymmetry is deliberate. Enter usually means "take the default", and for a question guarding something
// irreversible the safe default is no — so the destructive answer is the one that has to be typed on purpose.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch key.MsgBinding(msg) {
	case key.Y:
		o.close(true)
	case key.N, key.Esc, key.Enter:
		o.close(false)
	}

	return nil
}

// View renders the confirmation modal.
func (o *Overlay) View() string {
	var b strings.Builder
	b.WriteString(theme.Accent().Render(o.prompt))
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render("y: yes   ·   n / esc: no"))

	return theme.Modal().Render(b.String())
}

// close records the answer and hides the overlay.
func (o *Overlay) close(accepted bool) {
	o.isOpen = false
	o.answered = true
	o.accepted = accepted
}
