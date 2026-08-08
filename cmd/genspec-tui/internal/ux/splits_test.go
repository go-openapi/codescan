// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// ctrlArrow spells a split key the way a terminal sends it.
func ctrlArrow(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

// TestSplit_KeysMoveTheDividers pins the direction of travel: each key moves its divider the way its arrow points.
//
// Getting this backwards is the kind of thing that reads fine in the code and is immediately wrong in the hand, so the
// assertion is on the resulting geometry rather than on the percentage.
func TestSplit_KeysMoveTheDividers(t *testing.T) {
	t.Run("ctrl+right widens the left pane", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		before := m.leftW

		_ = m.handleKey(ctrlArrow(tea.KeyCtrlRight))

		assert.Greater(t, m.leftW, before)
	})

	t.Run("ctrl+left narrows it", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		before := m.leftW

		_ = m.handleKey(ctrlArrow(tea.KeyCtrlLeft))

		assert.Less(t, m.leftW, before)
	})

	t.Run("ctrl+up grows the diagnostics strip", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		before := m.diagH

		_ = m.handleKey(ctrlArrow(tea.KeyCtrlUp))

		assert.Greater(t, m.diagH, before)
	})

	t.Run("ctrl+down shrinks it", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		before := m.diagH

		_ = m.handleKey(ctrlArrow(tea.KeyCtrlDown))

		assert.Less(t, m.diagH, before)
	})
}

// TestSplit_PanesShareTheSpace pins that moving a divider REALLOCATES rather than resizing one side into the void.
func TestSplit_PanesShareTheSpace(t *testing.T) {
	m := testModel(t, sized(100, 40))

	_ = m.handleKey(ctrlArrow(tea.KeyCtrlRight))
	assert.Equal(t, 100, m.leftW+max(m.width-m.leftW, 1), "the two top panes still tile the width")

	_ = m.handleKey(ctrlArrow(tea.KeyCtrlUp))
	assert.Equal(t, 40, headerH+m.topH+m.diagH+statusH, "the rows still add up to the terminal height")
}

// TestSplit_DividersStopShortOfTheEdge pins the travel limits.
//
// A pane driven to nothing could not be recovered: the keys that would bring it back are advertised in a status line
// the collapsed pane no longer has room to explain.
func TestSplit_DividersStopShortOfTheEdge(t *testing.T) {
	t.Run("vertical", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		for range 100 {
			m.moveVSplit(-1)
		}
		assert.Equal(t, minLeftPct, m.leftPct)
		assert.Positive(t, m.leftW, "the left pane is still on screen")

		for range 100 {
			m.moveVSplit(+1)
		}
		assert.Equal(t, maxLeftPct, m.leftPct)
		assert.Positive(t, m.width-m.leftW, "the spec pane is still on screen")
	})

	t.Run("horizontal", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		for range 100 {
			m.moveHSplit(-1)
		}
		assert.Equal(t, minDiagPct, m.diagPct)
		assert.GreaterOrEqual(t, m.diagH, minDiagH)

		for range 100 {
			m.moveHSplit(+1)
		}
		assert.Equal(t, maxDiagPct, m.diagPct)
		assert.GreaterOrEqual(t, m.topH, minTopH, "the top row keeps its floor")
	})
}

// TestSplit_ProportionSurvivesAResize is why the dividers are percentages, not cell counts.
//
// A terminal resize must keep the layout the user chose,
// rather than hand the difference to whichever pane was measured in cells.
func TestSplit_ProportionSurvivesAResize(t *testing.T) {
	m := testModel(t, sized(100, 40))
	m.moveVSplit(+2) // 33% → 43%
	require.Equal(t, 43, m.leftPct)
	require.Equal(t, 43, m.leftW)

	// The terminal doubles in width.
	_, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})

	assert.Equal(t, 43, m.leftPct, "the chosen proportion is untouched")
	assert.Equal(t, 86, m.leftW, "and it is what the new width is divided by")
}

// TestSplit_DefaultsMatchTheHistoricLayout guards the starting geometry.
//
// This is a chrome feature, so a fresh session must look exactly as it did before the dividers could move.
func TestSplit_DefaultsMatchTheHistoricLayout(t *testing.T) {
	m := testModel(t, sized(120, 40))

	assert.Equal(t, 120*defaultLeftPct/100, m.leftW)
	assert.Equal(t, 40*defaultDiagPct/100, m.diagH)
	assert.Equal(t, 40/4, m.diagH, "a quarter of the height, as before")
}

// TestSplit_WorksFromTheEditor pins that the resize keys are not swallowed by the textarea.
//
// Wanting more room for the pane you are typing in is exactly when the resize is wanted, and every other key the editor
// does not name explicitly goes to the buffer.
func TestSplit_WorksFromTheEditor(t *testing.T) {
	f := newReloadFixture(t, "package p\n")
	f.m.width, f.m.height = 100, 40
	f.m.recalcLayout()
	f.m.fileView.StartEdit()
	require.True(t, f.m.fileView.Editing())
	before := f.m.leftW

	_ = f.m.handleKey(ctrlArrow(tea.KeyCtrlRight))

	assert.Greater(t, f.m.leftW, before, "the split key reached the model rather than the buffer")
	assert.Equal(t, "package p\n", f.m.fileView.Value(), "and typed nothing into the file")
}
