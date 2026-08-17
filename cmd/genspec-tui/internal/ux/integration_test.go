// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// End-to-end tests: a real scan of the petstore fixture, and a hand-built spec wired to real files on disk.
//
// These deliberately assemble everything - they are what would catch a wiring mistake between the layers the rest of
// the suite tests apart.

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// D3 - the whole chain against a REAL scan.
//
// Every other test in this package feeds the model hand-written JSON.
// That proves the indexes and the navigation agree with each other, but not that either agrees with what codescan
// actually emits: whether $refs arrive bare or allOf-wrapped, whether definition names survive as written, whether the
// provenance pointers line up with the rendered document.
//
// This scans the petstore fixture and drives the finished model over the result.

func TestE2E_RefIndexMatchesTheRenderedSpec(t *testing.T) {
	m := scanPetstore(t)
	lines := specLines(m)

	// The petstore references /definitions/pet from several operations.
	// The exact count is fixture-dependent, so assert the property that matters.
	sites := m.refIndex.RefsToPointer("/definitions/pet")
	require.GreaterOrEqual(t, len(sites), 2, "pet must be referenced from at least two places")

	var last int
	for i, site := range sites {
		require.Less(t, site.Line, len(lines), "site line is inside the document")

		// Every recorded site must really be a $ref pointing where we claim.
		text := lines[site.Line]
		assert.Contains(t, text, `"$ref"`, "line %d", site.Line)
		assert.Contains(t, text, "#/definitions/pet", "line %d", site.Line)

		if i > 0 {
			assert.Greater(t, site.Line, last, "sites are ordered by rendered line")
		}
		last = site.Line

		// And the node holding it must be addressable in the spec index.
		_, ok := m.specIndex.LineForPointer(site.Pointer)
		assert.True(t, ok, "holder %q is a real node", site.Pointer)
	}
}

// codescan emits $refs to responses as well as definitions; both must index.
func TestE2E_ResponseRefsIndex(t *testing.T) {
	m := scanPetstore(t)

	sites := m.refIndex.RefsToPointer("/responses/genericError")
	require.GreaterOrEqual(t, len(sites), 2, "the shared error response is reused across operations")

	for _, site := range sites {
		assert.True(t, site.Target.Local)
		assert.Equal(t, "/responses/genericError", site.Target.Pointer)
	}
}

// The round trip: park on a definition, F3 to one of its uses, Enter to come back.
//
// If either index disagreed with the render, this would land elsewhere.
func TestE2E_CycleThenGoToDefinitionRoundTrips(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	// F3 → the first use.
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	require.Contains(t, m.refs.Status, "of /definitions/pet")
	firstUse := m.spec.TopLine()

	// F3 again → a different use.
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.NotEqual(t, firstUse, m.spec.TopLine(), "the cycle advanced to another site")
	require.Contains(t, m.refs.Status, "ref 2/")

	// Enter on that $ref → back to the definition.
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "→ /definitions/pet", m.notice)
	assert.Contains(t, specLines(m)[defLine], `"pet"`,
		"the line we came back to really is the pet definition")
}

func TestE2E_CycleVisitsEverySiteExactlyOnce(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	want := m.refIndex.RefsToPointer("/definitions/pet")
	require.NotEmpty(t, want)

	seen := make(map[int]int, len(want))
	for range want {
		_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
		seen[m.refs.Site().Line]++
	}

	assert.Len(t, seen, len(want), "one full lap visits every site")
	for line, n := range seen {
		assert.Equal(t, 1, n, "site at line %d visited once", line)
	}

	// One more step wraps back to the start.
	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.Contains(t, m.refs.Status, "ref 1/")
}

// Both renders of the same scan must find the same reference sites - only the line numbers differ.
//
// This is what makes ctrl+j/ctrl+y safe mid-investigation.
func TestE2E_YAMLFindsTheSameSites(t *testing.T) {
	m := scanPetstore(t)
	require.Empty(t, m.scan.YAML, "the scan renders JSON only: YAML is converted when it is asked for")

	jsonHolders := holderSet(m, "/definitions/pet")
	require.NotEmpty(t, jsonHolders)

	switchToYAML(t, m)

	assert.Equal(t, jsonHolders, holderSet(m, "/definitions/pet"),
		"the same nodes reference pet in either render")
}

