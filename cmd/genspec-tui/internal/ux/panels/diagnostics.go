// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Diagnostics is the bottom diagnostics panel: a scrollable viewport whose content is composed by the model from the
// scan's grammar.Diagnostic slice (see renderDiagnostics).
//
// It stays presentation-only — the model owns the diagnostic data and formatting; the panel just displays and scrolls
// it.
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

// RevealLine scrolls the minimum distance that brings the 0-based content line into view, and nothing at all when it is
// already visible.
//
// Stepping the selection must not shift the whole list under the reader — the same rule the spec pane and the source
// viewer follow.
func (p *Diagnostics) RevealLine(line int) {
	switch {
	case line < p.vp.YOffset:
		p.vp.SetYOffset(line)
	case line >= p.vp.YOffset+p.vp.Height:
		p.vp.SetYOffset(line - p.vp.Height + 1)
	}
}

// TopLine is the 0-based index of the top visible content line.
func (p *Diagnostics) TopLine() int { return p.vp.YOffset }

// VisibleRows is how many content lines the viewport shows, for page-sized cursor moves.
func (p *Diagnostics) VisibleRows() int { return max(p.vp.Height, 1) }

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
