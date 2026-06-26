// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// FileView is the left pane's file display. It opens READ-ONLY and navigable —
// a highlighted line you move with the cursor keys and follow to the spec — and
// switches to an editable textarea on demand (`i`), returning to the viewer on
// Esc. Disk is the source of truth; saving writes back and the watcher drives
// the rescan. A VIM/VS-Code integration is the eventual full editor.
type FileView struct {
	ta      textarea.Model
	w, h    int
	title   string
	loaded  string // content as loaded/saved, for the dirty check
	editing bool
	navLine int // 0-based highlighted line in read-only mode
	offset  int // 0-based top visible line in read-only mode
}

// NewFileView returns an empty viewer.
func NewFileView() FileView {
	ta := textarea.New()
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // no limit; whole files
	ta.Prompt = ""   // line numbers are the gutter; no extra prompt
	return FileView{ta: ta}
}

// SetSize fits the panel to outer dimensions w×h (border + title reserved).
func (p *FileView) SetSize(w, h int) {
	p.w, p.h = w, h
	p.ta.SetWidth(max(w-2, 0))
	p.ta.SetHeight(max(h-3, 0))
	p.clampOffset()
}

// SetFile loads name/content read-only at the top and marks the buffer clean.
func (p *FileView) SetFile(name, content string) {
	p.title = name
	p.ta.SetValue(content)
	p.loaded = content
	p.editing = false
	p.navLine = 0
	p.offset = 0
}

// Title returns the current file's display name.
func (p *FileView) Title() string { return p.title }

// Value returns the current (possibly edited) buffer text.
func (p *FileView) Value() string { return p.ta.Value() }

// Content returns the buffer text, for clipboard copy.
func (p *FileView) Content() string { return p.ta.Value() }

// Dirty reports whether the buffer has unsaved edits.
func (p *FileView) Dirty() bool { return p.ta.Value() != p.loaded }

// MarkClean records the current buffer as the saved baseline.
func (p *FileView) MarkClean() { p.loaded = p.ta.Value() }

// Editing reports whether the pane is in editable mode (vs the read-only viewer).
func (p *FileView) Editing() bool { return p.editing }

// CurrentLine returns the 0-based "current" line: the editor cursor row while
// editing, else the read-only nav line. Used by source→spec cross-ref nav.
func (p *FileView) CurrentLine() int {
	if p.editing {
		return p.ta.Line()
	}
	return p.navLine
}

// NavUp / NavDown move the read-only nav line by one, keeping it visible.
func (p *FileView) NavUp()   { p.gotoNav(p.navLine - 1) }
func (p *FileView) NavDown() { p.gotoNav(p.navLine + 1) }

// ScrollBy moves the nav line by delta (mouse wheel in read-only mode).
func (p *FileView) ScrollBy(delta int) { p.gotoNav(p.navLine + delta) }

// GotoLine parks the read-only nav line on the 0-based line, scrolling it into
// view. Used by cross-ref navigation to land on a node's source line. The editor
// cursor is synced lazily by StartEdit, so this stays cheap when called on every
// follow-mode scroll.
func (p *FileView) GotoLine(line int) { p.gotoNav(line) }

// StartEdit switches to the editable textarea at the current nav line.
func (p *FileView) StartEdit() tea.Cmd {
	p.syncEditorCursor()
	p.editing = true
	return p.ta.Focus()
}

// StopEdit leaves edit mode, parking the nav line where the cursor was.
func (p *FileView) StopEdit() {
	p.navLine = p.ta.Line()
	p.editing = false
	p.ta.Blur()
	p.clampOffset()
}

// Focus focuses the editor when in edit mode (the read-only viewer needs no
// textarea focus). Retained for the model's focus plumbing.
func (p *FileView) Focus() tea.Cmd {
	if p.editing {
		return p.ta.Focus()
	}
	return nil
}

// Blur removes editor focus.
func (p *FileView) Blur() { p.ta.Blur() }

// Update forwards a message to the textarea (edit mode only; the read-only
// viewer is driven by the model's nav keys).
func (p *FileView) Update(msg tea.Msg) tea.Cmd {
	if !p.editing {
		return nil
	}
	var cmd tea.Cmd
	p.ta, cmd = p.ta.Update(msg)
	return cmd
}

// View renders the bordered panel: the textarea in edit mode, a line-numbered
// read-only viewer with the nav line highlighted otherwise. A "●" marks unsaved
// edits. focused drives the border/title brightness; navActive drives the
// nav-line highlight (true when focused OR mirroring as a follow follower).
func (p *FileView) View(focused, navActive bool) string {
	name := p.title
	if name == "" {
		name = "(no file)"
	}
	if p.Dirty() {
		name += " ●"
	}
	mode := "view"
	body := p.viewerBody(navActive)
	if p.editing {
		mode = "edit"
		body = p.ta.View()
	}
	title := theme.Title(focused).Render("file · " + name + " · " + mode)
	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + body)
}

// viewerBody renders the read-only window: a right-aligned line-number gutter
// and the file text, with the nav line highlighted when navActive (the pane is
// focused, or mirroring the followed line as a follow follower).
func (p *FileView) viewerBody(navActive bool) string {
	inner := max(p.w-2, 0)
	visible := max(p.h-3, 0)
	lines := strings.Split(p.ta.Value(), "\n")
	total := len(lines)
	numW := len(strconv.Itoa(total))
	textW := max(inner-(numW+1), 0)

	var b strings.Builder
	end := min(p.offset+visible, total)
	for i := p.offset; i < end; i++ {
		row := fmt.Sprintf("%*d ", numW, i+1) + fit(lines[i], textW)
		if i == p.navLine && navActive {
			row = theme.Selected().Render(row)
		}
		b.WriteString(row)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// gotoNav clamps and sets the nav line, then re-clamps the scroll offset.
func (p *FileView) gotoNav(line int) {
	p.navLine = clamp(line, 0, max(p.lineCount()-1, 0))
	p.clampOffset()
}

// syncEditorCursor steps the textarea cursor to navLine (textarea has no
// row-setter), so toggling into edit mode keeps the line.
func (p *FileView) syncEditorCursor() {
	for p.ta.Line() < p.navLine {
		before := p.ta.Line()
		p.ta.CursorDown()
		if p.ta.Line() == before {
			break
		}
	}
	for p.ta.Line() > p.navLine {
		before := p.ta.Line()
		p.ta.CursorUp()
		if p.ta.Line() == before {
			break
		}
	}
}

func (p *FileView) lineCount() int { return strings.Count(p.ta.Value(), "\n") + 1 }

// clampOffset keeps the nav line within the visible read-only window.
func (p *FileView) clampOffset() {
	visible := max(p.h-3, 1)
	if p.navLine < p.offset {
		p.offset = p.navLine
	}
	if p.navLine >= p.offset+visible {
		p.offset = p.navLine - visible + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}
