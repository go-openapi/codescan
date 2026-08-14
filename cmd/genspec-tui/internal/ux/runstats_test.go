// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for the run-cost card: the key that opens it, and what the model feeds it.

package ux

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
)

// errScanFailed stands in for a hard error out of codescan.Run.
var errScanFailed = errors.New("no packages")

func TestRunStats_OpensOnM(t *testing.T) {
	m := testModel(t, sized(100, 40))

	_ = m.handleKey(testutils.KeyRune('m'))

	assert.True(t, m.runstats.IsOpen(), "m opens the run-cost card")
	assert.Contains(t, testutils.StripANSI(m.runstats.View()), "Last run")
}

func TestRunStats_CapturesKeysWhileOpen(t *testing.T) {
	m := testModel(t, sized(100, 40))
	_ = m.handleKey(testutils.KeyRune('m'))
	require.True(t, m.runstats.IsOpen())

	for _, msg := range []tea.KeyMsg{testutils.KeyRune('r'), testutils.KeyRune('o'), testutils.KeyRune('/')} {
		_ = m.handleKey(msg)
	}

	assert.True(t, m.runstats.IsOpen(), "still open")
	assert.False(t, m.scan.Running, "r did not start a scan behind the card")
	assert.False(t, m.options.IsOpen(), "o did not open the options behind the card")
	assert.False(t, m.search.Active(), "/ did not open search behind the card")

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, m.runstats.IsOpen())
}

// m is an ordinary character in the editor; opening a modal there would make the buffer unusable.
func TestRunStats_DoesNotHijackTheEditor(t *testing.T) {
	m := testModel(t, sized(100, 40))
	m.loadFileQuietly(testutils.WriteTempGo(t, "package p\n"))
	m.focused, m.leftMode = paneTree, modeView
	_ = m.fileView.StartEdit()
	require.True(t, m.fileView.Editing())

	_ = m.handleKey(testutils.KeyRune('m'))

	assert.False(t, m.runstats.IsOpen(), "m typed into the editor is a character, not a command")
}

// The card reports the run the model absorbed, and is told whether that run replaced a document - which is what makes
// its retained figure readable.
func TestRunStats_TakesTheCostFromTheScan(t *testing.T) {
	m := testModel(t, sized(100, 40))

	first := scan.ResultMsg{JSON: "{}", Cost: scan.Cost{Measured: true, AllocScan: 4 << 20, LiveAfter: 2 << 20}}
	m.absorbScan(first)

	require.True(t, m.scan.Cost.Measured, "the model keeps what the run cost")
	assert.Equal(t, uint64(4<<20), m.scan.Cost.AllocScan)
	assert.NotContains(t, testutils.StripANSI(m.runstats.View()), "replaced a spec",
		"nothing was held before the first scan")

	m.absorbScan(first)
	assert.Contains(t, testutils.StripANSI(m.runstats.View()), "replaced a spec",
		"the second run was measured with the first document still live")
}

// A run that failed still costs something, and the reading is the only account of where it went.
func TestRunStats_KeepsTheCostOfAFailedScan(t *testing.T) {
	m := testModel(t, sized(100, 40))

	m.absorbScan(scan.ResultMsg{Err: errScanFailed, Cost: scan.Cost{Measured: true, AllocScan: 1 << 20}})

	assert.True(t, m.scan.Cost.Measured)
	assert.Contains(t, testutils.StripANSI(m.runstats.View()), "allocated")
}
