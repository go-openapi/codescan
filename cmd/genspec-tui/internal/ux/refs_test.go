// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Tests for the $ref machinery: the find-references cycle, go-to-definition, and the gutter markers that advertise
// which lines lead somewhere.

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestRefs_FixtureLines(t *testing.T) {
	m := newRefModel(t)

	// Guards the hand-counted constants the rest of the file relies on.
	for ptr, want := range map[string]int{
		"/definitions/User":                      rmLineUserDecl,
		"/definitions/User/properties/name":      rmLineUserName,
		"/definitions/Team/properties/lead":      rmLineLead - 1,
		"/paths/~1pets/get/responses/200/schema": rmLineRespRef - 1,
	} {
		got, ok := m.specIndex.LineForPointer(ptr)
		require.True(t, ok, "pointer %q", ptr)
		assert.Equal(t, want, got, "rendered line of %s", ptr)
	}
	require.Equal(t, 4, m.refIndex.Len(), "three local refs plus the external one")
}

func TestRefs_CycleForward(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl) // park on the definition

	require.Nil(t, m.cycleRefs(+1))
	assert.Equal(t, rmLineLead, m.spec.CursorLine(), "first use")
	assert.Contains(t, m.refs.Status, "ref 1/3")
	assert.Contains(t, m.refs.Status, "/definitions/Team/properties/lead")

	require.Nil(t, m.cycleRefs(+1))
	assert.Equal(t, rmLineItemsRef, m.spec.CursorLine(), "second use")
	assert.Contains(t, m.refs.Status, "ref 2/3")

	require.Nil(t, m.cycleRefs(+1))
	assert.Equal(t, rmLineRespRef, m.spec.CursorLine(), "third use")
	assert.Contains(t, m.refs.Status, "ref 3/3")

	// Wraps back to the first.
	require.Nil(t, m.cycleRefs(+1))
	assert.Equal(t, rmLineLead, m.spec.CursorLine(), "wrapped")
	assert.Contains(t, m.refs.Status, "ref 1/3")
}

func TestRefs_CycleBackwardEntersAtTheLastSite(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl)

	require.Nil(t, m.cycleRefs(-1))
	assert.Equal(t, rmLineRespRef, m.spec.CursorLine(),
		"a backward step into a fresh cycle enters at the last site")
	assert.Contains(t, m.refs.Status, "ref 3/3")

	require.Nil(t, m.cycleRefs(-1))
	assert.Contains(t, m.refs.Status, "ref 2/3")
}

// The cursor need not be on the definition line itself: a node inside the definition resolves up to it, the way the
// rest of the tool resolves pointers.
func TestRefs_CycleFromInsideTheDefinition(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserName) // /definitions/User/properties/name

	require.Nil(t, m.cycleRefs(+1))
	assert.Contains(t, m.refs.Status, "ref 1/3 of /definitions/User",
		"an inner node cycles the enclosing definition's uses")
}

// Scrolling away ends the cycle: the next F3 asks about wherever the user now is, rather than continuing to walk the
// previous definition's uses.
func TestRefs_ScrollingAwayStartsAFreshCycle(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl)
	require.Nil(t, m.cycleRefs(+1))
	require.Contains(t, m.refs.Status, "ref 1/3")

	// The user scrolls to a node nothing references.
	m.spec.SetCursor(rmLineLogo)
	require.NotNil(t, m.cycleRefs(+1), "a failed start returns the notice-clearing cmd")
	assert.Contains(t, m.notice, "nothing references")
	assert.Empty(t, m.refs.Status,
		"the previous cycle's status must not linger while the user is elsewhere")
	assert.Empty(t, m.refs.Anchor)
}

// Line 0 is the opening brace, which carries no pointer at all — a different miss from "this node has no references",
// and it must say so.
func TestRefs_CycleOnAnUnindexedLine(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(0)

	require.NotNil(t, m.cycleRefs(+1))
	assert.Equal(t, noNodeDesc, m.notice)
	assert.Empty(t, m.refs.Status)
}

