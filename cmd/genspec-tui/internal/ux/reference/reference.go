// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reference

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/key"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// wrapW is the width the popup lays its prose out at.
//
// Fixed rather than sized to the content: the Keywords field is a sentence or two, and a modal as wide as its longest
// unbroken line would stretch most of the way across a wide terminal to hold one paragraph. Wrapping is wanted here,
// which is what makes this the opposite case from the scrolling overlays.
const wrapW = 72

// Overlay is the annotation-reference modal.
//
// The zero value is a closed overlay; the root model opens it with Show.
type Overlay struct {
	width, height int

	isOpen bool
	name   string
	doc    grammar.AnnotationDoc
}

// New builds a closed reference overlay.
func New() Overlay {
	return Overlay{}
}

// SetSize fits the overlay to outer dimensions w×h.
func (o *Overlay) SetSize(w, h int) {
	o.width = w
	o.height = h
}

// IsOpen reports whether the overlay is currently covering the UI.
func (o *Overlay) IsOpen() bool { return o.isOpen }

// Show opens the popup on one annotation's entry.
func (o *Overlay) Show(name string, doc grammar.AnnotationDoc) {
	o.isOpen = true
	o.name = name
	o.doc = doc
}

// Close hides the overlay.
func (o *Overlay) Close() { o.isOpen = false }

// HandleKey dismisses the popup.
//
// There is nothing to navigate - it is a few lines of reference text - so every key that plausibly means "done" closes
// it, including the key that opened it. Everything else is swallowed, as with the other overlays: acting on a key whose
// effect is hidden behind the modal is worse than ignoring it.
func (o *Overlay) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "K" {
		o.Close()

		return nil
	}

	switch key.MsgBinding(msg) {
	case key.Esc, key.Enter, key.Question, key.H:
		o.Close()
	}

	return nil
}

// View renders the reference modal.
func (o *Overlay) View() string {
	var b strings.Builder
	b.WriteString(theme.Accent().Render(grammar.AnnotationPrefix + o.name))
	b.WriteString("\n\n")
	b.WriteString(o.doc.Summary)
	// The usage line carries the syntax, so it is worth a line of its own only when it says more than the heading
	// already did. Four annotations take no argument at all, and for those the two lines are the same token in two
	// colours - which reads as a rendering fault rather than as syntax.
	if o.doc.Usage != grammar.AnnotationPrefix+o.name {
		b.WriteString("\n\n")
		b.WriteString(theme.Syntax(theme.SyntaxKey).Render(o.doc.Usage))
	}
	if o.doc.Keywords != "" {
		b.WriteString("\n\n")
		b.WriteString(theme.Status().Render(o.doc.Keywords))
	}
	b.WriteString("\n\n")
	b.WriteString(theme.Status().Render("esc: close"))

	return theme.ModalAt(o.contentWidth()).Render(b.String())
}

// contentWidth is the prose width, narrowed when the terminal cannot hold it.
func (o *Overlay) contentWidth() int {
	const minW = 24

	return min(wrapW, max(o.width-theme.ModalChromeW, minW))
}
