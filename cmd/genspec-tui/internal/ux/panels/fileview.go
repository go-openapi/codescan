// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// FileView is the left pane's file display.
//
// It opens READ-ONLY and navigable — a highlighted line you move with the cursor keys and follow to the spec — and
// switches to an editable textarea on demand (`i`), returning to the viewer on Esc.
// Disk is the source of truth; saving writes back and the watcher drives the rescan.
// A VIM/VS-Code integration is the eventual full editor.
type FileView struct {
	ta      textarea.Model
	w, h    int
	title   string
	loaded  string // content as loaded/saved, for the dirty check
	editing bool
	navLine int // 0-based highlighted line in read-only mode
	offset  int // 0-based top visible line in read-only mode

	anchors map[int]bool         // 1-based source lines that produced a spec node
	spans   map[int][]theme.Span // 0-based line → lexical runs; nil renders plain
}

// NewFileView returns an empty viewer.
func NewFileView() FileView {
	ta := textarea.New()
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // no limit; whole files
	ta.Prompt = ""   // line numbers are the gutter; no extra prompt
	return FileView{ta: ta}
}

// SetAnchors installs the source lines that produced a spec node, for the viewer's link gutter (design §6.5).
//
// Keyed 1-based, matching token.Position.
// A nil map renders no gutter.
func (p *FileView) SetAnchors(lines map[int]bool) { p.anchors = lines }

// SetSpans installs the per-line lexical runs used to highlight the READ-ONLY viewer, keyed by 0-based line.
//
// A nil map renders the text plain, which is what a non-Go file gets.
//
// Edit mode is deliberately not covered: bubbles/textarea owns its own rendering and emits the buffer verbatim, so
// highlighting there would mean replacing the widget rather than styling its output.
func (p *FileView) SetSpans(spans map[int][]theme.Span) { p.spans = spans }

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
	p.loaded = p.ta.Value()
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

// CurrentLine returns the 0-based "current" line: the editor cursor row while editing, else the read-only nav line.
//
// Used by source→spec cross-ref nav.
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

// GotoLine parks the read-only nav line on the 0-based line and scrolls it to the VERTICAL CENTRE, clamped at the edges
// (design §6.1).
//
// This is the JUMP primitive — cross-ref landings and follow-mode mirroring — as opposed to the nav keys, which
// move the cursor one line and scroll as little as possible.
//
// The distinction matters: a follower target moves continuously as the driver scrolls, and minimal scrolling would pin
// it to whichever edge it entered from, whereas centring keeps it still while its context slides past.
//
// The editor cursor is synced lazily by StartEdit, so this stays cheap when called on every follow-mode move.
func (p *FileView) GotoLine(line int) {
	p.navLine = min(max(line, 0), max(p.lineCount()-1, 0))
	visible := max(p.h-3, 1)
	p.offset = min(max(p.navLine-visible/2, 0), max(p.lineCount()-visible, 0))
}

// LineColAt maps a point inside the panel — panel-relative, 0-based — to a 0-based buffer line and the 1-based rune
// column of the character under it.
//
// Reports ok=false for the panel's chrome, for a row past the end of the buffer, for a column inside the prefix, and
// for edit mode, where the textarea owns its own layout and this geometry does not apply.
//
// Columns are runes, so the mapping is exact for the annotations it exists to hit and drifts only on a line mixing
// double-width characters with one — which no swagger directive does.
func (p *FileView) LineColAt(x, y int) (line, col int, ok bool) {
	// The panel draws a top border and a title row before the first line of the body.
	const chromeH, borderW = 2, 1

	if p.editing {
		return 0, 0, false
	}

	row := y - chromeH
	if row < 0 || row >= max(p.h-3, 0) {
		return 0, 0, false
	}

	line = p.offset + row
	if line < 0 || line >= p.lineCount() {
		return 0, 0, false
	}

	// Bounded on the right by the panel's own border, so a click on the frame does not resolve to text.
	if x >= p.w-borderW {
		return 0, 0, false
	}

	numW, gutW := p.prefixWidth()
	col = x - borderW - gutW - (numW + 1)
	if col < 0 {
		return 0, 0, false
	}
	col++ // 1-based, matching the span columns

	// A line too long for the pane is drawn cut, with an ellipsis in its final column (see fit). That cell shows no
	// character of the buffer, so it must not resolve to one — otherwise a click on the ellipsis reports whatever
	// happens to sit at that offset in a line the user cannot actually see the end of.
	text, _ := p.Line(line)
	textW := max(p.w-2-(numW+1)-gutW, 0)
	if len([]rune(text)) > textW && col >= textW {
		return 0, 0, false
	}

	return line, col, true
}

// Line returns the text of a 0-based buffer line, reporting ok=false when it is out of range.
func (p *FileView) Line(i int) (string, bool) {
	lines := strings.Split(p.ta.Value(), "\n")
	if i < 0 || i >= len(lines) {
		return "", false
	}

	return lines[i], true
}

