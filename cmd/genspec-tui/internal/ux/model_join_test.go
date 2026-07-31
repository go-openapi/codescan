// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/index"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// C8 — the JOIN, exercised through the model.
//
// The two indexes have their own unit tests; what those cannot show is whether
// the model wires them together correctly. So these tests synthesize a scan —
// a rendered spec body plus the []scanner.Provenance a real scan would emit —
// build the indexes with the REAL builders, put the source files on disk, and
// then assert where the follower actually lands.
//
// The provenance set deliberately anchors only definitions and properties, the
// way codescan does (anchors-only emission, design §3.4): no anchor on
// `…/email/type`, none on a struct's closing brace. That is what makes
// nearest-ancestor (spec→source) and nearest-enclosing (source→spec) resolution
// observable rather than incidental.

// joinSpecJSON is what the spec pane renders. Line numbers are load-bearing —
// see joinLine* below.
const joinSpecJSON = `{
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
        "email": {
          "type": "string"
        },
        "manager": {
          "$ref": "#/definitions/User"
        }
      }
    }
  }
}`

// 0-based rendered lines of the nodes these tests navigate to.
const (
	joinLineAddress    = 2
	joinLineCity       = 4
	joinLineUser       = 9
	joinLineEmail      = 11
	joinLineEmailType  = 12 // NOT anchored — resolves up to the email property
	joinLineManager    = 14
	joinLineManagerRef = 15 // NOT anchored — resolves up to the manager property
)

// The synthesized source. Anchors point at the 1-based lines noted alongside.
const (
	joinUserGo = `package models

// User is a user.
type User struct {
	Email   string
	Manager *User
}
`
	joinAddressGo = `package models

// Address is an address.
type Address struct {
	City string
}
`
)

// 1-based source lines, matching the files above.
const (
	joinSrcUserDecl    = 4
	joinSrcEmail       = 5
	joinSrcManager     = 6
	joinSrcUserClose   = 7 // NOT anchored — resolves back to the manager field
	joinSrcAddressDecl = 4
	joinSrcCity        = 5
)

type joinFixture struct {
	m        *Model
	userGo   string
	addrGo   string
	specLine func(ptr string) int
}

// newJoinFixture writes the source files to a temp dir, builds the real spec
// index from the rendered bytes, and installs the provenance a scan would emit.
func newJoinFixture(t *testing.T) joinFixture {
	t.Helper()

	dir := t.TempDir()
	userGo := filepath.Join(dir, "user.go")
	addrGo := filepath.Join(dir, "address.go")
	require.NoError(t, os.WriteFile(userGo, []byte(joinUserGo), 0o600))
	require.NoError(t, os.WriteFile(addrGo, []byte(joinAddressGo), 0o600))

	// searchInput must be a real textinput: a zero-value one panics on Focus,
	// and production always builds it in New().
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView(), searchInput: textinput.New()}
	m.cfg.WorkDir = dir
	m.spec.SetSize(60, 10)
	m.fileView.SetSize(60, 10)
	m.specJSON = joinSpecJSON
	m.refreshSpec() // builds the real SpecIndex from the rendered bytes

	m.srcIndex = index.BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: token.Position{Filename: userGo, Line: joinSrcUserDecl}},
		{Pointer: "/definitions/User/properties/email", Pos: token.Position{Filename: userGo, Line: joinSrcEmail}},
		{Pointer: "/definitions/User/properties/manager", Pos: token.Position{Filename: userGo, Line: joinSrcManager}},
		{Pointer: "/definitions/Address", Pos: token.Position{Filename: addrGo, Line: joinSrcAddressDecl}},
		{Pointer: "/definitions/Address/properties/city", Pos: token.Position{Filename: addrGo, Line: joinSrcCity}},
	})

	return joinFixture{
		m: m, userGo: userGo, addrGo: addrGo,
		specLine: func(ptr string) int {
			line, ok := m.specIndex.LineForPointer(ptr)
			require.True(t, ok, "pointer %q must be in the rendered spec", ptr)
			return line
		},
	}
}

