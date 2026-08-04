// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for the cross-reference layer as the model drives it: follow modes, the JSON/YAML toggle, what a re-render
// does to the cursor, and the spans a render installs. The resolution underneath is tested purely in
// crossref_resolve_test.go.

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Auto-follow: one pane drives, the other mirrors, and any focus change ends it.
func TestFollow(t *testing.T) {
	t.Run("source driven", func(t *testing.T) {
		m := followFixture(t)
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/User/properties/email", Pos: token.Position{Filename: "user.go", Line: 5}},
		})
		m.currentFile = "user.go"
		m.fileView.SetFile("user.go", "a\nb\nc\nd\ne\nf")
		m.fileView.GotoLine(4) // 0-based row 4 == source line 5
		m.focused = paneTree
		m.leftMode = modeView

		m.toggleFollow(followSource)
		assert.Equal(t, followSource, m.follow, "f enters source-driven follow")
		assert.Equal(t, "/definitions/User/properties/email", m.followTarget)

		// A line with no anchor at or above reports honestly — and names the cause, rather than flattening every miss into
		// one opaque message.
		m.fileView.GotoLine(0) // source line 1, before the first anchor (line 5)
		m.syncFollowIfActive()
		assert.Equal(t, noAnchorDesc, m.followTarget)

		// Moving focus off the driver exits follow.
		m.focused = paneSpec
		m.syncFollowIfActive()
		assert.Equal(t, followOff, m.follow, "leaving the driver pane exits follow")
	})

	t.Run("spec driven", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "user.go")
		require.NoError(t, os.WriteFile(src, []byte("package p\n\ntype User struct{}\n"), 0o600))

		m := testModelIn(t, dir,
			panelSize(60, 20),
			// The node maps to the top line of the viewport (YOffset 0).
			withSpecContent("{\n  \"definitions\": {}\n}", map[int]string{0: "/definitions/User"}),
		)
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/User", Pos: token.Position{Filename: src, Line: 3}},
		})
		m.focused = paneSpec

		m.toggleFollow(followSpec)
		assert.Equal(t, followSpec, m.follow)
		assert.Equal(t, src, m.currentFile, "the source follower loads the producing file")
		assert.Equal(t, paneSpec, m.focused, "the driver keeps focus")
		assert.Contains(t, m.followTarget, "/definitions/User")
		assert.Contains(t, m.followTarget, "user.go:3")

		// f again toggles off.
		m.toggleFollow(followSpec)
		assert.Equal(t, followOff, m.follow)
		assert.Empty(t, m.followTarget)
	})

	t.Run("exit clears state", func(t *testing.T) {
		m := followFixture(t)
		m.srcIndex = index.BuildSourceIndex(nil)
		m.follow = followSpec
		m.followTarget = "something"

		m.exitFollow()
		assert.Equal(t, followOff, m.follow)
		assert.Empty(t, m.followTarget)
		// Idempotent.
		m.exitFollow()
		assert.Equal(t, followOff, m.follow)
	})
}

// The two renders of the same spec, indexing the same pointers at DIFFERENT lines — the whole point of preserving the
// pointer rather than the line across a format toggle.
//
// Both renders are deliberately taller than the test viewport (below) so neither clamps: YAML is roughly half the
// height of JSON, which is exactly why carrying the raw line number across is wrong.
const (
	toggleJSON = `{
  "definitions": {
    "Address": {
      "properties": {
        "city": {
          "type": "string"
        },
        "zip": {
          "type": "string"
        }
      }
    },
    "User": {
      "properties": {
        "email": {
          "type": "string"
        },
        "name": {
          "type": "string"
        }
      }
    }
  }
}`

	toggleYAML = `definitions:
  Address:
    properties:
      city:
        type: string
      zip:
        type: string
  User:
    properties:
      email:
        type: string
      name:
        type: string
`
)

// The email property's line in each render — the node the toggle must preserve.
const (
	emailPtr      = "/definitions/User/properties/email"
	emailJSONLine = 14
	emailYAMLLine = 9
)

