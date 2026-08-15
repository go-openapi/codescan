// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runstats

import (
	"fmt"
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
	scroll int

	cost    scan.Cost
	profile *scan.ProfileReport
	rescan  bool
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
func (o *Overlay) Set(c scan.Cost, report *scan.ProfileReport, rescan bool) {
	o.cost = c
	o.profile = report
	o.rescan = rescan
	o.scroll = 0
}

// SetSize fits the overlay to outer dimensions w×h.
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Open shows the card, always from the top: it is opened to read the last run, not resumed.
func (o *Overlay) Open() {
	o.isOpen = true
	o.scroll = 0
}

// Close hides it.
func (o *Overlay) Close() {
	o.isOpen = false
	o.scroll = 0
}

// HandleKey scrolls or dismisses the overlay, and swallows everything else.
//
// Unprofiled, the card fits a screen and the movement keys do nothing; a profiled run adds three tables, and a table
// below the fold is a measurement nobody sees. Quitting belongs to the root model, which checks for it before the
// overlay ever sees the key.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if delta, ok := key.Nav(key.MsgBinding(msg), o.visibleRows(), len(o.lines())); ok {
		o.scrollBy(delta)

		return nil
	}

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

	// A profiled run is measured while being watched: the sampler costs time, and the profiler collects at each fence
	// to keep its own figures current. The tables below are the accurate account; these figures are not.
	noteProfiled = "profiled run: these figures carry the profiler's overhead and its collections — read the tables below,\n" +
		"  which exclude the profiler's own work.\n" +
		"  CPU is charged to the call of ours that led there, so a row covers everything under it"

	// What the runtime's own row says. It is not a function to look up: it is the cost of the memory the table under
	// it accounts for, which is where a reader can do something about it.
	runtimeRow = "the runtime itself — collecting, allocating, scheduling"

	empty     = "no run measured yet"
	emptyHint = "the first scan is still running"
)

// The two columns the figures line up in: a label, then its value right-aligned so the units stack.
const (
	labelW = 13
	valueW = 11
)

// confidentSamples is the point below which the CPU table is a hint rather than a ranking.
//
// The sampler runs at 100 Hz, so this is a second of CPU: under it, the gap between the first row and the third is a
// couple of samples, and reordering them would take nothing.
const confidentSamples = 100

// View renders the card, with scrolling once it outgrows the terminal.
func (o *Overlay) View() string {
	all := o.lines()
	visible := o.visibleRows()

	shown := all
	top := 0
	if len(all) > visible {
		top = min(max(o.scroll, 0), len(all)-visible)
		shown = all[top : top+visible]
	}

	head := title
	if o.profile != nil {
		head += " · profiled"
	}

	return theme.ModalAt(o.contentWidth(all)).Render(
		theme.Accent().Render(head) + "\n\n" +
			strings.Join(shown, "\n") + "\n\n" +
			theme.Status().Render(o.footer(top, len(all))),
	)
}

// scrollBy moves the window, clamped so it can never scroll past the end.
func (o *Overlay) scrollBy(delta int) {
	o.scroll = min(max(o.scroll+delta, 0), max(len(o.lines())-o.visibleRows(), 0))
}

// visibleRows is how many body rows fit between the modal's chrome.
func (o *Overlay) visibleRows() int {
	const chrome = 10 // border 2 + padding 2 + title 2 + footer 2, with slack

	return max(o.height-chrome, 3)
}

// footer states the range as well as the keys: a profiled run does not fit a screen, and a table below the fold is a
// measurement nobody knows was taken.
func (o *Overlay) footer(top, total int) string {
	visible := o.visibleRows()
	if total <= visible {
		return footer
	}

	return fmt.Sprintf("%d–%d of %d  ·  ↑↓/jk: scroll · %s", top+1, min(top+visible, total), total, footer)
}

