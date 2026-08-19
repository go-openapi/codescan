// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package help

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Overlay is the keymap modal.
//
// The zero value is a closed overlay; the root model opens it and composites its View over the base UI.
type Overlay struct {
	width, height int

	isOpen bool
	scroll int
}

// New builds a closed keymap overlay.
func New() Overlay {
	return Overlay{}
}

// helpEntry is one row of the help overlay: the key(s), and what they do.
type helpEntry struct {
	keys   string
	action string
}

// helpSection groups bindings by the context they apply in.
//
// The binding surface is context-dependent - f follows from three different panes, Enter opens a file in the tree
// but follows a $ref in the spec - so a flat list would be actively misleading.
type helpSection struct {
	title   string
	entries []helpEntry
}

// helpSections is the whole keymap, in the order the overlay shows it.
//
// It is the single source of truth for the overlay. The README table mirrors it by hand.
//
//nolint:gochecknoglobals // static content, read-only
var helpSections = []helpSection{
	{"anywhere", []helpEntry{
		{"h  ?", "this help"},
		{"tab  shift+tab", "cycle focus"},
		{"ctrl+← →", "move the divider between the left pane and the spec"},
		{"ctrl+↑ ↓", "move the divider above the diagnostics"},
		{"click", "focus the pane under the pointer, or open the annotation under it"},
		{"wheel", "move the cursor in the pane under the pointer"},
		{"c", "copy the focused pane to the clipboard"},
		{"r", "rescan now"},
		{"F5", "reload the open file from disk (asks before discarding edits)"},
		{"v", "validate the generated spec (go-openapi/validate)"},
		{"V", "switch the diagnostics pane between scan and validation"},
		{"o", "scanner options"},
		{"m", "what the last scan cost (time, memory, and the profiles under -profile)"},
		{"ctrl+q  ctrl+c", "quit"},
	}},
	{"spec pane", []helpEntry{
		{"↑ ↓  j k", "move the cursor"},
		{"pgup  pgdn", "move it a page"},
		{"home  end", "first / last line"},
		{"ctrl+j  ctrl+y", "render as JSON / YAML (keeps the node)"},
		{"/", "search"},
		{"n  N", "next / previous match"},
		{"f", "follow mode → source"},
		{"F3  shift+F3", "next / previous reference to this node"},
		{"enter", "go to the definition of the $ref here"},
		{"esc", "clear search and the reference cycle"},
	}},
	{"source tree", []helpEntry{
		{"↑ ↓  j k", "move the selection"},
		{"pgup  pgdn", "move it a page"},
		{"home  end", "first / last entry"},
		{"← →", "collapse / expand a directory"},
		{"enter", "open a file / expand a directory"},
		{"g", "locate this file's first node in the spec"},
	}},
	{"file viewer", []helpEntry{
		{"↑ ↓  j k", "move the navigation line"},
		{"pgup  pgdn", "move it a page"},
		{"home  end", "first / last line"},
		{"f", "follow mode → spec"},
		{"K", "what the swagger: annotation on this line means"},
		{"i  enter", "start editing"},
		{"esc", "back to the tree"},
	}},
	{"file editor", []helpEntry{
		{"ctrl+f", "jump to the spec node this line produced"},
		{"ctrl+s", "save (triggers a rescan)"},
		{"F5", "reload from disk, discarding edits"},
		{"esc", "back to the viewer"},
	}},
	{"confirmation popup", []helpEntry{
		{"y", "yes, go ahead"},
		{"n  esc  enter", "no, leave things as they are"},
	}},
	{"diagnostics", []helpEntry{
		{"↑ ↓  j k", "select a diagnostic"},
		{"pgup  pgdn", "select a page at a time"},
		{"home  end", "first / last diagnostic"},
		{"enter", "go to this diagnostic's source line"},
		{"f", "follow mode → source"},
	}},
	{"validation tab", []helpEntry{
		{"↑ ↓  j k", "select a finding"},
		{"pgup  pgdn", "select a page at a time"},
		{"home  end", "first / last finding"},
		{"enter", "go to this finding's node in the spec"},
		{"f", "follow mode → spec"},
	}},
	{"options popup", []helpEntry{
		{"↑ ↓  j k", "move"},
		{"pgup  pgdn  home  end", "move faster"},
		{"space", "toggle"},
		{"esc  o", "apply and close"},
	}},
}

