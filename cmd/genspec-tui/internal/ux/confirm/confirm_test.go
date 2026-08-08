// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package confirm

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestOverlay_ZeroValueIsClosed(t *testing.T) {
	var o Overlay

	assert.False(t, o.IsOpen())

	_, ok := o.TakeAnswer()
	assert.False(t, ok, "an overlay nobody asked has no answer to give")
}

func TestOverlay_AnswerKeys(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tea.KeyMsg
		want bool
	}{
		{"y accepts", runes("y"), true},
		{"n declines", runes("n"), false},
		{"esc declines", tea.KeyMsg{Type: tea.KeyEsc}, false},
		// Enter usually means "take the default", and for a question guarding something irreversible the safe default is
		// no - so this is a decline, not an accept.
		{"enter declines", tea.KeyMsg{Type: tea.KeyEnter}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := New()
			o.Ask("Discard?")
			require.True(t, o.IsOpen())

			o.HandleKey(tc.key)

			assert.False(t, o.IsOpen(), "answering closes the overlay")
			accepted, ok := o.TakeAnswer()
			require.True(t, ok)
			assert.Equal(t, tc.want, accepted)
		})
	}
}

// TestOverlay_TakeAnswerIsConsuming pins that one "yes" cannot fire an action twice.
//
// The model calls TakeAnswer on every keypress that reaches an overlay, so an answer that stayed readable would re-run
// whatever it authorised for as long as nothing else overwrote it.
func TestOverlay_TakeAnswerIsConsuming(t *testing.T) {
	o := New()
	o.Ask("Discard?")
	o.HandleKey(runes("y"))

	accepted, ok := o.TakeAnswer()
	require.True(t, ok)
	require.True(t, accepted)

	_, ok = o.TakeAnswer()
	assert.False(t, ok, "the answer is collected exactly once")
}

// TestOverlay_UnknownKeysAreSwallowed pins that the overlay holds the question open.
//
// It must not act on a key whose effect the user cannot see behind the modal.
func TestOverlay_UnknownKeysAreSwallowed(t *testing.T) {
	o := New()
	o.Ask("Discard?")

	for _, k := range []tea.KeyMsg{runes("x"), runes("j"), {Type: tea.KeyTab}} {
		o.HandleKey(k)
		require.True(t, o.IsOpen(), "%s must not answer the question", k.String())
	}

	_, ok := o.TakeAnswer()
	assert.False(t, ok)
}

// TestOverlay_AskResetsAStaleAnswer pins that a new question never inherits the previous one's answer.
func TestOverlay_AskResetsAStaleAnswer(t *testing.T) {
	o := New()
	o.Ask("First?")
	o.HandleKey(runes("y")) // answered, but never collected

	o.Ask("Second?")

	require.True(t, o.IsOpen())
	_, ok := o.TakeAnswer()
	assert.False(t, ok, "the uncollected yes must not carry into the new question")
}

func TestOverlay_ViewShowsThePrompt(t *testing.T) {
	o := New()
	o.SetSize(80, 24)
	o.Ask("Discard unsaved edits to user.go?")

	view := o.View()

	assert.Contains(t, view, "Discard unsaved edits to user.go?")
	assert.True(t, strings.Contains(view, "y:") && strings.Contains(view, "esc"),
		"the modal states how to answer it: %q", view)
}