// contentWidth pins the frame to the widest line it can EVER show, not the widest one currently on screen - sizing to
// the window is what makes a box appear to twitch as its content scrolls under it.
func (o *Overlay) contentWidth(lines []string) int {
	return max(
		lipgloss.Width(strings.Join(lines, "\n")),
		lipgloss.Width(title+" · profiled"),
		lipgloss.Width(o.footer(0, len(lines))),
	)
}

// lines renders the card body: the figures, then what they do and do not mean.
func (o *Overlay) lines() []string {
	if !o.cost.Measured {
		return []string{theme.Status().Render(empty), "", theme.Status().Render(emptyHint)}
	}

	c := o.cost
	lines := []string{
		o.recap(),
		"",
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
	}

	// What these figures are not. Under a profiled run the first caveat is answered by the tables below - the noise is
	// named there - but two new ones apply, since observing the run costs the run something.
	if o.profile == nil {
		lines = append(lines, theme.Status().Render(noteWindow))
	} else {
		lines = append(lines, theme.Status().Render(noteProfiled))
	}
	if o.rescan {
		lines = append(lines, theme.Status().Render(noteRescan))
	}

	return append(lines, o.profileLines()...)
}

// recap is the one line a reader takes away: how the run divided between its two phases.
//
// The detail below answers it in absolute terms, but two figures in different units are a comparison the reader has to
// do in their head, and the answer they are usually after - is this a scanning problem or a rendering problem? - is a
// ratio. Time and memory can disagree sharply (a phase that churns while barely running, or the reverse), which is
// exactly why both are here rather than one standing in for the other.
//
// Under a profiled run the sampled CPU split joins them, and is the more honest of the two clocks: wall time counts
// waiting, CPU time counts working.
func (o *Overlay) recap() string {
	c := o.cost
	parts := []string{
		share("time", int64(c.ScanFor), int64(c.RenderFor)),
		share("memory", int64(c.AllocScan), int64(c.AllocRender)), //nolint:gosec // allocation figures never approach MaxInt64
	}

	if p := o.profile; p != nil && (p.Scan.Sampled() || p.Render.Sampled()) {
		parts = append(parts, share("cpu", int64(p.Scan.CPUTotal), int64(p.Render.CPUTotal)))
	}

	return "  " + pad("split", labelW) + strings.Join(parts, "   ·   ") +
		theme.Status().Render("      scanning / rendering")
}

// share renders one split as two percentages that add up to a hundred.
//
// The second is derived from the first rather than rounded on its own: independently rounded halves produce "96% / 5%",
// which reads as an arithmetic error in a line whose whole job is to be glanced at.
func share(label string, scanning, rendering int64) string {
	total := scanning + rendering
	if total <= 0 {
		return label + " n/a"
	}

	pct := (scanning*100 + total/2) / total

	return fmt.Sprintf("%s %d%% / %d%%", label, pct, 100-pct)
}

// profileLines renders what the profiler saw: for each phase, where the CPU went and what allocated the memory, then
// where the artifacts are.
//
// This is the account the figures above cannot give. A scalar cannot say who spent it, so the caveat about the redraw
// loop and the watcher can only be stated there; here they are simply rows, and a reader discounts them by name.
func (o *Overlay) profileLines() []string {
	p := o.profile
	if p == nil {
		return nil
	}

	lines := []string{""}
	for _, phase := range []scan.PhaseProfile{p.Scan, p.Render} {
		lines = append(lines, o.cpu(phase)...)
		lines = append(lines, o.sites(phase, p.Exact)...)
	}

	lines = append(lines, theme.Accent().Render("  artifacts"))
	for _, path := range append([]string{p.Scan.CPUPath, p.Render.CPUPath}, p.MemPaths...) {
		if path != "" {
			lines = append(lines, "    "+theme.Status().Render(path))
		}
	}
	if p.Scan.CPUPath != "" {
		lines = append(lines, "", "    "+theme.Status().Render("go tool pprof -http=: "+p.Scan.CPUPath))
	}
	if len(p.MemPaths) == scan.MemSnapshots {
		lines = append(lines, "    "+theme.Status().Render(
			"go tool pprof -http=: -base "+p.MemPaths[0]+" "+p.MemPaths[1]))
	}

	for _, note := range p.Notes {
		lines = append(lines, "", theme.Status().Render("  "+note))
	}

	return lines
}