// The JSON/YAML toggle keeps the cursor on the same NODE, never the same line number.
func TestSpecFormatToggle(t *testing.T) {
	t.Run("preserves pointer not line", func(t *testing.T) {
		m := toggleFixture(t)

		// Park the viewport on the `email` property.
		jsonLine, ok := m.specIndex.LineForPointer(emailPtr)
		require.True(t, ok, "the email property must be indexed in the JSON render")
		require.Equal(t, emailJSONLine, jsonLine)
		m.spec.SetCursor(jsonLine)
		require.Equal(t, jsonLine, m.spec.CursorLine())

		m.setSpecFormat("YAML")

		require.Equal(t, "YAML", m.spec.Format())
		yamlLine, ok := m.specIndex.LineForPointer(emailPtr)
		require.True(t, ok, "the same pointer must be indexed in the YAML render")
		require.Equal(t, emailYAMLLine, yamlLine, "the two renders put the node on different lines")

		assert.Equal(t, yamlLine, m.spec.CursorLine(),
			"the toggle must land on the same NODE, not the same line number")

		// Round-tripping back restores the JSON line for the same node.
		m.setSpecFormat("JSON")
		assert.Equal(t, jsonLine, m.spec.CursorLine())
	})

	t.Run("same format is noop", func(t *testing.T) {
		m := toggleFixture(t)
		m.spec.SetCursor(emailJSONLine)

		m.setSpecFormat("JSON")

		assert.Equal(t, emailJSONLine, m.spec.CursorLine(),
			"re-selecting the active format must not move the cursor")
	})

	t.Run("unindexed cursor line", func(t *testing.T) {
		m := toggleFixture(t)
		m.specIndex = nil // no index: nothing to preserve, but nothing may panic either

		m.setSpecFormat("YAML")

		assert.Equal(t, "YAML", m.spec.Format())
	})

	// TestSpecFormatToggle_ViaKey checks the binding actually routes through the pointer-preserving path (the bug was in
	// the key handler, not the helper).
	t.Run("via key", func(t *testing.T) {
		m := toggleFixture(t)
		m.spec.SetCursor(emailJSONLine)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlY})

		assert.Equal(t, "YAML", m.spec.Format())
		assert.Equal(t, emailYAMLLine, m.spec.CursorLine(),
			"ctrl+y must preserve the node under the cursor")
	})
}

func TestFollowBadge_StaleWhileDirty(t *testing.T) {
	m := testModel(t, viewing("user.go", "package p\n"))
	m.follow = followSource
	m.followTarget = "/definitions/User"

	require.False(t, m.stale(), "a freshly loaded buffer matches the last scan")
	assert.NotContains(t, testutils.StripANSI(m.followBadge()), "STALE")

	// An unsaved edit shifts every anchor below it: positions are now older than what is on screen.
	m.fileView.StartEdit()
	_ = m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	require.True(t, m.stale(), "an edited buffer invalidates the recorded positions")
	assert.Contains(t, testutils.StripANSI(m.followBadge()), "STALE")

	// Saving (MarkClean is what saveFile does) clears it again.
	m.fileView.MarkClean()
	assert.False(t, m.stale())
	assert.NotContains(t, testutils.StripANSI(m.followBadge()), "STALE")
}

