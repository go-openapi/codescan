// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Update is the only door the bubbletea runtime uses, but the rest of the suite reaches past it to
// handleKey. These drive one message of each kind through the door itself, so the dispatch arms
// - and the commands they hand back to the runtime - are exercised the way the program runs.
//
// The returned commands are deliberately NOT executed: several of them block on a channel or sleep
// out a debounce window, which is the runtime's job, not a test's.

func TestUpdate_WindowSize(t *testing.T) {
	m := testModel(t)
	m.ready = false

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})

	assert.Nil(t, cmd)
	assert.True(t, m.ready, "the first size message is what makes the model renderable")
	assert.Equal(t, 120, m.width)
	assert.Positive(t, m.topH, "the layout regions are recomputed")
}

func TestUpdate_Key(t *testing.T) {
	m := testModel(t, sized(100, 40))

	_, _ = m.Update(testutils.KeyRune('h'))

	assert.True(t, m.help.IsOpen(), "a key reaches the dispatch through Update")
}

func TestUpdate_SpinnerTick(t *testing.T) {
	t.Run("while a scan runs", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		m.scan.Running = true

		_, cmd := m.Update(spinner.TickMsg{})

		assert.NotNil(t, cmd, "the tick loop keeps itself going")
	})

	t.Run("once it has finished", func(t *testing.T) {
		m := testModel(t, sized(100, 40))

		_, cmd := m.Update(spinner.TickMsg{})

		assert.Nil(t, cmd, "no scan in flight, so the loop stops rather than spinning forever")
	})
}

func TestUpdate_ScanResult(t *testing.T) {
	m := testModel(t, sized(100, 40))
	m.scan.Running = true

	_, cmd := m.Update(scan.ResultMsg{
		JSON:    refSpecJSON,
		Paths:   2,
		Defs:    5,
		Elapsed: 300 * time.Millisecond,
	})

	assert.Nil(t, cmd)
	assert.False(t, m.scan.Running, "the result ends the scan")
	assert.Equal(t, 5, m.scan.NumDefs)
	assert.Contains(t, m.spec.Content(), "definitions", "the spec pane renders the result")
	require.NotNil(t, m.specIndex, "and the indexes are rebuilt from it")
}

// A change event opens a debounce window and keeps listening.
//
// An editor save often fires several events, and only the last one may survive to trigger a rescan.
func TestUpdate_FileSystemEvent(t *testing.T) {
	m := testModel(t, sized(100, 40))
	before := m.watch.gen

	_, cmd := m.Update(fsEventMsg{})

	assert.NotNil(t, cmd)
	assert.Greater(t, m.watch.gen, before, "the window is (re)opened")
}

func TestUpdate_Debounce(t *testing.T) {
	t.Run("the newest window rescans", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		gen := m.watch.Bump()

		_, cmd := m.Update(debounceMsg{gen: gen})

		assert.NotNil(t, cmd)
		assert.True(t, m.scan.Running, "the rescan is under way")
	})

	t.Run("a superseded window does nothing", func(t *testing.T) {
		m := testModel(t, sized(100, 40))
		stale := m.watch.Bump()
		m.watch.Bump() // a newer change arrived during the quiet period

		_, cmd := m.Update(debounceMsg{gen: stale})

		assert.Nil(t, cmd)
		assert.False(t, m.scan.Running)
	})
}

func TestUpdate_CopyResult(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := testModel(t, sized(100, 40))

		_, cmd := m.Update(copyResultMsg{})

		assert.NotNil(t, cmd, "the notice is scheduled to expire")
		assert.Equal(t, "copied to clipboard", m.notice)
	})

	t.Run("failure names the error", func(t *testing.T) {
		m := testModel(t, sized(100, 40))

		_, _ = m.Update(copyResultMsg{err: errClipboardTest})

		assert.Contains(t, m.notice, "no clipboard tool")
	})
}

type testError string

func (e testError) Error() string {
	return string(e)
}