func TestRefs_NothingReferencesTheNode(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineLogo) // the external-ref property: nothing points here

	require.NotNil(t, m.cycleRefs(+1))
	assert.Contains(t, m.notice, "nothing references")
	assert.Empty(t, m.refs.Status, "no cycle was started")
}

// The Phase-D keys belong to the spec pane.
//
// From anywhere else they must fall through untouched — Enter in particular still has to open a file in the tree.
func TestRefs_KeysAreSpecPaneOnly(t *testing.T) {
	for _, p := range []pane{paneTree, paneDiag} {
		m := newRefModel(t)
		m.spec.SetCursor(rmLineUserDecl)
		m.focused = p

		for _, msg := range []tea.KeyMsg{
			{Type: tea.KeyF3}, {Type: tea.KeyF15}, {Type: tea.KeyEnter},
		} {
			_ = m.handleKey(msg)
		}

		assert.Empty(t, m.refs.Status, "pane %d", p)
		assert.Empty(t, m.notice, "pane %d", p)
	}
}

func TestRefs_KeyBindings(t *testing.T) {
	// shift+F3 reaches bubbletea v1 as F15 (no Shift modifier on Key; xterm maps shift+F1..F12 onto F13..F24).
	// Both spellings must step backward.
	for _, c := range []struct {
		name string
		msg  tea.KeyMsg
		want string
	}{
		{"f3", tea.KeyMsg{Type: tea.KeyF3}, "ref 1/3"},
		{"f15 (shift+f3 on xterm)", tea.KeyMsg{Type: tea.KeyF15}, "ref 3/3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := newRefModel(t)
			m.spec.SetCursor(rmLineUserDecl)

			_ = m.handleKey(c.msg)

			assert.Contains(t, m.refs.Status, c.want)
		})
	}
}

func TestRefs_GotoDefinition(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineLead) // a local $ref

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, rmLineUserDecl, m.spec.CursorLine(),
		"enter followed the $ref to its definition")
	assert.Equal(t, "→ /definitions/User", m.notice)
}

// Regression: a jump CENTRES its target, so after F3 the line we landed on is not the top of the viewport.
//
// Enter must still act on where the jump put the user — otherwise the headline workflow (find a use, then go back to
// the definition) reports "no $ref on this line".
func TestRefs_CycleThenGotoDefinition(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	require.Contains(t, m.refs.Status, "ref 1/3")
	require.NotEqual(t, rmLineLead, m.spec.TopLine(),
		"precondition: the jump centred, so TopLine is NOT the target line")
	require.Equal(t, rmLineLead, m.spec.CursorLine(), "but the cursor is")

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "→ /definitions/User", m.notice)
	assert.Equal(t, rmLineUserDecl, m.spec.CursorLine(),
		"and the definition we landed on becomes the new cursor, so Enter can chain")
}

// Moving the cursor off the site the cycle parked it on ends the cycle: the next F3 asks about the node the user is now
// on.
func TestRefs_MovingTheCursorEndsTheCycle(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl)
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	require.Equal(t, rmLineLead, m.spec.CursorLine())
	require.Contains(t, m.refs.Status, "ref 1/3")

	// One line down and we are off the site, so the cycle cannot continue. (That line is inside Team, which nothing
	// references — hence no new cycle.)
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, rmLineLead+1, m.spec.CursorLine())

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.NotContains(t, m.refs.Status, "ref 2/3", "the cycle did not continue")
	assert.Contains(t, m.notice, "nothing references")

	// Park inside User again and F3 re-anchors there, from the first site.
	m.spec.SetCursor(rmLineUserName)
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.Contains(t, m.refs.Status, "ref 1/3 of /definitions/User",
		"a fresh cycle, re-anchored on the node now under the cursor")
}