// Syntax spans installed by a render, and how they compose with the cursor, the search highlight and the gutter.
func TestSyntax(t *testing.T) {
	// The highlight index rides the same walk as the other two, so a refresh must install all three or none.
	t.Run("refresh installs spans", func(t *testing.T) {
		m := syntaxModel(t)

		require.NotNil(t, m.specIndex)
		require.NotNil(t, m.refIndex)
		assert.Contains(t, testutils.StripANSI(m.spec.View(false)), `"swagger"`,
			"the text is unchanged by highlighting")
	})

	// An empty spec must clear the spans along with the indexes; leaving stale runs behind would colour the placeholder by
	// the old document's columns.
	t.Run("empty spec clears spans", func(t *testing.T) {
		m := syntaxModel(t)
		m.scan.JSON = ""
		m.refreshSpec()

		view := testutils.StripANSI(m.spec.View(false))
		assert.Contains(t, view, "(no spec generated yet)")
		assert.NotContains(t, view, "swagger")
	})

	// Highlighting must never alter the text — the same invariant the renderer is tested on, asserted here through the
	// whole pipeline.
	t.Run("text survives highlighting", func(t *testing.T) {
		m := syntaxModel(t)

		plain := testutils.StripANSI(m.spec.View(false))
		for _, want := range []string{`"swagger": "2.0",`, `"maxLength": 64`, `"type": "string",`} {
			assert.Contains(t, plain, want)
		}
	})

	// Precedence: the cursor and a search hit answer questions the USER asked, so they take the whole line rather than
	// compete with syntax colour for it.
	t.Run("precedence cursor then search then syntax", func(t *testing.T) {
		m := syntaxModel(t)

		// Both renders must contain the same visible text regardless of which styling layer won.
		m.spec.SetCursor(1)
		withCursor := testutils.StripANSI(m.spec.View(true))
		assert.Contains(t, withCursor, `"swagger": "2.0",`)

		require.Positive(t, m.spec.Search("maxLength"))
		withSearch := testutils.StripANSI(m.spec.View(true))
		assert.Contains(t, withSearch, `"maxLength": 64`)

		m.spec.ClearSearch()
		assert.Contains(t, testutils.StripANSI(m.spec.View(true)), `"maxLength": 64`)
	})

	// A search must still count matches on the raw line: highlighting changes how a line is drawn, never what it contains.
	t.Run("search still counts matches", func(t *testing.T) {
		m := syntaxModel(t)

		n := m.spec.Search(`"type"`)

		assert.Equal(t, 1, n)
		cur, total := m.spec.MatchInfo()
		assert.Equal(t, 1, cur)
		assert.Equal(t, 1, total)
	})

	// The gutter is prefixed after styling, so the two must coexist.
	t.Run("coexists with the gutter", func(t *testing.T) {
		m := syntaxModel(t)
		m.spec.SetGutter(map[int]rune{2: panels.GutterAnchor})

		view := testutils.StripANSI(m.spec.View(false))

		assert.Contains(t, view, string(panels.GutterAnchor))
		assert.Contains(t, view, `"definitions"`)
	})
}

// Against a real scan, over both renders: the visible text must be exactly the document, highlighted or not.
func TestE2E_SyntaxLeavesTheSpecIntact(t *testing.T) {
	m := scanPetstore(t)
	m.spec.SetSize(200, 40)

	for _, format := range []string{"JSON", "YAML"} {
		m.setSpecFormat(format)
		body := m.scan.JSON
		if format == "YAML" {
			body = m.scan.YAML
		}
		require.NotEmpty(t, body)

		// Take a line from the middle of the document and require it to appear verbatim in the rendered pane.
		lines := strings.Split(body, "\n")
		probe := strings.TrimRight(lines[len(lines)/2], " ")
		require.NotEmpty(t, strings.TrimSpace(probe))

		m.spec.JumpTo(len(lines) / 2)
		assert.Contains(t, testutils.StripANSI(m.spec.View(false)), strings.TrimSpace(probe),
			"%s: highlighting altered the rendered text", format)
	}
}

// B-rescan-anchor — a re-render must keep the user on the same NODE.
//
// This is the hot path: every save triggers a rescan, and live-reload is the tool's reason to exist.
// Carrying the raw line number across would slide the user to a different node whenever the spec gained or lost lines
// above them.

