// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Spec is the right-hand generated-spec panel. It tracks the active render
// format (JSON/YAML) and an optional case-insensitive search that highlights
// matching lines and scrolls between them.
type Spec struct {
	vp      viewport.Model
	w, h    int
	format  string
	content string // raw, unhighlighted spec text (also what Content() copies)

	query    string
	matches  []int // indices of content lines containing the query
	matchIdx int

	xrefLine int // content line highlighted by cross-ref follow, -1 for none
}

// NewSpec returns a Spec defaulting to JSON with placeholder content.
func NewSpec() Spec {
	const placeholder = "(no spec generated yet)"
	vp := viewport.New(0, 0)
	vp.SetContent(placeholder)
	return Spec{vp: vp, format: "JSON", content: placeholder, xrefLine: -1}
}

// SetSize fits the panel to outer dimensions w×h (border + title reserved).
func (p *Spec) SetSize(w, h int) {
	p.w, p.h = w, h
	p.vp.Width = max(w-2, 0)
	p.vp.Height = max(h-3, 0)
}

// SetFormat sets the title's format label ("JSON" or "YAML").
func (p *Spec) SetFormat(f string) { p.format = f }

// Format returns the active render format label.
func (p *Spec) Format() string { return p.format }

// SetContent replaces the raw spec text, re-applying any active search. A new
// spec invalidates any cross-ref highlight (the line may have moved).
func (p *Spec) SetContent(s string) {
	p.content = s
	p.xrefLine = -1
	p.render()
}

// Content returns the raw (unhighlighted) panel text, for clipboard copy.
func (p *Spec) Content() string { return p.content }

// Search sets the query, highlights matching lines, scrolls to the first
// match, and returns the match count. A search supersedes any cross-ref
// highlight.
func (p *Spec) Search(query string) int {
	p.query = query
	p.matchIdx = 0
	p.xrefLine = -1
	p.render()
	if len(p.matches) > 0 {
		p.scrollToMatch()
	}
	return len(p.matches)
}

// Step moves to the next (dir +1) or previous (dir -1) match, wrapping around.
func (p *Spec) Step(dir int) {
	if len(p.matches) == 0 {
		return
	}
	p.matchIdx = (p.matchIdx + dir + len(p.matches)) % len(p.matches)
	p.scrollToMatch()
}

// ClearSearch drops the query and re-renders the plain spec.
func (p *Spec) ClearSearch() {
	p.query = ""
	p.matches = nil
	p.matchIdx = 0
	p.render()
}

// MatchInfo returns the 1-based current match and the total (0,0 when none).
func (p *Spec) MatchInfo() (cur, total int) {
	if len(p.matches) == 0 {
		return 0, 0
	}
	return p.matchIdx + 1, len(p.matches)
}

// TopLine returns the 0-based index of the top visible content line, for
// mapping the current scroll position to a spec node via the SpecIndex.
func (p *Spec) TopLine() int { return p.vp.YOffset }

// scrollContext is how many lines of context to keep above a scrolled-to target.
const scrollContext = 2

// MarkLine marks the 0-based content line as the cross-ref target (same style
// the source pane uses for its nav line) WITHOUT scrolling. Used when the spec
// is the driver pane (the user controls its scroll) and only the node mapping
// should be shown.
func (p *Spec) MarkLine(line int) {
	if p.xrefLine == line {
		return // already marked; avoid a re-render on every driver scroll
	}
	p.xrefLine = line
	p.render()
}

// HighlightLine marks the line and scrolls it into view with a little context
// above. Used when the spec is the follower (source→spec navigation).
func (p *Spec) HighlightLine(line int) {
	p.vp.SetYOffset(max(line-scrollContext, 0))
	p.MarkLine(line)
}

// ClearHighlight drops the cross-ref highlight, if any.
func (p *Spec) ClearHighlight() {
	if p.xrefLine == -1 {
		return
	}
	p.xrefLine = -1
	p.render()
}

// Update forwards a message to the underlying viewport (scrolling).
func (p *Spec) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return cmd
}

// View renders the bordered panel; focused brightens the border/title.
func (p *Spec) View(focused bool) string {
	title := theme.Title(focused).Render("spec · " + p.format)
	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + p.vp.View())
}

// render rebuilds the viewport content from the raw text, applying the active
// search highlight (per-substring) and the cross-ref highlight (whole line).
// The cross-ref line takes the whole-line style; search matches are still
// counted on it so n/N stays consistent.
func (p *Spec) render() {
	if p.query == "" && p.xrefLine == -1 {
		p.matches = nil
		p.vp.SetContent(p.content)
		return
	}

	needle := ""
	if p.query != "" {
		needle = strings.ToLower(p.query)
	}
	lines := strings.Split(p.content, "\n")
	p.matches = p.matches[:0]
	for i, ln := range lines {
		isMatch := needle != "" && strings.Contains(strings.ToLower(ln), needle)
		if isMatch {
			p.matches = append(p.matches, i)
		}
		switch {
		case i == p.xrefLine:
			lines[i] = theme.Selected().Render(ln)
		case isMatch:
			lines[i] = highlightAll(ln, p.query)
		}
	}
	p.vp.SetContent(strings.Join(lines, "\n"))
}

func (p *Spec) scrollToMatch() {
	if len(p.matches) == 0 {
		return
	}
	// keep a little context above the match
	p.vp.SetYOffset(max(p.matches[p.matchIdx]-2, 0))
}

// highlightAll wraps every case-insensitive occurrence of query in line with
// the match style, preserving the original casing of the matched text.
func highlightAll(line, query string) string {
	if query == "" {
		return line
	}
	style := theme.Match()
	lower := strings.ToLower(line)
	lq := strings.ToLower(query)

	var b strings.Builder
	for {
		i := strings.Index(lower, lq)
		if i < 0 {
			b.WriteString(line)
			break
		}
		b.WriteString(line[:i])
		b.WriteString(style.Render(line[i : i+len(query)]))
		line = line[i+len(query):]
		lower = lower[i+len(query):]
	}
	return b.String()
}
