// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// B-rescan-anchor — a re-render must keep the user on the same NODE.
//
// This is the hot path: every save triggers a rescan, and live-reload is the
// tool's reason to exist. Carrying the raw line number across would slide the
// user to a different node whenever the spec gained or lost lines above them.

// rescanBase is the starting render.
const rescanBase = `{
  "definitions": {
    "Address": {
      "properties": {
        "city": {
          "type": "string"
        }
      }
    },
    "User": {
      "properties": {
        "name": {
          "type": "string"
        }
      }
    }
  }
}`

// rescanGrown is the same spec with a definition inserted ABOVE both, so every
// node below shifts down.
const rescanGrown = `{
  "definitions": {
    "AAA": {
      "properties": {
        "zzz": {
          "type": "string"
        }
      }
    },
    "Address": {
      "properties": {
        "city": {
          "type": "string"
        }
      }
    },
    "User": {
      "properties": {
        "name": {
          "type": "string"
        }
      }
    }
  }
}`

// rescanShrunk drops User entirely — the type was deleted.
const rescanShrunk = `{
  "definitions": {
    "Address": {
      "properties": {
        "city": {
          "type": "string"
        }
      }
    }
  }
}`

func newRescanModel(t *testing.T) *Model {
	t.Helper()
	m := &Model{
		spec:        panels.NewSpec(),
		fileView:    panels.NewFileView(),
		searchInput: textinput.New(),
	}
	// Tall enough that a node shifted by a few lines is still on screen —
	// otherwise "did it avoid scrolling?" cannot be asked.
	m.spec.SetSize(60, 20)
	m.fileView.SetSize(60, 20)
	m.focused = paneSpec
	m.specJSON = rescanBase
	m.refreshSpec()

	return m
}

// parkOn puts the cursor on a pointer and returns the line it was on.
func parkOn(t *testing.T, m *Model, ptr string) int {
	t.Helper()
	line, ok := m.specIndex.LineForPointer(ptr)
	require.True(t, ok, "pointer %q must be in the render", ptr)
	m.spec.SetCursor(line)

	return line
}

func TestRescan_KeepsTheCursorOnTheSameNode(t *testing.T) {
	m := newRescanModel(t)
	const ptr = "/definitions/User"
	before := parkOn(t, m, ptr)

	// A rescan whose spec gained a definition above the one being read.
	m.specJSON = rescanGrown
	m.refreshSpec()

	after, ok := m.specIndex.LineForPointer(ptr)
	require.True(t, ok)
	require.NotEqual(t, before, after, "precondition: the node moved in the new render")

	assert.Equal(t, after, m.spec.CursorLine(),
		"the cursor followed the node, not the line number")
}

func TestRescan_ViaScanResultMessage(t *testing.T) {
	m := newRescanModel(t)
	const ptr = "/definitions/User/properties/name"
	parkOn(t, m, ptr)

	// The real path a scan arrives by.
	_, _ = m.Update(scanResultMsg{json: rescanGrown})

	after, ok := m.specIndex.LineForPointer(ptr)
	require.True(t, ok)
	assert.Equal(t, after, m.spec.CursorLine())
}

// When the node is gone, land in its neighbourhood rather than somewhere
// arbitrary: the walk falls back to the nearest surviving ancestor.
func TestRescan_DeletedNodeFallsBackToItsAncestor(t *testing.T) {
	m := newRescanModel(t)
	parkOn(t, m, "/definitions/User/properties/name")

	m.specJSON = rescanShrunk
	m.refreshSpec()

	_, gone := m.specIndex.LineForPointer("/definitions/User")
	require.False(t, gone, "precondition: User was deleted")

	definitionsLine, ok := m.specIndex.LineForPointer("/definitions")
	require.True(t, ok)
	assert.Equal(t, definitionsLine, m.spec.CursorLine(),
		"fell back to the nearest ancestor that survived")
}

// An unchanged rescan — the common case, since most saves do not move anything
// — must not move the cursor at all.
func TestRescan_IdenticalSpecDoesNotMoveTheCursor(t *testing.T) {
	m := newRescanModel(t)
	before := parkOn(t, m, "/definitions/User")
	topBefore := m.spec.TopLine()

	m.refreshSpec()

	assert.Equal(t, before, m.spec.CursorLine())
	assert.Equal(t, topBefore, m.spec.TopLine(),
		"and the viewport did not jump either")
}

// The restore scrolls minimally rather than centring: on the hot path, yanking
// the viewport on every save would be worse than the drift it fixes.
func TestRescan_DoesNotYankTheViewport(t *testing.T) {
	m := newRescanModel(t)
	parkOn(t, m, "/definitions/User")
	topBefore := m.spec.TopLine()

	m.specJSON = rescanGrown
	m.refreshSpec()

	// The node moved a few lines but is still on screen, so the view should be
	// steady — not recentred on the cursor.
	assert.Equal(t, topBefore, m.spec.TopLine(),
		"the node was still visible, so nothing needed to scroll")
}

// ...whereas an explicit format switch is a deliberate change of view, and does
// recentre.
func TestRescan_FormatSwitchRecentres(t *testing.T) {
	m := newRescanModel(t)
	m.specYAML = "definitions:\n  Address:\n    properties:\n      city:\n        type: string\n  User:\n    properties:\n      name:\n        type: string\n"
	parkOn(t, m, "/definitions/User")

	m.setSpecFormat("YAML")

	line, ok := m.specIndex.LineForPointer("/definitions/User")
	require.True(t, ok)
	assert.Equal(t, line, m.spec.CursorLine(), "same node")
	assert.Equal(t, max(line-(20-3)/2, 0), m.spec.TopLine(), "centred in the viewport")
}

// The first scan has no previous index to anchor against, and must not panic or
// jump anywhere.
func TestRescan_FirstScanStartsAtTheTop(t *testing.T) {
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.spec.SetSize(60, 20)

	m.specJSON = rescanBase
	m.refreshSpec()

	assert.Equal(t, 0, m.spec.CursorLine())
}
