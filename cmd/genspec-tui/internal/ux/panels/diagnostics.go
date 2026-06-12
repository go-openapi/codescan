// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Diagnostics is the bottom diagnostics panel: a scrollable viewport whose
// content is composed by the model from the scan's grammar.Diagnostic slice
// (see renderDiagnostics). It stays presentation-only — the model owns the
// diagnostic data and formatting; the panel just displays and scrolls it.
type Diagnostics struct {
	vp      viewport.Model
	w, h    int
	content string
}

// NewDiagnostics returns a Diagnostics panel with placeholder content.
func NewDiagnostics() Diagnostics {
	const placeholder = "(no diagnostics)"
	vp := viewport.New(0, 0)
	vp.SetContent(placeholder)
	return Diagnostics{vp: vp, content: placeholder}
}

// SetSize fits the panel to outer dimensions w×h (border + title reserved).
func (p *Diagnostics) SetSize(w, h int) {
	p.w, p.h = w, h
	p.vp.Width = max(w-2, 0)
	p.vp.Height = max(h-3, 0)
}

// SetContent replaces the rendered diagnostics text.
func (p *Diagnostics) SetContent(s string) {
	p.content = s
	p.vp.SetContent(s)
}

// Content returns the raw (unwrapped) panel text, for clipboard copy.
func (p *Diagnostics) Content() string { return p.content }

// ScrollToLine scrolls so the 0-based content line sits near the top (with a
// little context above). Used to keep the selected diagnostic in view.
func (p *Diagnostics) ScrollToLine(line int) { p.vp.SetYOffset(max(line-scrollContext, 0)) }

// Update forwards a message to the underlying viewport (scrolling).
func (p *Diagnostics) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return cmd
}

// View renders the bordered panel; focused brightens the border/title.
func (p *Diagnostics) View(focused bool) string {
	title := theme.Title(focused).Render("diagnostics")
	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + p.vp.View())
}