// cpu renders one phase's CPU table, or says why there is none.
//
// A phase with no samples still gets its heading. Rendering is short enough that the sampler often catches nothing at
// all, and that is worth saying: an absent section reads as an oversight, while "nothing sampled" is a measurement.
func (o *Overlay) cpu(phase scan.PhaseProfile) []string {
	heading := theme.Accent().Render("  where the CPU went — " + phase.Label)

	switch {
	case phase.CPUPath == "":
		return []string{heading + theme.Status().Render("   not profiled"), ""}
	case !phase.Sampled():
		return []string{heading + theme.Status().Render(
			"   nothing sampled: the phase was shorter than the sampler's 10 ms interval"), ""}
	}

	samples := "samples"
	if phase.CPUSamples == 1 {
		samples = "sample"
	}
	lines := []string{heading + theme.Status().Render(fmt.Sprintf("   %s across %d %s, every goroutine included",
		humanize.Duration(phase.CPUTotal), phase.CPUSamples, samples))}

	// A percentage is only worth as much as the samples under it. At 100 Hz a sub-second phase gives a handful, and a
	// row drawn from two of them ranks nothing - so the card says which of the two tables this is.
	if phase.CPUSamples < confidentSamples {
		lines = append(lines, "    "+theme.Status().Render(
			"too few to rank functions: profile a larger tree, or read the artifact below"))
	}
	for _, f := range phase.CPU {
		lines = append(lines, "    "+lead(fmt.Sprintf("%.0f%%", f.Share*100), 5)+
			lead(humanize.Duration(f.Spent), 9)+"   "+charged(f))
	}

	return append(lines, "")
}

// charged names what a row's time was counted against.
//
// The pair is the actionable unit: the callee says what the time went into, and the frame of ours says where to go and
// change it. Naming only one of the two leaves the reader to find the other.
func charged(f scan.Func) string {
	switch {
	case f.Runtime:
		return theme.Status().Render(runtimeRow)
	case f.Callee != "":
		return shortFunc(f.Name) + theme.Status().Render(" → ") + shortFunc(f.Callee)
	default:
		return shortFunc(f.Name)
	}
}

// sites renders one phase's allocation table, or says why it is empty.
func (o *Overlay) sites(phase scan.PhaseProfile, exact bool) []string {
	estimate := "estimated from sampling"
	if exact {
		estimate = "every allocation counted"
	}

	lines := []string{theme.Accent().Render("  what allocated it — "+phase.Label) + theme.Status().Render("   "+estimate)}
	if len(phase.Alloc) == 0 {
		return append(lines, "    "+theme.Status().Render("nothing sampled in this phase"), "")
	}

	for _, s := range phase.Alloc {
		lines = append(lines, "    "+lead(humanize.Bytes(uint64(max(s.Bytes, 0))), 10)+
			lead(humanize.SignedCount(s.Objects), 9)+"   "+shortFunc(s.Name))
	}

	return append(lines, "")
}

// shortFunc keeps the last two path segments of a symbol, so a table of Go function names stays a table without
// becoming a table of ambiguities.
//
// "github.com/go-openapi/codescan/internal/builders/schema.(*Builder).Build" is eighty columns of mostly repository,
// and the modal is deliberately not wrapped. One segment is too few, though: our own package loader and the one it
// stands in for both end in "packages", and a run through each is exactly what a reader compares. Two segments tell
// "internal/packages" from "go/packages" - and carry a major version along with the library it belongs to, since
// "v3.is_space" names nothing while "yaml/v3.is_space" says which library is doing the work.
func shortFunc(name string) string {
	i := strings.LastIndex(name, "/")
	if i < 0 {
		return name
	}

	j := strings.LastIndex(name[:i], "/")
	if j < 0 {
		return name
	}

	return name[j+1:]
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