// The two halves of the linker must agree on a real scan.
//
// A definition the ref index points at should also be a node the provenance index can take to source.
func TestE2E_RefTargetsHaveSource(t *testing.T) {
	m := scanPetstore(t)
	require.Positive(t, m.srcIndex.Len(), "the scan emitted provenance")

	for _, target := range []string{"/definitions/pet", "/definitions/order"} {
		require.NotEmpty(t, m.refIndex.RefsToPointer(target), "%s is referenced", target)

		pos, ok := m.srcIndex.PositionFor(target)
		require.True(t, ok, "%s resolves to source", target)
		assert.True(t, strings.HasSuffix(pos.Filename, ".go"), "%s → %s", target, pos.Filename)
		assert.Positive(t, pos.Line)
	}
}

// Spec to source follow, end to end.
//
// Park on a definition, turn on follow, and the source pane must open the Go file that actually declares it.
func TestE2E_FollowOpensTheDeclaringFile(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	m.toggleFollow(followSpec)

	require.Equal(t, followSpec, m.follow)
	assert.True(t, strings.HasSuffix(m.currentFile, ".go"), "opened %q", m.currentFile)
	assert.Contains(t, m.followTarget, "/definitions/pet")
	assert.Contains(t, m.fileView.Content(), "type Pet struct",
		"the follower landed in the file that declares the type")
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

// Driving from the spec: the source pane mirrors the node under the cursor.
func TestJoin_SpecToSource(t *testing.T) {
	t.Run("lands on the anchored line", func(t *testing.T) {
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
	})

	// The spec has far more nodes than codescan anchors, so most lines resolve via nearest-ancestor.
	//
	// `.../email/type` has no anchor of its own.
	t.Run("nearest ancestor", func(t *testing.T) {
		f := newJoinFixture(t)
		f.m.focused = paneSpec
		f.m.toggleFollow(followSpec)

		f.driveSpec(joinLineEmailType)

		assert.Equal(t, joinSrcEmail-1, f.m.fileView.CurrentLine(),
			"an unanchored child resolves to its nearest anchored ancestor")
		assert.Contains(t, f.m.followTarget, "/definitions/User/properties/email/type",
			"the status names the node under the cursor, not the ancestor it resolved through")
		assert.Contains(t, f.m.followTarget, "user.go:"+strconv.Itoa(joinSrcEmail))
	})

	t.Run("switches file when the target moves", func(t *testing.T) {
		f := newJoinFixture(t)
		f.m.focused = paneSpec
		f.m.spec.SetCursor(joinLineEmail)
		f.m.toggleFollow(followSpec)
		require.Equal(t, f.userGo, f.m.currentFile)

		f.driveSpec(joinLineCity)

		assert.Equal(t, f.addrGo, f.m.currentFile, "the follower reopened the other file")
		assert.Equal(t, joinSrcCity-1, f.m.fileView.CurrentLine())
	})

	// Walking the driver must re-mirror the follower each time, not just on entry.
	t.Run("tracks the driver", func(t *testing.T) {
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
	})
}

// Driving from the source: the spec pane mirrors the node the current line produced.
func TestJoin_SourceToSpec(t *testing.T) {
	t.Run("lands on the produced node", func(t *testing.T) {
		f := newJoinFixture(t)
		f.m.loadFileQuietly(f.userGo)
		f.m.focused, f.m.leftMode = paneTree, modeView
		f.m.fileView.GotoLine(joinSrcEmail - 1)

		f.m.toggleFollow(followSource)

		require.Equal(t, followSource, f.m.follow)
		assert.Equal(t, "/definitions/User/properties/email", f.m.followTarget)
		assert.Equal(t, joinLineEmail, f.m.spec.CursorLine(),
			"the spec follower centred on the produced node")
	})

	// A source line between anchors resolves to the nearest anchor at or above it.
	t.Run("nearest enclosing", func(t *testing.T) {
		f := newJoinFixture(t)
		f.m.loadFileQuietly(f.userGo)
		f.m.focused, f.m.leftMode = paneTree, modeView
		f.m.toggleFollow(followSource)

		f.driveSource(joinSrcUserClose) // the struct's closing brace: no anchor

		assert.Equal(t, "/definitions/User/properties/manager", f.m.followTarget,
			"an unanchored line resolves to the nearest enclosing anchor")
		assert.Equal(t, joinLineManager, f.m.spec.CursorLine())
	})

	t.Run("tracks the driver", func(t *testing.T) {
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
	})

	// A source line ABOVE every anchor in the file has no enclosing node.
	//
	// The follower must hold rather than snap to something arbitrary.
	t.Run("above first anchor holds", func(t *testing.T) {
		f := newJoinFixture(t)
		f.m.loadFileQuietly(f.userGo)
		f.m.focused, f.m.leftMode = paneTree, modeView
		f.m.fileView.GotoLine(joinSrcEmail - 1)
		f.m.toggleFollow(followSource)
		before := f.m.spec.TopLine()

		f.driveSource(1) // `package models`, above the first anchor

		assert.Equal(t, noAnchorDesc, f.m.followTarget)
		assert.Equal(t, before, f.m.spec.TopLine(), "the follower held its position")
	})
}

// A rescan rebuilds both indexes, and follow must re-resolve against the new spec.
//
// Keeping a line number from the old render would point somewhere arbitrary.
func TestJoin_FollowSurvivesRescan(t *testing.T) {
	f := newJoinFixture(t)
	f.m.loadFileQuietly(f.userGo)
	f.m.focused, f.m.leftMode = paneTree, modeView
	f.m.fileView.GotoLine(joinSrcEmail - 1)
	f.m.toggleFollow(followSource)
	require.Equal(t, joinLineEmail, f.m.spec.CursorLine())

	// The same spec with a definition inserted ABOVE User, so every User node shifts down.
	// A stale line number would now point at the wrong node.
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

	_, _ = f.m.Update(scan.ResultMsg{
		JSON: grown,
		Provenance: []scanner.Provenance{
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

	// Spec-driven, with the left pane back on the tree.
	// `/` and `o` are only reachable from here: handleViewerKey swallows every key it does not own, so neither reaches the
	// global bindings while a file is open.
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
		_, handled := f.m.handleSearchControl(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		require.True(t, handled)
		assert.Equal(t, followOff, f.m.follow, "search takes over the spec pane")
	})

	t.Run("opening options", func(t *testing.T) {
		f := specDriven(t)
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		require.True(t, f.m.options.IsOpen())
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

// The read-only viewer shadows only the keys it owns; everything else reaches the global bindings.
//
// Reading source is exactly when you want to rescan or flip JSON↔YAML, and those used to be dead for as long as a
// file was open.
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
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		assert.True(t, f.m.search.Active())
	})

	t.Run("o opens the options popup", func(t *testing.T) {
		f := viewing(t)
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		assert.True(t, f.m.options.IsOpen())
	})

	t.Run("format toggle works while reading source", func(t *testing.T) {
		f := viewing(t)
		f.m.scan.YAML = "definitions: {}\n"
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlY})
		assert.Equal(t, "YAML", f.m.spec.Format())
	})

	t.Run("r triggers a rescan", func(t *testing.T) {
		f := viewing(t)
		cmd := f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		assert.NotNil(t, cmd, "a scan command was issued")
		assert.True(t, f.m.scan.Running)
	})

	// ...but the keys the viewer owns still belong to it.
	t.Run("j still moves the nav line", func(t *testing.T) {
		f := viewing(t)
		before := f.m.fileView.CurrentLine()
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		assert.Equal(t, before+1, f.m.fileView.CurrentLine())
		assert.False(t, f.m.search.Active())
	})

	t.Run("esc returns to the tree", func(t *testing.T) {
		f := viewing(t)
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, modeBrowse, f.m.leftMode)
	})

	// Tab / c / ctrl+q were duplicated in both handlers; the global copies now serve the viewer too, and must behave
	// identically.
	t.Run("tab still changes focus", func(t *testing.T) {
		f := viewing(t)
		_ = f.m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, paneSpec, f.m.focused)
	})
}

