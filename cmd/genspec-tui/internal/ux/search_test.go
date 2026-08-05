// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The suite knew how to OPEN the search prompt and never typed into it, so everything past that —
// the keystrokes reaching the input, running the query, stepping the matches, giving the keyboard
// back — went untested.

// typeQuery sends each rune of q through the dispatch, as a user would.
func typeQuery(t *testing.T, m *Model, q string) {
	t.Helper()
	for _, r := range q {
		_ = m.handleKey(testutils.KeyRune(r))
	}
}

func searchModel(t *testing.T) *Model {
	t.Helper()

	return testModel(t, sized(100, 40), focusedOn(paneSpec), withSpecJSON(refSpecJSON))
}

func TestSearch_RunsTheQuery(t *testing.T) {
	m := searchModel(t)

	_ = m.handleKey(testutils.KeyRune('/'))
	require.True(t, m.search.Active(), "`/` opens the prompt")

	typeQuery(t, m, "definitions")
	assert.Equal(t, "definitions", m.search.Query(),
		"keystrokes reach the input rather than the panes behind it")

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, m.search.Active(), "running the query gives the keyboard back")
	cur, total := m.spec.MatchInfo()
	assert.Positive(t, total, "the spec pane holds the matches")
	assert.Equal(t, 1, cur, "and parks on the first one")
}

// n / N are case-sensitive, so they are routed before the binding table lowercases them.
func TestSearch_StepsThroughMatches(t *testing.T) {
	m := searchModel(t)
	_ = m.handleKey(testutils.KeyRune('/'))
	typeQuery(t, m, "definitions")
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	_, total := m.spec.MatchInfo()
	require.Greater(t, total, 1, "precondition: more than one match to step through")

	_ = m.handleKey(testutils.KeyRune('n'))
	cur, _ := m.spec.MatchInfo()
	assert.Equal(t, 2, cur)

	_ = m.handleKey(testutils.KeyRune('N'))
	cur, _ = m.spec.MatchInfo()
	assert.Equal(t, 1, cur, "N steps back")
}

func TestSearch_NoMatchesSaysSo(t *testing.T) {
	m := searchModel(t)
	_ = m.handleKey(testutils.KeyRune('/'))
	typeQuery(t, m, "zzz")

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.NotNil(t, cmd, "the notice is scheduled to expire")
	assert.Contains(t, m.notice, "no matches: zzz")
	assert.False(t, m.search.Active())
}

func TestSearch_EscapeCancels(t *testing.T) {
	m := searchModel(t)
	_ = m.handleKey(testutils.KeyRune('/'))
	typeQuery(t, m, "definitions")
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotZero(t, matchTotal(m))

	_ = m.handleKey(testutils.KeyRune('/'))
	typeQuery(t, m, "Team")
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, m.search.Active())
	assert.Zero(t, matchTotal(m), "cancelling drops the previous highlights too")
}

// An empty query is a way to clear the highlights without a notice.
func TestSearch_EmptyQueryClears(t *testing.T) {
	m := searchModel(t)
	_ = m.handleKey(testutils.KeyRune('/'))
	typeQuery(t, m, "definitions")
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotZero(t, matchTotal(m))

	_ = m.handleKey(testutils.KeyRune('/'))
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Nil(t, cmd, "nothing to report")
	assert.Empty(t, m.notice)
	assert.Zero(t, matchTotal(m))
}

// While the prompt holds the keyboard, keys that would otherwise act on the panes must reach the
// input instead — otherwise a query containing `r` would trigger a rescan.
func TestSearch_SwallowsPaneKeys(t *testing.T) {
	m := searchModel(t)
	_ = m.handleKey(testutils.KeyRune('/'))

	typeQuery(t, m, "ro")

	assert.Equal(t, "ro", m.search.Query())
	assert.False(t, m.scan.Running, "r did not start a scan")
	assert.False(t, m.options.IsOpen(), "o did not open the options")
	assert.True(t, m.search.Active())
}

// Opening the prompt is a change of view, so it drops the follow mode and the reference cycle that
// were describing the old one.
func TestSearch_OpeningDropsFollowAndRefs(t *testing.T) {
	m := searchModel(t)
	m.follow = followSpec
	m.refs.Status = "ref 1/3 of /definitions/User"

	_ = m.handleKey(testutils.KeyRune('/'))

	assert.Equal(t, followOff, m.follow)
	assert.Empty(t, m.refs.Status)
	assert.Equal(t, paneSpec, m.focused, "the search acts on the spec pane, so it takes focus")
}

func matchTotal(m *Model) int {
	_, total := m.spec.MatchInfo()

	return total
}
