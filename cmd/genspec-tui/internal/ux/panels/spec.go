// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// Spec is the right-hand generated-spec panel.
//
// It tracks the active render format (JSON or YAML) and an optional case-insensitive search
// that highlights matching lines and scrolls between them.
type Spec struct {
	vp      viewport.Model
	w, h    int
	format  string
	content string // raw, unhighlighted spec text (also what Content() copies)

	query    string
	matches  []int // indices of content lines containing the query
	matchIdx int

	cursor  int  // 0-based content line the user is on
	focused bool // last focus state seen by View; picks the cursor's style

	gutter map[int]rune         // content line → link marker; nil renders no gutter at all
	spans  map[int][]theme.Span // content line → lexical runs; nil renders plain
}

// Gutter markers: which lines actually lead somewhere.
const (
	// GutterAnchor marks a node with a source position of its OWN, so following it lands exactly there rather than on an
	// ancestor.
	GutterAnchor = '•'

	// GutterRef marks a followable $ref - Enter goes to its definition.
	GutterRef = '→'

	// gutterWidth is the marker plus its separating space.
	gutterWidth = 2
)

// NewSpec returns a Spec defaulting to JSON with placeholder content.
func NewSpec() Spec {
	const placeholder = "(no spec generated yet)"
	vp := viewport.New(0, 0)
	vp.SetContent(placeholder)
	return Spec{vp: vp, format: "JSON", content: placeholder}
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

// SetContent replaces the raw spec text, re-applying any active search.
//
// The cursor is CLAMPED, not reset: a rescan usually re-renders nearly the same document, and dropping the user back to
// line 0 on every save would make the live-reload loop unusable.
// Restoring it to the same NODE is the caller's job (see Model.refreshSpec).
func (p *Spec) SetContent(s string) {
	p.content = s
	p.cursor = min(max(p.cursor, 0), max(p.lineCount()-1, 0))
	p.render()
	p.revealCursor()
}

// Content returns the raw (unhighlighted) panel text, for clipboard copy.
func (p *Spec) Content() string { return p.content }

// Search sets the query, highlights matching lines, moves the cursor to the first match, and returns the match count.
//
// Putting the CURSOR on the match (rather than merely scrolling to it) means every cursor-driven action - follow,
// find-references, go-to-definition - acts on what you just searched for.
func (p *Spec) Search(query string) int {
	p.query = query
	p.matchIdx = 0
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

// scrollContext is how many lines of context to keep above a scrolled-to match.
const scrollContext = 2

// CursorLine returns the 0-based content line the user is on.
//
// This is what every "the node under the cursor" question resolves against.
func (p *Spec) CursorLine() int { return p.cursor }

// TopLine returns the 0-based index of the top visible content line.
func (p *Spec) TopLine() int { return p.vp.YOffset }

// LastLine is the index of the final content line.
func (p *Spec) LastLine() int { return max(p.lineCount()-1, 0) }

// SetCursor parks the cursor on the 0-based line, scrolling only as far as needed to keep it visible.
//
// The incremental primitive.
func (p *Spec) SetCursor(line int) {
	p.moveCursorTo(line)
	p.revealCursor()
}

// MoveCursor steps the cursor by delta, scrolling minimally.
//
// Used by the nav keys and the wheel, where a lurching viewport would be miserable.
func (p *Spec) MoveCursor(delta int) { p.SetCursor(p.cursor + delta) }

// JumpTo parks the cursor on the line and scrolls it to the VERTICAL CENTRE, clamped at the edges.
//
// The JUMP primitive: every cross-ref landing comes through here - follow-mode mirroring, g locate, the ctrl+f
// jump, F3 cycling, go-to-definition.
//
// Centring rather than the top-biased scroll search uses: in follow mode the target moves continuously, so a top bias
// pins it against whichever edge it entered from and makes it jitter, instead of letting it sit still while its
// surroundings slide past.
// For a one-shot jump it simply shows context on both sides of the destination.
func (p *Spec) JumpTo(line int) {
	p.moveCursorTo(line)
	p.vp.SetYOffset(max(p.cursor-p.vp.Height/2, 0))
}

// Update forwards a message to the underlying viewport (scrolling).
func (p *Spec) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return cmd
}

// View renders the bordered panel; focused brightens the border and title.
//
// Focus also decides how the cross-ref line is painted: the driver pane keeps focus in follow mode, so
// "focused" and "is the driver" are the same bit.
// Re-render only on a focus TRANSITION - the spec can be thousands of lines and View runs on every message.
func (p *Spec) View(focused bool) string {
	if p.focused != focused {
		p.focused = focused
		p.render()
	}
	title := theme.Title(focused).Render("spec · " + p.format)
	return theme.Panel(p.w, p.h, focused).Render(title + "\n" + p.vp.View())
}

// SetSpans installs the per-line lexical runs used for syntax highlighting.
//
// A nil map renders the spec plain, which is what happens before the first scan.
func (p *Spec) SetSpans(spans map[int][]theme.Span) {
	p.spans = spans
	p.render()
}

// SetGutter installs the link markers, keyed by content line.
//
// A nil or empty map renders no gutter at all, so the pane costs no width until there is something to say (before the
// first scan, or with provenance switched off).
func (p *Spec) SetGutter(g map[int]rune) {
	p.gutter = g
	p.render()
}

// moveCursorTo clamps and sets the cursor, re-rendering only when it actually moved.
//
// The cursor style is baked into the viewport content, so a move that changes nothing must not rebuild it.
func (p *Spec) moveCursorTo(line int) {
	line = min(max(line, 0), max(p.lineCount()-1, 0))
	if line == p.cursor {
		return
	}
	p.cursor = line
	p.render()
}

// revealCursor scrolls the minimum distance that brings the cursor into view.
func (p *Spec) revealCursor() {
	switch {
	case p.cursor < p.vp.YOffset:
		p.vp.SetYOffset(p.cursor)
	case p.cursor >= p.vp.YOffset+p.vp.Height:
		p.vp.SetYOffset(p.cursor - p.vp.Height + 1)
	}
}

func (p *Spec) lineCount() int { return strings.Count(p.content, "\n") + 1 }

// cursorStyle is the whole-line style for the cursor.
//
// The strong bar when this pane drives, a muted tint when it is mirroring another pane.
func (p *Spec) cursorStyle() lipgloss.Style {
	if p.focused {
		return theme.Selected()
	}
	return theme.Follower()
}

// render rebuilds the viewport content from the raw text.
//
// It applies the active search highlight per substring, the cross-ref highlight over the whole line,
// and the link gutter.
//
// The cross-ref line takes the whole-line style; search matches are still counted on it so n/N stays consistent.
//
// The gutter is prefixed AFTER highlighting, so the styles apply to the text the user searched for, not to the marker
// column.
func (p *Spec) render() {
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

		// Precedence: cursor, then search, then syntax.
		// The first two are answers to "where am I" and "what did I ask for" - questions the user posed - so they take
		// the whole line rather than compete with colour for it.
		// One plain line reads fine; a line wearing three styles does not.
		switch {
		case i == p.cursor:
			ln = p.cursorStyle().Render(ln)
		case isMatch:
			ln = highlightAll(ln, p.query)
		case len(p.spans) > 0:
			ln = renderSpans(ln, p.spans[i], len([]rune(ln)))
		}
		lines[i] = p.gutterFor(i) + ln
	}
	p.vp.SetContent(strings.Join(lines, "\n"))
}

// gutterFor renders line i's marker column, or blanks of the same width so the text stays aligned.
//
// Empty string when no gutter is installed.
func (p *Spec) gutterFor(i int) string {
	if len(p.gutter) == 0 {
		return ""
	}
	marker, ok := p.gutter[i]
	if !ok {
		return strings.Repeat(" ", gutterWidth)
	}

	return theme.Gutter().Render(string(marker)) + " "
}

func (p *Spec) scrollToMatch() {
	if len(p.matches) == 0 {
		return
	}
	p.moveCursorTo(p.matches[p.matchIdx])
	// keep a little context above the match, rather than centring: when stepping matches you want to see the ones that
	// follow.
	p.vp.SetYOffset(max(p.cursor-scrollContext, 0))
}

// highlightAll wraps every case-insensitive occurrence of query in line with the match style.
//
// The original casing of the matched text is preserved.
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