// The JOIN, exercised through the model.
//
// The two indexes have their own unit tests; what those cannot show is whether the model wires them together correctly.
// So these tests synthesize a scan - a rendered spec body plus the []scanner.Provenance a real scan would emit
// - build the indexes with the REAL builders, put the source files on disk, and then assert where the follower actually
// lands.
//
// The provenance set deliberately anchors only definitions and properties, the way codescan does (anchors-only
// emission): no anchor on `.../email/type`, none on a struct's closing brace.
// That is what makes nearest-ancestor (spec→source) and nearest-enclosing (source→spec) resolution observable
// rather than incidental.

// joinSpecJSON is what the spec pane renders.
//
// Line numbers are load-bearing - see joinLine* below.
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
	joinLineEmailType  = 12 // NOT anchored - resolves up to the email property
	joinLineManager    = 14
	joinLineManagerRef = 15 // NOT anchored - resolves up to the manager property
)

// The synthesized source.
//
// Anchors point at the 1-based lines noted alongside.
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
	joinSrcUserClose   = 7 // NOT anchored - resolves back to the manager field
	joinSrcAddressDecl = 4
	joinSrcCity        = 5
)

type joinFixture struct {
	m        *Model
	userGo   string
	addrGo   string
	specLine func(ptr string) int
}