// driveSpec moves the spec cursor to `line` (what driveSpecToSource reads) and
// re-mirrors the follower.
func (f joinFixture) driveSpec(line int) {
	f.m.spec.SetCursor(line)
	f.m.syncFollowIfActive()
}

// driveSource moves the source nav cursor to a 1-based source line and
// re-mirrors the follower.
func (f joinFixture) driveSource(srcLine int) {
	f.m.fileView.GotoLine(srcLine - 1)
	f.m.syncFollowIfActive()
}

func TestJoin_SpecIndexMatchesFixture(t *testing.T) {
	f := newJoinFixture(t)

	// Guards the hand-counted line constants the rest of the file relies on.
	for ptr, want := range map[string]int{
		"/definitions/Address":                      joinLineAddress,
		"/definitions/Address/properties/city":      joinLineCity,
		"/definitions/User":                         joinLineUser,
		"/definitions/User/properties/email":        joinLineEmail,
		"/definitions/User/properties/email/type":   joinLineEmailType,
		"/definitions/User/properties/manager":      joinLineManager,
		"/definitions/User/properties/manager/$ref": joinLineManagerRef,
	} {
		assert.Equal(t, want, f.specLine(ptr), "rendered line of %s", ptr)
	}
}

func TestJoin_SpecToSource_LandsOnTheAnchoredLine(t *testing.T) {
	f := newJoinFixture(t)
	f.m.focused = paneSpec
	f.m.spec.SetCursor(joinLineEmail)

	f.m.toggleFollow(followSpec)

	require.Equal(t, followSpec, f.m.follow)
	assert.Equal(t, f.userGo, f.m.currentFile, "the follower opened the producing file")
	assert.Equal(t, joinSrcEmail-1, f.m.fileView.CurrentLine(),
		"the follower parked on the email field's source line")
	assert.Equal(t, paneSpec, f.m.focused, "the driver keeps focus")
	assert.Contains(t, f.m.followTarget, "user.go:"+strconv.Itoa(joinSrcEmail))
}

// The spec has far more nodes than codescan anchors, so most lines resolve via
// nearest-ancestor. `…/email/type` has no anchor of its own.
func TestJoin_SpecToSource_NearestAncestor(t *testing.T) {
	f := newJoinFixture(t)
	f.m.focused = paneSpec
	f.m.toggleFollow(followSpec)

	f.driveSpec(joinLineEmailType)

	assert.Equal(t, joinSrcEmail-1, f.m.fileView.CurrentLine(),
		"an unanchored child resolves to its nearest anchored ancestor")
	assert.Contains(t, f.m.followTarget, "/definitions/User/properties/email/type",
		"the status names the node under the cursor, not the ancestor it resolved through")
	assert.Contains(t, f.m.followTarget, "user.go:"+strconv.Itoa(joinSrcEmail))
}

func TestJoin_SpecToSource_SwitchesFileWhenTheTargetMoves(t *testing.T) {
	f := newJoinFixture(t)
	f.m.focused = paneSpec
	f.m.spec.SetCursor(joinLineEmail)
	f.m.toggleFollow(followSpec)
	require.Equal(t, f.userGo, f.m.currentFile)

	f.driveSpec(joinLineCity)

	assert.Equal(t, f.addrGo, f.m.currentFile, "the follower reopened the other file")
	assert.Equal(t, joinSrcCity-1, f.m.fileView.CurrentLine())
}

// Walking the driver must re-mirror the follower each time, not just on entry.
func TestJoin_SpecToSource_TracksTheDriver(t *testing.T) {
	f := newJoinFixture(t)
	f.m.focused = paneSpec
	f.m.toggleFollow(followSpec)

	for _, step := range []struct {
		specLine    int
		wantSrcLine int
		wantFile    string
	}{
		{joinLineAddress, joinSrcAddressDecl, f.addrGo},
		{joinLineCity, joinSrcCity, f.addrGo},
		{joinLineUser, joinSrcUserDecl, f.userGo},
		{joinLineEmail, joinSrcEmail, f.userGo},
		{joinLineManager, joinSrcManager, f.userGo},
		{joinLineManagerRef, joinSrcManager, f.userGo}, // unanchored → manager
	} {
		f.driveSpec(step.specLine)
		assert.Equal(t, step.wantFile, f.m.currentFile, "spec line %d", step.specLine)
		assert.Equal(t, step.wantSrcLine-1, f.m.fileView.CurrentLine(), "spec line %d", step.specLine)
	}
}