// What a re-render does to the cursor: it follows the node, and moves the viewport as little as it can.
func TestRescan(t *testing.T) {
	t.Run("keeps the cursor on the same node", func(t *testing.T) {
		m := newRescanModel(t)
		const ptr = "/definitions/User"
		before := parkOn(t, m, ptr)

		// A rescan whose spec gained a definition above the one being read.
		m.scan.JSON = rescanGrown
		m.refreshSpec()

		after, ok := m.specIndex.LineForPointer(ptr)
		require.True(t, ok)
		require.NotEqual(t, before, after, "precondition: the node moved in the new render")

		assert.Equal(t, after, m.spec.CursorLine(),
			"the cursor followed the node, not the line number")
	})

	t.Run("via scan result message", func(t *testing.T) {
		m := newRescanModel(t)
		const ptr = "/definitions/User/properties/name"
		parkOn(t, m, ptr)

		// The real path a scan arrives by.
		_, _ = m.Update(scan.ResultMsg{JSON: rescanGrown})

		after, ok := m.specIndex.LineForPointer(ptr)
		require.True(t, ok)
		assert.Equal(t, after, m.spec.CursorLine())
	})

	// When the node is gone, land in its neighbourhood rather than somewhere arbitrary: the walk falls back to the nearest
	// surviving ancestor.
	t.Run("deleted node falls back to its ancestor", func(t *testing.T) {
		m := newRescanModel(t)
		parkOn(t, m, "/definitions/User/properties/name")

		m.scan.JSON = rescanShrunk
		m.refreshSpec()

		_, gone := m.specIndex.LineForPointer("/definitions/User")
		require.False(t, gone, "precondition: User was deleted")

		definitionsLine, ok := m.specIndex.LineForPointer("/definitions")
		require.True(t, ok)
		assert.Equal(t, definitionsLine, m.spec.CursorLine(),
			"fell back to the nearest ancestor that survived")
	})

	// An unchanged rescan — the common case, since most saves do not move anything — must not move the cursor at all.
	t.Run("identical spec does not move the cursor", func(t *testing.T) {
		m := newRescanModel(t)
		before := parkOn(t, m, "/definitions/User")
		topBefore := m.spec.TopLine()

		m.refreshSpec()

		assert.Equal(t, before, m.spec.CursorLine())
		assert.Equal(t, topBefore, m.spec.TopLine(),
			"and the viewport did not jump either")
	})

	// The restore scrolls minimally rather than centring: on the hot path, yanking the viewport on every save would be
	// worse than the drift it fixes.
	t.Run("does not yank the viewport", func(t *testing.T) {
		m := newRescanModel(t)
		parkOn(t, m, "/definitions/User")
		topBefore := m.spec.TopLine()

		m.scan.JSON = rescanGrown
		m.refreshSpec()

		// The node moved a few lines but is still on screen, so the view should be steady — not recentred on the cursor.
		assert.Equal(t, topBefore, m.spec.TopLine(),
			"the node was still visible, so nothing needed to scroll")
	})

	// ...whereas an explicit format switch is a deliberate change of view, and does recentre.
	t.Run("format switch recentres", func(t *testing.T) {
		m := newRescanModel(t)
		m.scan.YAML = "definitions:\n  Address:\n    properties:\n      city:\n        type: string\n  User:\n    properties:\n      name:\n        type: string\n"
		parkOn(t, m, "/definitions/User")

		m.setSpecFormat("YAML")

		line, ok := m.specIndex.LineForPointer("/definitions/User")
		require.True(t, ok)
		assert.Equal(t, line, m.spec.CursorLine(), "same node")
		assert.Equal(t, max(line-(20-3)/2, 0), m.spec.TopLine(), "centred in the viewport")
	})

	// The first scan has no previous index to anchor against, and must not panic or jump anywhere.
	t.Run("first scan starts at the top", func(t *testing.T) {
		m := testModel(t, panelSize(60, 20))

		m.scan.JSON = rescanBase
		m.refreshSpec()

		assert.Equal(t, 0, m.spec.CursorLine())
	})
}

// followFixture builds a Model wired with a known spec/source index pair, ready to drive follow mode without a real
// scan.
func followFixture(t *testing.T) *Model {
	t.Helper()

	return testModel(t,
		panelSize(60, 20),
		withSpecContent(
			"line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
			map[int]string{7: "/definitions/User/properties/email"},
		),
	)
}

const syntaxSpec = `{
  "swagger": "2.0",
  "definitions": {
    "User": {
      "properties": {
        "email": {
          "type": "string",
          "maxLength": 64
        }
      }
    }
  }
}`

func syntaxModel(t *testing.T) *Model {
	t.Helper()

	return testModel(t, panelSize(70, 24), withSpecJSON(syntaxSpec))
}

// toggleFixture builds a model holding both renders of the same spec, with the spec pane focused and the JSON index
// live.
//
// The pane is short on purpose (a 5-line viewport) so both renders can actually scroll to the target node.
func toggleFixture(t *testing.T) *Model {
	t.Helper()
	return testModel(t,
		panelSize(60, 8),
		focusedOn(paneSpec),
		withRenders(toggleJSON, toggleYAML),
	)
}

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

// rescanGrown is the same spec with a definition inserted ABOVE both, so every node below shifts down.
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
	// Tall enough that a node shifted by a few lines is still on screen — otherwise "did it avoid scrolling?" cannot be
	// asked.
	m := testModel(t,
		panelSize(60, 20),
		focusedOn(paneSpec),
		withSpecJSON(rescanBase),
	)

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
