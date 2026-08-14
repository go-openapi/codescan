// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runstats

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/humanize"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Overlay reports what the last scan cost.
//
// The zero value is a closed overlay holding no reading; the root model feeds it each result and composites its View
// over the base UI.
//
// It is a modal rather than another status-line field on purpose: the status line is already carrying the pane hints,
// the follow badge and the search prompt, and none of those can be given up for a figure most sessions never ask for.
type Overlay struct {
	width, height int

	isOpen bool

	cost   scan.Cost
	rescan bool
}

// New builds a closed overlay.
func New() Overlay {
	return Overlay{}
}

// Set records the cost of the run that has just landed.
//
// rescan says whether that run replaced a document the model was already holding, which the card has to disclose: at
// the closing fence the previous spec and the new one are both live, so the retained figure reads high by about one
// document. Neither the scan nor this overlay can see that on its own - only the model knows what it was holding.
func (o *Overlay) Set(c scan.Cost, rescan bool) {
	o.cost = c
	o.rescan = rescan
}

// SetSize fits the overlay to outer dimensions w×h.
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Open shows the card.
func (o *Overlay) Open() { o.isOpen = true }

// Close hides it.
func (o *Overlay) Close() { o.isOpen = false }

// HandleKey dismisses the overlay, and swallows everything else.
//
// The card is a single screen with nothing to navigate, so - unlike the keymap - there is nothing here for a movement
// key to do. Quitting belongs to the root model, which checks for it before the overlay ever sees the key.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	switch key.MsgBinding(msg) {
	case key.Esc, key.M, key.Enter:
		o.Close()
	}

	return nil
}

// The modal's fixed texts, named so the frame can be measured against them as well as against the figures.
const (
	title  = "Last run"
	footer = "esc / m: close"

	// What the reading is and is not. Stated on the card rather than in the README, because the numbers are read here
	// and a caveat nobody sees is a caveat that does not exist.
	noteWindow = "measured across the whole run: the redraw loop and the file watcher allocate on other goroutines too"
	noteRescan = "this run replaced a spec still held while it ran, so retained reads high by about one document"

	empty     = "no run measured yet"
	emptyHint = "the first scan is still running"
)

// The two columns the figures line up in: a label, then its value right-aligned so the units stack.
const (
	labelW = 13
	valueW = 11
)

// View renders the card.
func (o *Overlay) View() string {
	lines := o.lines()

	body := strings.Join(lines, "\n")

	return theme.ModalAt(o.contentWidth(lines)).Render(
		theme.Accent().Render(title) + "\n\n" +
			body + "\n\n" +
			theme.Status().Render(footer),
	)
}

// contentWidth pins the frame to the widest line it can show, so the box does not resize as the figures change.
func (o *Overlay) contentWidth(lines []string) int {
	return max(lipgloss.Width(strings.Join(lines, "\n")), lipgloss.Width(title), lipgloss.Width(footer))
}

// lines renders the card body: the figures, then what they do and do not mean.
func (o *Overlay) lines() []string {
	if !o.cost.Measured {
		return []string{theme.Status().Render(empty), "", theme.Status().Render(emptyHint)}
	}

	c := o.cost
	lines := []string{
		row("elapsed", humanize.Duration(c.ScanFor+c.RenderFor),
			"scanning "+humanize.Duration(c.ScanFor)+" · rendering "+humanize.Duration(c.RenderFor)),
		"",
		row("allocated", humanize.Bytes(c.Allocated()), "churned through, garbage included"),
		sub("scanning", humanize.Bytes(c.AllocScan)),
		sub("rendering", humanize.Bytes(c.AllocRender)),
		"",
		row("retained", humanize.SignedBytes(c.Retained()),
			humanize.Bytes(c.LiveBefore)+" → "+humanize.Bytes(c.LiveAfter)+" live"),
		sub("scanning", humanize.SignedBytes(c.RetainScan)),
		sub("rendering", humanize.SignedBytes(c.RetainRender)),
		"",
		row("objects", humanize.SignedCount(c.Objects), "live on the heap"),
		row("GC cycles", strconv.FormatUint(uint64(c.GCCycles), 10), ""),
		row("from the OS", humanize.Bytes(c.Sys), "high-water: the runtime rarely hands it back"),
		"",
		theme.Status().Render(noteWindow),
	}
	if o.rescan {
		lines = append(lines, theme.Status().Render(noteRescan))
	}

	return lines
}

// row renders one figure: label, value right-aligned in its column, and an optional gloss.
func row(label, value, note string) string {
	line := "  " + pad(label, labelW) + lead(value, valueW)
	if note != "" {
		line += "   " + theme.Status().Render(note)
	}

	return line
}

// sub renders a breakdown line under the figure it decomposes, indented so the two read as one block.
func sub(label, value string) string {
	return "    " + pad(theme.Status().Render(label), labelW-2) + theme.Status().Render(lead(value, valueW))
}

// pad left-aligns s in a column w wide, measuring what is drawn rather than the bytes (the labels may be styled).
func pad(s string, w int) string {
	return s + strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
}

// lead right-aligns s in a column w wide, which is what makes the units line up under one another.
func lead(s string, w int) string {
	return strings.Repeat(" ", max(w-lipgloss.Width(s), 0)) + s
}