func TestJoin_SourceToSpec_LandsOnTheProducedNode(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.fileView.GotoLine(joinSrcEmail - 1)

	f.m.toggleFollow(followSource)

	require.Equal(t, followSource, f.m.follow)
	assert.Equal(t, "/definitions/User/properties/email", f.m.followTarget)
	assert.Equal(t, joinLineEmail, f.m.spec.CursorLine(),
		"the spec follower centred on the produced node")
}

// A source line between anchors resolves to the nearest anchor at or above it.
func TestJoin_SourceToSpec_NearestEnclosing(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.toggleFollow(followSource)

	f.driveSource(joinSrcUserClose) // the struct's closing brace: no anchor

	assert.Equal(t, "/definitions/User/properties/manager", f.m.followTarget,
		"an unanchored line resolves to the nearest enclosing anchor")
	assert.Equal(t, joinLineManager, f.m.spec.CursorLine())
}

func TestJoin_SourceToSpec_TracksTheDriver(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.toggleFollow(followSource)

	for _, step := range []struct {
		srcLine  int
		wantPtr  string
		wantSpec int
	}{
		{joinSrcUserDecl, "/definitions/User", joinLineUser},
		{joinSrcEmail, "/definitions/User/properties/email", joinLineEmail},
		{joinSrcManager, "/definitions/User/properties/manager", joinLineManager},
		{joinSrcUserClose, "/definitions/User/properties/manager", joinLineManager},
	} {
		f.driveSource(step.srcLine)
		assert.Equal(t, step.wantPtr, f.m.followTarget, "source line %d", step.srcLine)
		assert.Equal(t, step.wantSpec, f.m.spec.CursorLine(), "source line %d", step.srcLine)
	}
}

// A source line ABOVE every anchor in the file has no enclosing node. The
// follower must hold rather than snap to something arbitrary.
func TestJoin_SourceToSpec_AboveFirstAnchorHolds(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.fileView.GotoLine(joinSrcEmail - 1)
	f.m.toggleFollow(followSource)
	before := f.m.spec.TopLine()

	f.driveSource(1) // `package models`, above the first anchor

	assert.Equal(t, noAnchorDesc, f.m.followTarget)
	assert.Equal(t, before, f.m.spec.TopLine(), "the follower held its position")
}

// A rescan rebuilds both indexes; follow must re-resolve against the NEW spec
// rather than keep pointing at a line number from the old render.
func TestJoin_FollowSurvivesRescan(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.fileView.GotoLine(joinSrcEmail - 1)
	f.m.toggleFollow(followSource)
	require.Equal(t, joinLineEmail, f.m.spec.CursorLine())

	// The same spec with a definition inserted ABOVE User, so every User node
	// shifts down. A stale line number would now point at the wrong node.
	grown := `{
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
        "email": {
          "type": "string"
        },
        "manager": {
          "$ref": "#/definitions/User"
        }
      }
    }
  }
}`

	_, _ = f.m.Update(scanResultMsg{
		json: grown,
		provenance: []scanner.Provenance{
			{Pointer: "/definitions/User/properties/email", Pos: token.Position{Filename: f.userGo, Line: joinSrcEmail}},
		},
	})

	newLine, ok := f.m.specIndex.LineForPointer("/definitions/User/properties/email")
	require.True(t, ok)
	require.NotEqual(t, joinLineEmail, newLine, "precondition: the node moved in the new render")

	assert.Equal(t, "/definitions/User/properties/email", f.m.followTarget)
	assert.Equal(t, newLine, f.m.spec.CursorLine(),
		"follow re-resolved against the rebuilt index")
}

