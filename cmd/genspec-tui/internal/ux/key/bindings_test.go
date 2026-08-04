// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package key

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-openapi/testify/v2/assert"
)

func TestNav_Movements(t *testing.T) {
	const page, span = 10, 500

	for _, tc := range []struct {
		binding Binding
		want    int
	}{
		{Up, -1},
		{K, -1},
		{Down, +1},
		{J, +1},
		{PgUp, -page},
		{PgDown, +page},
		{Home, -span},
		{End, +span},
	} {
		delta, ok := Nav(tc.binding, page, span)

		assert.True(t, ok, "%q is a movement", tc.binding)
		assert.Equal(t, tc.want, delta, "%q", tc.binding)
	}
}

// Home and End are deltas rather than absolute positions: every caller clamps, so ∓span always lands on the ends
// wherever the cursor started. That is what lets one rule serve a scroll offset, a list index and a line cursor.
func TestNav_HomeAndEndReachTheEndsFromAnywhere(t *testing.T) {
	const span = 40
	clamp := func(v int) int { return min(max(v, 0), span) }

	for _, from := range []int{0, 1, span / 2, span - 1, span} {
		home, _ := Nav(Home, 5, span)
		end, _ := Nav(End, 5, span)

		assert.Zero(t, clamp(from+home), "home from %d", from)
		assert.Equal(t, span, clamp(from+end), "end from %d", from)
	}
}

// A pane must be able to tell a movement from a key it owns, or it would swallow the lot.
func TestNav_IgnoresNonMovements(t *testing.T) {
	for _, b := range []Binding{Enter, Esc, Space, F, I, O, R, Tab, F3, CtrlQ, Left, Right} {
		delta, ok := Nav(b, 10, 100)

		assert.False(t, ok, "%q is not a movement", b)
		assert.Zero(t, delta)
	}
}

func TestBinding_Quit(t *testing.T) {
	assert.True(t, MsgBinding(tea.KeyMsg{Type: tea.KeyCtrlQ}).Quit())
	assert.True(t, MsgBinding(tea.KeyMsg{Type: tea.KeyCtrlC}).Quit())
	assert.False(t, MsgBinding(tea.KeyMsg{Type: tea.KeyEsc}).Quit())
	assert.False(t, MsgBinding(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}).Quit())
}

// Shift+F3 arrives as F15 from the xterm family, and only as "shift+f3" from terminals that report the modifier.
func TestMsgBinding_NormalizesCase(t *testing.T) {
	assert.Equal(t, K, MsgBinding(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}))
	assert.Equal(t, Binding("n"), MsgBinding(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}}),
		"MsgBinding lowercases, which is why case-sensitive keys are routed before it")
}
