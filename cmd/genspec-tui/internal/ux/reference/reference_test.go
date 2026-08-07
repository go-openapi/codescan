// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package reference

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func sample() grammar.AnnotationDoc {
	return grammar.AnnotationDoc{
		Usage:    "swagger:model [NAME]",
		Summary:  "Publishes a Go type as a definitions entry.",
		Keywords: "required, readOnly, discriminator and the validations.",
	}
}

func shown(t *testing.T) *Overlay {
	t.Helper()

	o := New()
	o.SetSize(120, 40)
	o.Show("model", sample())
	require.True(t, o.IsOpen())

	return &o
}

func TestOverlay_ZeroValueIsClosed(t *testing.T) {
	var o Overlay

	assert.False(t, o.IsOpen())
}

func TestOverlay_ViewShowsEveryField(t *testing.T) {
	o := shown(t)
	doc := sample()

	view := o.View()

	for _, want := range []string{"swagger:model", doc.Summary, doc.Usage, doc.Keywords} {
		assert.Contains(t, view, want)
	}
}

// TestOverlay_OmitsAnEmptyKeywordsBlock keeps the popup from showing a blank gap where an annotation has no body.
func TestOverlay_OmitsAnEmptyKeywordsBlock(t *testing.T) {
	o := New()
	o.SetSize(120, 40)
	o.Show("ignore", grammar.AnnotationDoc{Usage: "swagger:ignore", Summary: "Excludes the declaration."})

	body := o.View()

	assert.NotContains(t, body, "\n\n\n", "an absent keywords block must not leave a double gap")
}

// TestOverlay_OmitsAUsageLineThatOnlyRepeatsTheHeading covers the four annotations that take no argument.
//
// Their usage form IS the bare annotation name, so a usage line would print the heading's own token a second time in a
// different colour - which reads as a rendering fault rather than as syntax.
func TestOverlay_OmitsAUsageLineThatOnlyRepeatsTheHeading(t *testing.T) {
	o := New()
	o.SetSize(120, 40)
	o.Show("meta", grammar.AnnotationDoc{
		Usage:    "swagger:meta",
		Summary:  "Declares the package as the top-level OpenAPI spec container.",
		Keywords: "Body: Version, Host, BasePath.",
	})

	body := o.View()

	assert.Equal(t, 1, strings.Count(body, "swagger:meta"),
		"the heading already said it; the usage line adds nothing")
	assert.Contains(t, body, "Body: Version", "the keywords brief still shows")
}

func TestOverlay_DismissKeys(t *testing.T) {
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'K'}},
	} {
		o := shown(t)

		require.Nil(t, o.HandleKey(msg))

		assert.False(t, o.IsOpen(), "%s must dismiss the popup", msg.String())
	}
}

// TestOverlay_SwallowsOtherKeys pins that the popup stays put rather than acting on a key whose effect is hidden behind
// it, matching the other overlays.
func TestOverlay_SwallowsOtherKeys(t *testing.T) {
	o := shown(t)

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'k'}}, // lowercase: the nav key, NOT the dismiss key
		{Type: tea.KeyTab},
	} {
		require.Nil(t, o.HandleKey(msg))
		assert.True(t, o.IsOpen(), "%s must not dismiss the popup", msg.String())
	}
}

// TestOverlay_WrapsRatherThanStretching is why this overlay pins a width at all: Keywords is a sentence or two, and a
// modal sized to hold it unbroken would run most of the way across a wide terminal.
func TestOverlay_WrapsRatherThanStretching(t *testing.T) {
	o := New()
	o.SetSize(300, 40)
	o.Show("model", grammar.AnnotationDoc{
		Usage:    "swagger:model [NAME]",
		Summary:  "Publishes a Go type as a definitions entry.",
		Keywords: strings.Repeat("a long sentence about keywords. ", 8),
	})

	assert.LessOrEqual(t, lipgloss.Width(o.View()), wrapW+theme.ModalChromeW,
		"the popup must wrap its prose, not stretch to hold it")
}

// TestOverlay_NarrowsForASmallTerminal pins the other end: the prose width gives way rather than the box overflowing.
func TestOverlay_NarrowsForASmallTerminal(t *testing.T) {
	o := New()
	o.SetSize(40, 20)
	o.Show("model", sample())

	assert.LessOrEqual(t, lipgloss.Width(o.View()), 40)
}
