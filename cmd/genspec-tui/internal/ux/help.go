// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"strings"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// helpEntry is one row of the help overlay: the key(s), and what they do.
type helpEntry struct {
	keys   string
	action string
}

// helpSection groups bindings by the context they apply in. The binding surface
// is context-dependent — `f` follows from three different panes, `Enter` opens a
// file in the tree but follows a $ref in the spec — so a flat list would be
// actively misleading.
type helpSection struct {
	title   string
	entries []helpEntry
}

// helpSections is the whole keymap, in the order the overlay shows it. It is the
// single source of truth for the overlay; the README table mirrors it by hand.
//
//nolint:gochecknoglobals // static content, read-only
var helpSections = []helpSection{
	{"anywhere", []helpEntry{
		{"h  ?", "this help"},
		{"tab  shift+tab", "cycle focus"},
		{"click", "focus the pane under the pointer"},
		{"wheel", "move the cursor in the pane under the pointer"},
		{"c", "copy the focused pane to the clipboard"},
		{"r", "rescan now"},
		{"o", "scanner options"},
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
		{"i  enter", "start editing"},
		{"esc", "back to the tree"},
	}},
	{"file editor", []helpEntry{
		{"ctrl+f", "jump to the spec node this line produced"},
		{"ctrl+s", "save (triggers a rescan)"},
		{"esc", "back to the viewer"},
	}},
	{"diagnostics", []helpEntry{
		{"↑ ↓  j k", "select a diagnostic"},
		{"pgup  pgdn", "select a page at a time"},
		{"home  end", "first / last diagnostic"},
		{"enter", "go to this diagnostic's source line"},
		{"f", "follow mode → source"},
	}},
	{"options popup", []helpEntry{
		{"↑ ↓  j k", "move"},
		{"pgup  pgdn  home  end", "move faster"},
		{"space", "toggle"},
		{"esc  o", "apply and close"},
	}},
}

// helpLines renders the overlay body: a key column wide enough for every entry,
// then the action, with a blank line between sections.
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

// helpVisibleRows is how many body rows fit between the modal's chrome.
func (m *Model) helpVisibleRows() int {
	const chrome = 10 // border 2 + padding 2 + title 2 + footer 2, with slack

	return max(m.height-chrome, 3)
}

// helpView renders the help modal, scrolled to m.helpScroll.
func (m *Model) helpView() string {
	lines := helpLines()
	visible := m.helpVisibleRows()

	var b strings.Builder
	b.WriteString(theme.Accent().Render("Key bindings"))
	b.WriteString("\n\n")

	if len(lines) > visible {
		top := clampInt(m.helpScroll, 0, len(lines)-visible)
		lines = lines[top : top+visible]
	}
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render("↑↓/jk: scroll · esc/h/?: close"))

	return theme.Modal().Render(b.String())
}

// scrollHelp moves the help window, clamped so it can never scroll past the end.
func (m *Model) scrollHelp(delta int) {
	m.helpScroll = clampInt(m.helpScroll+delta, 0, max(len(helpLines())-m.helpVisibleRows(), 0))
}