// newJoinFixture writes the source files to a temp dir and wires the indexes over them.
//
// The spec index is built from the rendered bytes, and the provenance is the one a scan would emit.
func newJoinFixture(t *testing.T) joinFixture {
	t.Helper()

	dir := t.TempDir()
	userGo := filepath.Join(dir, "user.go")
	addrGo := filepath.Join(dir, "address.go")
	require.NoError(t, os.WriteFile(userGo, []byte(joinUserGo), 0o600))
	require.NoError(t, os.WriteFile(addrGo, []byte(joinAddressGo), 0o600))

	// withSpecJSON builds the real SpecIndex from the rendered bytes, so the line numbers below are the ones a user
	// would actually be looking at.
	m := testModelIn(t, dir,
		panelSize(60, 10),
		withSpecJSON(joinSpecJSON),
		withProvenance(
			scanner.Provenance{Pointer: "/definitions/User", Pos: token.Position{Filename: userGo, Line: joinSrcUserDecl}},
			scanner.Provenance{Pointer: "/definitions/User/properties/email", Pos: token.Position{Filename: userGo, Line: joinSrcEmail}},
			scanner.Provenance{Pointer: "/definitions/User/properties/manager", Pos: token.Position{Filename: userGo, Line: joinSrcManager}},
			scanner.Provenance{Pointer: "/definitions/Address", Pos: token.Position{Filename: addrGo, Line: joinSrcAddressDecl}},
			scanner.Provenance{Pointer: "/definitions/Address/properties/city", Pos: token.Position{Filename: addrGo, Line: joinSrcCity}},
		),
	)

	return joinFixture{
		m: m, userGo: userGo, addrGo: addrGo,
		specLine: func(ptr string) int {
			line, ok := m.specIndex.LineForPointer(ptr)
			require.True(t, ok, "pointer %q must be in the rendered spec", ptr)
			return line
		},
	}
}

// withDiagnostics puts diagnostics in the pane and gives it the keyboard.
//
// The fixture can then be driven from the diag side as well as the spec and source ones.
func (f joinFixture) withDiagnostics(t *testing.T, diags ...grammar.Diagnostic) joinFixture {
	t.Helper()

	f.m.diag.SetSize(80, 8)
	f.m.diagH = 8
	f.m.scan.Diags = diags
	f.m.refreshDiagnostics()
	f.m.focused = paneDiag

	return f
}

// selectDiag moves the diagnostic selection and re-mirrors the followers, as the Update loop does.
func (f joinFixture) selectDiag(delta int) {
	f.m.moveDiagCursor(delta)
	f.m.syncFollowIfActive()
}

// TestJoin_DiagFollowDrivesBothPanes pins that the diagnostics driver mirrors the source AND the spec.
//
// A diagnostic names a place in the source, but what it says is usually about what that place produced - so unlike the
// spec- and source-driven modes, which each drive their opposite, this one drives both.
func TestJoin_DiagFollowDrivesBothPanes(t *testing.T) {
	f := newJoinFixture(t)
	f = f.withDiagnostics(t,
		grammar.Warnf(pos(f.userGo, joinSrcEmail, 2), grammar.CodeInvalidNumber, "about the email field"),
		grammar.Warnf(pos(f.addrGo, joinSrcCity, 2), grammar.CodeInvalidNumber, "about the city field"),
	)

	f.m.toggleFollow(followDiag)

	require.Equal(t, paneDiag, f.m.focused, "the diagnostics pane stays the driver")
	assert.Equal(t, f.userGo, f.m.currentFile)
	assert.Equal(t, joinSrcEmail-1, f.m.fileView.CurrentLine(), "source follower on the diagnostic's line")
	assert.Equal(t, f.specLine("/definitions/User/properties/email"), f.m.spec.CursorLine(),
		"spec follower on the node that line produced")
	assert.Contains(t, f.m.followTarget, "/definitions/User/properties/email")

	// Moving the selection re-drives both halves, across a change of file.
	f.selectDiag(+1)

	assert.Equal(t, f.addrGo, f.m.currentFile)
	assert.Equal(t, joinSrcCity-1, f.m.fileView.CurrentLine())
	assert.Equal(t, f.specLine("/definitions/Address/properties/city"), f.m.spec.CursorLine())
	assert.Contains(t, f.m.followTarget, "/definitions/Address/properties/city")
}

