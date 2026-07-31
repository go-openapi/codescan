// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// S13 — the link gutter (design §6.5): which lines actually lead somewhere.

func TestGutter_MarksAnchorsAndRefs(t *testing.T) {
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
}

// An external $ref is not followable, so marking it would promise a jump that
// Enter cannot make.
func TestGutter_ExternalRefsAreNotMarked(t *testing.T) {
	m := newRefModel(t)
	m.srcIndex = index.BuildSourceIndex(nil)
	m.rebuildGutters()

	_, marked := m.specGutter()[rmLineLogo]
	assert.False(t, marked, "the external ref line carries no marker")
}

// Nothing to say means no gutter at all, so the pane keeps its full width.
func TestGutter_NilWhenNothingLinks(t *testing.T) {
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.spec.SetSize(60, 10)
	m.specJSON = `{"swagger":"2.0"}`
	m.refreshSpec()

	assert.Nil(t, m.specGutter(), "no provenance and no refs → no gutter")
}

// Only nodes with an anchor of their OWN are marked. Marking everything that
// merely resolves through an ancestor would dot nearly every line.
func TestGutter_OnlyExactAnchors(t *testing.T) {
	m := newRefModel(t)
	m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: token.Position{Filename: "user.go", Line: 3}},
	})
	m.rebuildGutters()

	g := m.specGutter()
	require.Equal(t, panels.GutterAnchor, g[rmLineUserDecl])

	// The name property resolves to User by nearest-ancestor, but has no anchor
	// of its own, so it stays unmarked.
	_, marked := g[rmLineUserName]
	assert.False(t, marked,
		"a node that only resolves through an ancestor is not marked")

	// Sanity: it really does resolve, which is what makes the distinction real.
	_, ok := m.srcIndex.PositionFor("/definitions/User/properties/name")
	require.True(t, ok)
}

// The gutter holds rendered line numbers, so a format toggle must rebuild it.
func TestGutter_RebuiltOnFormatToggle(t *testing.T) {
	m := newRefModel(t)
	m.specYAML = "definitions:\n  Team:\n    properties:\n      lead:\n        $ref: '#/definitions/User'\n  User: {}\n"
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
}

func TestGutter_SourceAnchorsFollowTheOpenFile(t *testing.T) {
	dir := t.TempDir()
	userGo := filepath.Join(dir, "user.go")
	teamGo := filepath.Join(dir, "team.go")
	require.NoError(t, os.WriteFile(userGo, []byte("package p\n\ntype User struct{}\n"), 0o600))
	require.NoError(t, os.WriteFile(teamGo, []byte("package p\n\ntype Team struct{}\n"), 0o600))

	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.cfg.WorkDir = dir
	m.spec.SetSize(60, 10)
	m.fileView.SetSize(60, 10)
	m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: token.Position{Filename: userGo, Line: 3}},
		{Pointer: "/definitions/Team", Pos: token.Position{Filename: teamGo, Line: 3}},
	})

	m.loadFileQuietly(userGo)
	assert.Equal(t, map[int]bool{3: true}, m.srcIndex.AnchorLines(userGo))

	// Opening another file must swap the anchor set, not keep the old one.
	m.loadFileQuietly(teamGo)
	assert.Equal(t, map[int]bool{3: true}, m.srcIndex.AnchorLines(teamGo))
	assert.Nil(t, m.srcIndex.AnchorLines(filepath.Join(dir, "nope.go")),
		"a file that produced nothing has no anchors")
}

// Against a real scan: the gutter must mark real nodes, and every marked line
// must genuinely be navigable.
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