// The spec pane is navigable in its own right: the cursor keys move the cursor, and paging moves it with the view
// rather than leaving it behind off screen.
func TestSpecNav_CursorKeys(t *testing.T) {
	m := newRefModel(t)
	m.topH = 10 // gives handleSpecNav a page size
	m.spec.SetCursor(rmLineUserDecl)

	for _, c := range []struct {
		name string
		msg  tea.KeyMsg
		want int
	}{
		{"down", tea.KeyMsg{Type: tea.KeyDown}, rmLineUserDecl + 1},
		{"j", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, rmLineUserDecl + 2},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, rmLineUserDecl + 1},
		{"k", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, rmLineUserDecl},
		{"home", tea.KeyMsg{Type: tea.KeyHome}, 0},
		{"end", tea.KeyMsg{Type: tea.KeyEnd}, m.spec.LastLine()},
	} {
		t.Run(c.name, func(t *testing.T) {
			_ = m.handleKey(c.msg)
			assert.Equal(t, c.want, m.spec.CursorLine())
		})
	}

	// Paging moves the cursor too, so F3/Enter never act on an off-screen node.
	m.spec.SetCursor(0)
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Positive(t, m.spec.CursorLine(), "page down carried the cursor with it")
}

// The nav keys belong to the spec pane only — j/k must still drive the tree and the diagnostics list.
func TestSpecNav_OnlyFromTheSpecPane(t *testing.T) {
	m := newRefModel(t)
	m.spec.SetCursor(rmLineUserDecl)
	m.focused = paneDiag

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})

	assert.Equal(t, rmLineUserDecl, m.spec.CursorLine(), "the spec cursor did not move")
}

func TestRefs_GotoDefinitionEdges(t *testing.T) {
	t.Run("external ref is reported, not guessed at", func(t *testing.T) {
		m := newRefModel(t)
		m.spec.SetCursor(rmLineLogo)
		before := m.spec.TopLine()

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Contains(t, m.notice, "external ref")
		assert.Contains(t, m.notice, "logo.json")
		assert.Equal(t, before, m.spec.TopLine(), "the viewport held")
	})

	t.Run("no $ref on this line", func(t *testing.T) {
		m := newRefModel(t)
		m.spec.SetCursor(rmLineUserDecl)

		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

		assert.Equal(t, "no $ref on this line", m.notice)
	})
}