func TestJoin_FollowExits(t *testing.T) {
	// Source-driven: the file viewer is the driver.
	sourceDriven := func(t *testing.T) joinFixture {
		t.Helper()
		f := newJoinFixture(t)
		f.m.loadFileQuietly(f.userGo)
		f.m.focused, f.m.leftMode = paneTree, modeView
		f.m.fileView.GotoLine(joinSrcEmail - 1)
		f.m.toggleFollow(followSource)
		require.Equal(t, followSource, f.m.follow)
		return f
	}

	// Spec-driven, with the left pane back on the tree. `/` and `o` are only
	// reachable from here: handleViewerKey swallows every key it does not own,
	// so neither reaches the global bindings while a file is open.
	specDriven := func(t *testing.T) joinFixture {
		t.Helper()
		f := newJoinFixture(t)
		f.m.focused, f.m.leftMode = paneSpec, modeBrowse
		f.m.spec.SetCursor(joinLineEmail)
		f.m.toggleFollow(followSpec)
		require.Equal(t, followSpec, f.m.follow)
		return f
	}

	t.Run("opening search", func(t *testing.T) {
		f := specDriven(t)
		_, _, handled := f.m.handleSearchControl(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		require.True(t, handled)
		assert.Equal(t, followOff, f.m.follow, "search takes over the spec pane")
	})

	t.Run("opening options", func(t *testing.T) {
		f := specDriven(t)
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		require.True(t, f.m.optionsOpen)
		assert.Equal(t, followOff, f.m.follow, "a rescan is about to invalidate the indexes")
	})

	t.Run("starting to edit", func(t *testing.T) {
		f := sourceDriven(t)
		_ = f.m.fileView.StartEdit()
		f.m.syncFollowIfActive()
		assert.Equal(t, followOff, f.m.follow, "positions go stale once the buffer is edited")
	})

	t.Run("second f", func(t *testing.T) {
		f := sourceDriven(t)
		f.m.toggleFollow(followSource)
		assert.Equal(t, followOff, f.m.follow)
		assert.Empty(t, f.m.followTarget)
	})

	t.Run("focus change", func(t *testing.T) {
		f := sourceDriven(t)
		f.m.focused = paneDiag
		f.m.syncFollowIfActive()
		assert.Equal(t, followOff, f.m.follow, "the driver pane lost focus")
	})
}

// The read-only viewer shadows only the keys it owns; everything else reaches
// the global bindings. Reading source is exactly when you want to rescan or
// flip JSON↔YAML, and those used to be dead for as long as a file was open.
func TestJoin_ViewerPassesGlobalKeysThrough(t *testing.T) {
	viewing := func(t *testing.T) joinFixture {
		t.Helper()
		f := newJoinFixture(t)
		f.m.loadFileQuietly(f.userGo)
		f.m.focused, f.m.leftMode = paneTree, modeView
		return f
	}

	t.Run("slash opens search", func(t *testing.T) {
		f := viewing(t)
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		assert.True(t, f.m.searching)
	})

	t.Run("o opens the options popup", func(t *testing.T) {
		f := viewing(t)
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		assert.True(t, f.m.optionsOpen)
	})

	t.Run("format toggle works while reading source", func(t *testing.T) {
		f := viewing(t)
		f.m.specYAML = "definitions: {}\n"
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlY})
		assert.Equal(t, "YAML", f.m.spec.Format())
	})

	t.Run("r triggers a rescan", func(t *testing.T) {
		f := viewing(t)
		_, cmd := f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		assert.NotNil(t, cmd, "a scan command was issued")
		assert.True(t, f.m.scanning)
	})

	// ...but the keys the viewer owns still belong to it.
	t.Run("j still moves the nav line", func(t *testing.T) {
		f := viewing(t)
		before := f.m.fileView.CurrentLine()
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, before+1, f.m.fileView.CurrentLine())
		assert.False(t, f.m.searching)
	})

	t.Run("esc returns to the tree", func(t *testing.T) {
		f := viewing(t)
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, modeBrowse, f.m.leftMode)
	})

	// Tab / c / ctrl+q were duplicated in both handlers; the global copies now
	// serve the viewer too, and must behave identically.
	t.Run("tab still changes focus", func(t *testing.T) {
		f := viewing(t)
		_, _ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, paneSpec, f.m.focused)
	})
}