// SetSize fits the overlay to outer dimensions w×h (border + title reserved).
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Open shows the keymap overlay, always from the top: it is opened to look something up, not resumed.
func (o *Overlay) Open() {
	o.isOpen = true
	o.scroll = 0
}

// Close hides the overlay.
func (o *Overlay) Close() {
	o.isOpen = false
	o.scroll = 0
}

// HandleKey drives the overlay: scroll, or dismiss.
//
// It swallows everything else - the overlay covers the UI, so acting on a key the user cannot see the effect of would
// be worse than ignoring it. Quitting is the one exception, and it belongs to the root model, which checks for it
// before the overlay ever sees the key.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if delta, ok := key.Nav(key.MsgBinding(msg), o.visibleRows(), len(helpLines())); ok {
		o.scrollBy(delta)

		return nil
	}

	switch key.MsgBinding(msg) {
	case key.Esc, key.H, key.Question, key.Enter:
		o.Close()
	}

	return nil
}

// The modal's fixed texts, named so the width can be measured against them as well as against the keymap.
const (
	helpTitle  = "Key bindings"
	helpFooter = "↑↓/jk: scroll · esc/h/?: close"
)

// View renders the help modal, with scrolling.
func (o *Overlay) View() string {
	lines := helpLines()
	visible := o.visibleRows()

	var b strings.Builder
	b.WriteString(theme.Accent().Render(helpTitle))
	b.WriteString("\n\n")

	top := 0
	if len(lines) > visible {
		top = min(max(o.scroll, 0), len(lines)-visible)
		lines = lines[top : top+visible]
	}
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render(o.footer(top)))

	return theme.ModalAt(o.contentWidth()).Render(b.String())
}

// contentWidth is the width to pin the frame at.
//
// It is the widest line the overlay can EVER show, not the widest one currently on screen.
//
// Sizing to the visible window makes the box change width as the keymap scrolls under it - the frame appears
// to twitch while the content moves, which reads as a rendering fault rather than as scrolling.
//
// Deliberately not capped to the terminal. A cap makes lipgloss WRAP the rows that overflow,
// which breaks the key/action columns apart; a modal wider than a narrow terminal is simply clipped,
// which is what it already did.
func (o *Overlay) contentWidth() int {
	return max(
		lipgloss.Width(strings.Join(helpLines(), "\n")),
		lipgloss.Width(helpTitle),
		// Measured at the LAST page, where the counter carries its largest numbers. Measuring the current page instead
		// would put a digit's worth of width back under the scroll - the very thing this frame is pinned to prevent.
		lipgloss.Width(o.footer(max(len(helpLines())-o.visibleRows(), 0))),
	)
}

// footer renders the bottom line for a window starting at line top.
//
// It states the range as well as the keys, because the keymap no longer fits a screen:
// a section below the fold is a binding nobody knows exists,
// and hiding one on the very screen whose job is discoverability is the worst place for it.
func (o *Overlay) footer(top int) string {
	total := len(helpLines())
	visible := o.visibleRows()
	if total <= visible {
		return helpFooter
	}

	return fmt.Sprintf("%d–%d of %d  ·  %s", top+1, min(top+visible, total), total, helpFooter)
}

// scrollBy moves the help window, clamped so it can never scroll past the end.
func (o *Overlay) scrollBy(delta int) {
	o.scroll = min(max(o.scroll+delta, 0), max(len(helpLines())-o.visibleRows(), 0))
}

// visibleRows is how many body rows fit between the modal's chrome.
func (o *Overlay) visibleRows() int {
	const chrome = 10 // border 2 + padding 2 + title 2 + footer 2, with slack

	return max(o.height-chrome, 3)
}

// helpLines renders the overlay body.
//
// A key column wide enough for every entry, then the action, with a blank line between sections.
func helpLines() []string {
	keyW := 0
	for _, sec := range helpSections {
		for _, e := range sec.entries {
			keyW = max(keyW, len([]rune(e.keys)))
		}
	}

	var lines []string
	for i, sec := range helpSections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, theme.Accent().Render(sec.title))
		for _, e := range sec.entries {
			pad := strings.Repeat(" ", keyW-len([]rune(e.keys)))
			lines = append(lines, "  "+e.keys+pad+"   "+theme.Status().Render(e.action))
		}
	}

	return lines
}