// The cycle holds rendered line numbers, so anything that replaces the render must drop it rather than let F3 jump to a
// line that no longer means anything.
func TestRefs_CycleResets(t *testing.T) {
	active := func(t *testing.T) *Model {
		t.Helper()
		m := newRefModel(t)
		m.scan.YAML = "definitions:\n  Team:\n    properties:\n      lead:\n        $ref: '#/definitions/User'\n  User: {}\n"
		m.spec.SetCursor(rmLineUserDecl)
		require.Nil(t, m.cycleRefs(+1))
		require.NotEmpty(t, m.refs.Status)
		return m
	}

	t.Run("format toggle", func(t *testing.T) {
		m := active(t)
		m.setSpecFormat("YAML")
		assert.Empty(t, m.refs.Status)
		assert.Empty(t, m.refs.Anchor)
	})

	t.Run("rescan", func(t *testing.T) {
		m := active(t)
		_, _ = m.Update(scan.ResultMsg{JSON: refModelSpec})
		assert.Empty(t, m.refs.Status)
		assert.Empty(t, m.refs.Anchor)
	})

	t.Run("esc", func(t *testing.T) {
		m := active(t)
		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Empty(t, m.refs.Status)
		assert.Empty(t, m.refs.Anchor)
	})

	t.Run("entering follow mode", func(t *testing.T) {
		m := active(t)
		m.toggleFollow(followSpec)
		assert.Empty(t, m.refs.Status, "follow drives the viewport; the cycle's lines go stale")
	})

	t.Run("opening search", func(t *testing.T) {
		m := active(t)
		_, handled := m.handleSearchControl(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		require.True(t, handled)
		assert.Empty(t, m.refs.Status)
	})
}

// The link gutter design: which lines actually lead somewhere.

// Gutter markers: the discoverability layer saying which lines actually lead somewhere.
func TestGutter(t *testing.T) {
	t.Run("marks anchors and refs", func(t *testing.T) {
		m := newRefModel(t)
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/User", Pos: token.Position{Filename: "user.go", Line: 3}},
			{Pointer: "/definitions/Team", Pos: token.Position{Filename: "team.go", Line: 3}},
		})
		m.rebuildGutters()

		g := m.specGutter()
		require.NotNil(t, g)

		// Anchored definitions carry the anchor marker at their declaration line.
		assert.Equal(t, panels.GutterAnchor, g[rmLineUserDecl], "/definitions/User is anchored")

		// Local $refs carry the ref marker.
		assert.Equal(t, panels.GutterRef, g[rmLineLead], "the lead property's $ref is followable")
		assert.Equal(t, panels.GutterRef, g[rmLineItemsRef])
		assert.Equal(t, panels.GutterRef, g[rmLineRespRef])
	})

	// An external $ref is not followable, so marking it would promise a jump that Enter cannot make.
	t.Run("external refs are not marked", func(t *testing.T) {
		m := newRefModel(t)
		m.srcIndex = index.BuildSourceIndex(nil)
		m.rebuildGutters()

		_, marked := m.specGutter()[rmLineLogo]
		assert.False(t, marked, "the external ref line carries no marker")
	})

	// Nothing to say means no gutter at all, so the pane keeps its full width.
	t.Run("nil when nothing links", func(t *testing.T) {
		m := testModel(t, panelSize(60, 10), withSpecJSON(`{"swagger":"2.0"}`))

		assert.Nil(t, m.specGutter(), "no provenance and no refs → no gutter")
	})

	// Only nodes with an anchor of their OWN are marked.
	//
	// Marking everything that merely resolves through an ancestor would dot nearly every line.
	t.Run("only exact anchors", func(t *testing.T) {
		m := newRefModel(t)
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/User", Pos: token.Position{Filename: "user.go", Line: 3}},
		})
		m.rebuildGutters()

		g := m.specGutter()
		require.Equal(t, panels.GutterAnchor, g[rmLineUserDecl])

		// The name property resolves to User by nearest-ancestor, but has no anchor of its own, so it stays unmarked.
		_, marked := g[rmLineUserName]
		assert.False(t, marked,
			"a node that only resolves through an ancestor is not marked")

		// Sanity: it really does resolve, which is what makes the distinction real.
		_, ok := m.srcIndex.PositionFor("/definitions/User/properties/name")
		require.True(t, ok)
	})

	// The gutter holds rendered line numbers, so a format toggle must rebuild it.
	t.Run("rebuilt on format toggle", func(t *testing.T) {
		m := newRefModel(t)
		m.scan.YAML = "definitions:\n  Team:\n    properties:\n      lead:\n        $ref: '#/definitions/User'\n  User: {}\n"
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/User", Pos: token.Position{Filename: "user.go", Line: 3}},
		})
		m.rebuildGutters()
		jsonGutter := m.specGutter()
		require.Equal(t, panels.GutterAnchor, jsonGutter[rmLineUserDecl])

		m.setSpecFormat("YAML")

		yamlGutter := m.specGutter()
		require.NotNil(t, yamlGutter)
		assert.NotEqual(t, jsonGutter, yamlGutter, "the YAML render puts the nodes on other lines")

		userLine, ok := m.specIndex.LineForPointer("/definitions/User")
		require.True(t, ok)
		assert.Equal(t, panels.GutterAnchor, yamlGutter[userLine], "marked at its YAML line")
	})

	t.Run("source anchors follow the open file", func(t *testing.T) {
		dir := t.TempDir()
		userGo := filepath.Join(dir, "user.go")
		teamGo := filepath.Join(dir, "team.go")
		require.NoError(t, os.WriteFile(userGo, []byte("package p\n\ntype User struct{}\n"), 0o600))
		require.NoError(t, os.WriteFile(teamGo, []byte("package p\n\ntype Team struct{}\n"), 0o600))

		m := testModelIn(t, dir,
			panelSize(60, 10),
			withProvenance(
				scanner.Provenance{Pointer: "/definitions/User", Pos: token.Position{Filename: userGo, Line: 3}},
				scanner.Provenance{Pointer: "/definitions/Team", Pos: token.Position{Filename: teamGo, Line: 3}},
			),
		)

		m.loadFileQuietly(userGo)
		assert.Equal(t, map[int]bool{3: true}, m.srcIndex.AnchorLines(userGo))

		// Opening another file must swap the anchor set, not keep the old one.
		m.loadFileQuietly(teamGo)
		assert.Equal(t, map[int]bool{3: true}, m.srcIndex.AnchorLines(teamGo))
		assert.Nil(t, m.srcIndex.AnchorLines(filepath.Join(dir, "nope.go")),
			"a file that produced nothing has no anchors")
	})
}