// TestJoin_DiagFollowHalvesResolveIndependently pins that one half missing does not suppress the other.
//
// A diagnostic on a line that produced nothing is the ordinary case:
// a parse error means nothing was built from it.
// So the source half must still resolve, and must still be reported.
func TestJoin_DiagFollowHalvesResolveIndependently(t *testing.T) {
	f := newJoinFixture(t)
	f = f.withDiagnostics(t,
		// Line 1 is the package clause: above every anchor in the file, so nothing was produced at or before it.
		grammar.Errorf(pos(f.userGo, 1, 1), grammar.CodeInvalidNumber, "nothing here produced a node"),
	)
	before := f.m.spec.CursorLine()

	f.m.toggleFollow(followDiag)

	assert.Equal(t, f.userGo, f.m.currentFile, "the source half resolves")
	assert.Equal(t, 0, f.m.fileView.CurrentLine())
	assert.Equal(t, before, f.m.spec.CursorLine(), "the spec follower holds position rather than guessing")
	assert.Contains(t, f.m.followTarget, "user.go:1", "the source half is still reported")
	assert.Contains(t, f.m.followTarget, noAnchorDesc, "and the spec half names which miss this is")
}

// driveSpec moves the spec cursor to line (what driveSpecToSource reads) and re-mirrors the follower.
func (f joinFixture) driveSpec(line int) {
	f.m.spec.SetCursor(line)
	f.m.syncFollowIfActive()
}

// driveSource moves the source nav cursor to a 1-based source line and re-mirrors the follower.
func (f joinFixture) driveSource(srcLine int) {
	f.m.fileView.GotoLine(srcLine - 1)
	f.m.syncFollowIfActive()
}

// fixturesDir resolves the repo-level fixtures directory from this file's own location.
//
// That lets the test run from any working directory. CI runs it from cmd/genspec-tui rather than the repo root.
//
// Deliberately local rather than borrowing scantest.FixturesDir - the TUI module should not grow a dependency on the
// library's test helpers for one path join.
func fixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve the caller's file path")

	// thisFile is <repo>/cmd/genspec-tui/internal/ux/model_e2e_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "testdata"))
}

// petstoreScan caches the scan across the whole package.
//
// A real scan means a real packages.Load, which costs seconds under -race; running one per test took this package from
// ~1s to ~36s. The result is immutable, and each test still gets its own Model built from it.
// Mirrors the caching the library's own scantest helpers do for the same reason.
var (
	petstoreOnce sync.Once      //nolint:gochecknoglobals // test-only scan cache
	petstoreRes  scan.ResultMsg //nolint:gochecknoglobals // test-only scan cache
)

// scanPetstore hands the cached scan to a fresh Model through the same message the bubbletea loop delivers.
func scanPetstore(t *testing.T) *Model {
	t.Helper()

	petstoreOnce.Do(func() {
		petstoreRes = scan.Do(testutils.ApplyLoader(&codescan.Options{
			WorkDir:    fixturesDir(t),
			Packages:   []string{"./goparsing/petstore/..."},
			ScanModels: true,
		}), &scan.Profiling{})
	})
	res := petstoreRes
	require.NoError(t, res.Err, "the petstore fixture must scan cleanly")
	require.NotEmpty(t, res.JSON)

	// The whole result goes in through Update, exactly as the running program receives it.
	m := testModelIn(t, fixturesDir(t), panelSize(100, 30), focusedOn(paneSpec))
	_, _ = m.Update(res)

	return m
}

// switchToYAML flips the spec pane to YAML the way the runtime does: the toggle hands back a command, and the message
// it produces comes back in through Update. The conversion is on demand, so nothing is rendered until it lands.
func switchToYAML(t *testing.T, m *Model) {
	t.Helper()

	cmd := m.setSpecFormat("YAML")
	require.NotNil(t, cmd, "the first switch of a session renders the document")
	_, _ = m.Update(cmd())

	require.Equal(t, "YAML", m.spec.Format())
	require.NotEmpty(t, m.scan.YAML)
}

// specLines is the rendered document the indexes were built from.
func specLines(m *Model) []string { return strings.Split(m.scan.JSON, "\n") }

func holderSet(m *Model, target string) map[string]bool {
	out := make(map[string]bool)
	for _, site := range m.refIndex.RefsToPointer(target) {
		out[site.Pointer] = true
	}

	return out
}
