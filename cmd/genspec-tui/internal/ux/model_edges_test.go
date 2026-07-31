// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The two renders of the same spec, indexing the same pointers at DIFFERENT
// lines — the whole point of preserving the pointer rather than the line
// across a format toggle. Both renders are deliberately taller than the test
// viewport (below) so neither clamps: YAML is roughly half the height of JSON,
// which is exactly why carrying the raw line number across is wrong.
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

// toggleFixture builds a model holding both renders of the same spec, with the
// spec pane focused and the JSON index live. The pane is short on purpose (a
// 5-line viewport) so both renders can actually scroll to the target node.
func toggleFixture(t *testing.T) *Model {
	t.Helper()
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.spec.SetSize(60, 8)
	m.fileView.SetSize(60, 8)
	m.specJSON, m.specYAML = toggleJSON, toggleYAML
	m.focused = paneSpec
	m.refreshSpec()
	return m
}

func TestSpecFormatToggle_PreservesPointerNotLine(t *testing.T) {
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
}

func TestSpecFormatToggle_SameFormatIsNoop(t *testing.T) {
	m := toggleFixture(t)
	m.spec.SetCursor(emailJSONLine)

	m.setSpecFormat("JSON")

	assert.Equal(t, emailJSONLine, m.spec.CursorLine(),
		"re-selecting the active format must not move the cursor")
}

func TestSpecFormatToggle_UnindexedCursorLine(t *testing.T) {
	m := toggleFixture(t)
	m.specIndex = nil // no index: nothing to preserve, but nothing may panic either

	m.setSpecFormat("YAML")

	assert.Equal(t, "YAML", m.spec.Format())
}

// TestSpecFormatToggle_ViaKey checks the binding actually routes through the
// pointer-preserving path (the bug was in the key handler, not the helper).
func TestSpecFormatToggle_ViaKey(t *testing.T) {
	m := toggleFixture(t)
	m.spec.SetCursor(emailJSONLine)

	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlY})

	assert.Equal(t, "YAML", m.spec.Format())
	assert.Equal(t, emailYAMLLine, m.spec.CursorLine(),
		"ctrl+y must preserve the node under the cursor")
}

func TestLinkSourceToSpec_NamesEachMiss(t *testing.T) {
	const ptr = "/definitions/User"

	t.Run("no file open", func(t *testing.T) {
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		desc, moved := m.linkSourceToSpec()
		assert.False(t, moved)
		assert.Equal(t, noFileDesc, desc)
	})

	t.Run("no provenance at all", func(t *testing.T) {
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		m.currentFile = "user.go"
		m.fileView.SetFile("user.go", "a\nb\nc")
		m.srcIndex = index.BuildSourceIndex(nil)

		desc, moved := m.linkSourceToSpec()
		assert.False(t, moved)
		assert.Equal(t, noProvenanceDesc, desc,
			"an empty index means nothing was anchored — not that this line is special")
	})

	t.Run("anchored file but line above the first anchor", func(t *testing.T) {
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		m.currentFile = "user.go"
		m.fileView.SetFile("user.go", "a\nb\nc")
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: ptr, Pos: token.Position{Filename: "user.go", Line: 3}},
		})
		m.fileView.GotoLine(0) // source line 1

		desc, moved := m.linkSourceToSpec()
		assert.False(t, moved)
		assert.Equal(t, noAnchorDesc, desc)
	})

	t.Run("anchored but not rendered in this view", func(t *testing.T) {
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		m.spec.SetSize(60, 10)
		m.currentFile = "user.go"
		m.fileView.SetFile("user.go", "a\nb\nc")
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: ptr, Pos: token.Position{Filename: "user.go", Line: 1}},
		})
		// The spec index knows a different node entirely, so the pointer
		// resolves on the source side but has nowhere to land.
		m.specIndex = index.NewSpecIndex(
			map[int]string{0: "/definitions/Other"},
			map[string]int{"/definitions/Other": 0},
		)
		m.fileView.GotoLine(0)

		desc, moved := m.linkSourceToSpec()
		assert.False(t, moved)
		assert.Equal(t, ptr+notRenderedSuffix, desc,
			"a node that exists but isn't in this render is a different answer from no source")
	})
}

func TestDriveSpecToSource_NamesEachMiss(t *testing.T) {
	newModel := func() *Model {
		m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
		m.spec.SetSize(60, 10)
		m.spec.SetContent("line0\nline1\nline2")
		m.specIndex = index.NewSpecIndex(
			map[int]string{0: "/definitions/User"},
			map[string]int{"/definitions/User": 0},
		)
		return m
	}

	t.Run("no provenance at all", func(t *testing.T) {
		m := newModel()
		m.srcIndex = index.BuildSourceIndex(nil)

		desc := m.driveSpecToSource()
		assert.Contains(t, desc, noProvenanceDesc)
		assert.Empty(t, m.currentFile, "the follower holds rather than jumping")
	})

	t.Run("spec-only node", func(t *testing.T) {
		m := newModel()
		// Some other node is anchored, so the index is populated — this node
		// simply wasn't produced from code (the InputSpec overlay case, §3.8).
		m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
			{Pointer: "/definitions/Other", Pos: token.Position{Filename: "other.go", Line: 1}},
		})

		desc := m.driveSpecToSource()
		assert.Equal(t, "/definitions/User"+noSourceSuffix, desc)
		assert.Empty(t, m.currentFile, "the follower holds rather than jumping")
	})

	t.Run("no node under the viewport top", func(t *testing.T) {
		m := newModel()
		m.specIndex = nil
		m.srcIndex = index.BuildSourceIndex(nil)

		assert.Equal(t, noNodeDesc, m.driveSpecToSource())
	})
}

func TestFollowBadge_StaleWhileDirty(t *testing.T) {
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.fileView.SetFile("user.go", "package p\n")
	m.follow = followSource
	m.followTarget = "/definitions/User"

	require.False(t, m.stale(), "a freshly loaded buffer matches the last scan")
	assert.NotContains(t, stripANSI(m.followBadge()), "STALE")

	// An unsaved edit shifts every anchor below it: positions are now older
	// than what is on screen.
	m.fileView.StartEdit()
	_ = m.fileView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	require.True(t, m.stale(), "an edited buffer invalidates the recorded positions")
	assert.Contains(t, stripANSI(m.followBadge()), "STALE")

	// Saving (MarkClean is what saveFile does) clears it again.
	m.fileView.MarkClean()
	assert.False(t, m.stale())
	assert.NotContains(t, stripANSI(m.followBadge()), "STALE")
}

// stripANSI removes the SGR escape sequences lipgloss emits, so assertions can
// look at the text a user reads rather than the styling around it.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