// VisibleRows is how many lines the read-only window shows, for page-sized moves of the nav line.
func (p *FileView) VisibleRows() int { return max(p.h-3, 1) }

// LastLine is the index of the final line.
func (p *FileView) LastLine() int { return max(p.lineCount()-1, 0) }

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

// Focus focuses the editor when in edit mode (the read-only viewer needs no textarea focus).
//
// Retained for the model's focus plumbing.
func (p *FileView) Focus() tea.Cmd {
	if p.editing {
		return p.ta.Focus()
	}
	return nil
}

// Blur removes editor focus.
func (p *FileView) Blur() { p.ta.Blur() }

// Update forwards a message to the textarea (edit mode only; the read-only viewer is driven by the model's nav keys).
func (p *FileView) Update(msg tea.Msg) tea.Cmd {
	if !p.editing {
		return nil
	}
	var cmd tea.Cmd
	p.ta, cmd = p.ta.Update(msg)
	return cmd
}

// View renders the bordered panel: the textarea in edit mode, a line-numbered read-only viewer with the nav line
// highlighted otherwise.
//
// A "●" marks unsaved edits. focused drives the border/title brightness; navActive drives the nav-line highlight
// (true when focused OR mirroring as a follow follower).
func (p *FileView) View(focused, navActive bool) string {
	name := p.title
	if name == "" {
		name = "(no file)"
	}
	if p.Dirty() {
		name += " ●"
	}
	mode := "view"
	body := p.viewerBody(focused, navActive)
	if p.editing {
		mode = "edit"
		body = p.ta.View()
	}
	title := theme.Title(focused).Render("file · " + name + " · " + mode)
	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + body)
}

// viewerBody renders the read-only window: a right-aligned line-number gutter and the file text, with the nav line
// highlighted when navActive (the pane is focused, or mirroring the followed line as a follow follower).
//
// The driver pane keeps focus in follow mode, so focused doubles as "this is the driver line" — strong bar when it
// is, muted tint when this pane is only mirroring (§6.5).
func (p *FileView) viewerBody(focused, navActive bool) string {
	inner := max(p.w-2, 0)
	visible := max(p.h-3, 0)
	lines := strings.Split(p.ta.Value(), "\n")
	total := len(lines)
	numW, gutW := p.prefixWidth()
	textW := max(inner-(numW+1)-gutW, 0)

	style := navStyle(focused)

	var b strings.Builder
	end := min(p.offset+visible, total)
	for i := p.offset; i < end; i++ {
		prefix := p.gutterFor(i+1, gutW) + fmt.Sprintf("%*d ", numW, i+1)

		// The nav line answers a question the user asked, so it takes the whole row; syntax colour underneath it would fight
		// the bar's background and break it up with its own resets.
		// Same precedence as the spec pane.
		row := prefix + renderSpans(lines[i], p.spans[i], textW)
		if i == p.navLine && navActive {
			row = style.Render(prefix + fit(lines[i], textW))
		}
		b.WriteString(row)
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// prefixWidth is what the row prefix costs before a line's text: the line-number column, and the link gutter when
// there is anything to mark in it.
//
// Shared with LineColAt rather than recomputed there, because a hit test that disagreed with the renderer by one column
// would be wrong only at the edges of a token — the hardest kind of wrong to notice.
func (p *FileView) prefixWidth() (numW, gutW int) {
	numW = len(strconv.Itoa(len(strings.Split(p.ta.Value(), "\n"))))
	if len(p.anchors) > 0 {
		gutW = gutterWidth
	}

	return numW, gutW
}

// gutterFor renders the link marker for a 1-based source line, or blanks of the same width to keep the line numbers
// aligned.
//
// Empty when the gutter is off.
func (p *FileView) gutterFor(srcLine, width int) string {
	if width == 0 {
		return ""
	}
	if !p.anchors[srcLine] {
		return strings.Repeat(" ", width)
	}

	return theme.Gutter().Render(string(GutterAnchor)) + " "
}

// navStyle is the whole-line style for the highlighted nav line: the strong bar when this pane drives (the driver keeps
// focus in follow mode, design §6.1), a muted tint when it is only mirroring another pane's cursor (§6.5).
func navStyle(focused bool) lipgloss.Style {
	if focused {
		return theme.Selected()
	}
	return theme.Follower()
}

// gotoNav clamps and sets the nav line, then re-clamps the scroll offset.
func (p *FileView) gotoNav(line int) {
	p.navLine = min(max(line, 0), max(p.lineCount()-1, 0))
	p.clampOffset()
}

// syncEditorCursor steps the textarea cursor to navLine (textarea has no row-setter), so toggling into edit mode keeps
// the line.
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