// Against a real scan: the gutter must mark real nodes, and every marked line must genuinely be navigable.
func TestE2E_GutterMarksNavigableLines(t *testing.T) {
	m := scanPetstore(t)

	g := m.specGutter()
	require.NotEmpty(t, g, "a real scan produces both anchors and refs")

	lines := specLines(m)
	var anchors, refs int
	for line, marker := range g {
		require.Less(t, line, len(lines), "marker is inside the document")

		switch marker {
		case panels.GutterAnchor:
			anchors++
			ptr, ok := m.specIndex.PointerAt(line)
			require.True(t, ok, "line %d has a pointer", line)
			_, hasSrc := m.srcIndex.PositionFor(ptr)
			assert.True(t, hasSrc, "anchored line %d (%s) leads to source", line, ptr)
		case panels.GutterRef:
			refs++
			site, ok := m.refIndex.RefAt(line)
			require.True(t, ok, "line %d holds a $ref", line)
			assert.True(t, site.Target.Local, "marked refs are followable")
			_, ok = m.specIndex.LineForPointer(site.Target.Pointer)
			assert.True(t, ok, "line %d resolves to a rendered node", line)
		default:
			t.Fatalf("unexpected marker %q on line %d", marker, line)
		}
	}

	assert.Positive(t, anchors, "the petstore has anchored definitions")
	assert.Positive(t, refs, "and followable refs")
	assert.Less(t, len(g), len(lines),
		"the gutter is a hint, not a mark on every line — otherwise it says nothing")
}

// D2 — find-references cycling and go-to-definition, through the model.
//
// The spec pane carries a real line cursor, so "the node under the cursor" means exactly that.
// Tests park it with SetCursor and assert on CursorLine.

// refModelSpec references /definitions/User from three places and carries one external ref.
//
// Line numbers below are load-bearing.
const refModelSpec = `{
  "definitions": {
    "Team": {
      "properties": {
        "lead": {
          "$ref": "#/definitions/User"
        },
        "logo": {
          "$ref": "https://example.com/logo.json#/Logo"
        },
        "members": {
          "items": {
            "$ref": "#/definitions/User"
          },
          "type": "array"
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
  },
  "paths": {
    "/pets": {
      "get": {
        "responses": {
          "200": {
            "schema": {
              "$ref": "#/definitions/User"
            }
          }
        }
      }
    }
  }
}`

const (
	rmLineLead     = 5
	rmLineLogo     = 8
	rmLineItemsRef = 12
	rmLineUserDecl = 18
	rmLineUserName = 20
	rmLineRespRef  = 32
)

func newRefModel(t *testing.T) *Model {
	t.Helper()
	return testModel(t,
		panelSize(60, 10),
		focusedOn(paneSpec),
		withSpecJSON(refModelSpec),
	)
}