const errClipboardTest testError = "no clipboard tool"

func TestUpdate_ClearNotice(t *testing.T) {
	m := testModel(t, sized(100, 40))
	m.notice = "saved user.go"

	_, cmd := m.Update(clearNoticeMsg{})

	assert.Nil(t, cmd)
	assert.Empty(t, m.notice)
}

// waitForFS is the listen loop.
//
// Each call blocks for one event and is re-issued,
// so a closed channel has to end the loop quietly rather than spinning on a nil read.
//
// Its command is safe to execute here because the channel is primed.
func TestWaitForFS(t *testing.T) {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}

	assert.Equal(t, fsEventMsg{}, waitForFS(ch)(), "an event becomes one message")

	close(ch)
	assert.Nil(t, waitForFS(ch)(), "a closed channel ends the loop")
}

// The YAML view costs more than the JSON it is made from, and most sessions never open it - so it is rendered when it
// is asked for, and kept until the document it was made from is replaced.
func TestSpecYAML_RendersOnDemandAndIsKept(t *testing.T) {
	m := scanPetstore(t)
	require.Empty(t, m.scan.YAML)

	switchToYAML(t, m)
	rendered := m.scan.YAML

	assert.Nil(t, m.setSpecFormat("JSON"), "going back needs no work")
	assert.Nil(t, m.setSpecFormat("YAML"), "nor does coming back: the render is still the right one")
	assert.Equal(t, rendered, m.scan.YAML)
}

// Between asking for YAML and getting it there is a frame to draw, and an empty one reads as a scan that produced
// nothing.
func TestSpecYAML_SaysItIsRendering(t *testing.T) {
	m := scanPetstore(t)

	require.NotNil(t, m.setSpecFormat("YAML"))

	assert.Contains(t, testutils.StripANSI(m.spec.View(false)), "rendering YAML")
}

// A conversion outlived by a rescan describes a document nobody is looking at any more.
func TestSpecYAML_DropsAConversionARescanOvertook(t *testing.T) {
	m := scanPetstore(t)

	cmd := m.setSpecFormat("YAML")
	require.NotNil(t, cmd)
	converted := cmd() // finished, not yet delivered

	_, _ = m.Update(petstoreRes) // ...and a rescan lands first
	_, _ = m.Update(converted)

	assert.Empty(t, m.scan.YAML, "the render was of the document that has just been replaced")
}

// Anything the model does not recognise belongs to whichever panel has focus.
func TestUpdate_UnknownMessageGoesToTheFocusedPane(t *testing.T) {
	m := testModel(t, sized(100, 40), viewing("user.go", "a\nb\n"), focusedOn(paneTree))

	for _, p := range []pane{paneTree, paneSpec, paneDiag} {
		m.focused = p
		_, cmd := m.Update(struct{ nothing bool }{})
		assert.Nil(t, cmd, "pane %d ignores a message it has no use for", p)
	}
}

// A scan runs on a goroutine of the runtime's while the event loop keeps taking keys, and nothing stops the options
// overlay from being opened and toggled meanwhile.
//
// Both reach the model's configuration, so this one only says anything under -race - which is how CI runs it.
func TestScan_OptionsAreToggledWhileAScanRuns(t *testing.T) {
	m := testModelIn(t, filepath.Join(fixturesDir(t), "goparsing", "petstore"))

	// Running already, so startScan hands back the bare scan command rather than batching a spinner tick with it -
	// which is also the case that matters: a rescan fired while the previous one is still in flight.
	m.scan.Running = true
	run := m.startScan() // built on the event loop...

	done := make(chan struct{})
	go func() {
		defer close(done)

		_ = run() // ...and executed elsewhere, as the bubbletea runtime does
	}()

	_ = m.handleKey(testutils.KeyRune('o'))
	for {
		select {
		case <-done:
			return
		default:
			_ = m.handleKey(testutils.KeyRune(' '))
		}
	}
}
