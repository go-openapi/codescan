// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/validation"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// invalidSpecJSON has a response with no description, which Swagger 2.0 requires.
const invalidSpecJSON = `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {"operationId": "getPets", "responses": {"200": {}}}
    }
  }
}`

// runValidation drives v through the real dispatch and delivers the resulting message, as the event loop would.
func runValidation(t *testing.T, m *Model) {
	t.Helper()

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	require.NotNil(t, cmd, "v must start a validation")

	msg := cmd()
	vm, ok := msg.(validationMsg)
	require.True(t, ok, "v produced %T, not a validation result", msg)

	_, _ = m.Update(vm)
}

func validationModel(t *testing.T) *Model {
	t.Helper()

	return testModel(t, sized(120, 40), diagSize(120, 10), withSpecJSON(invalidSpecJSON))
}

// TestValidation_TabAppearsOnlyOnceRun pins the tab's lifetime.
//
// It is not a permanent mode advertising itself before anybody asked for it.
func TestValidation_TabAppearsOnlyOnceRun(t *testing.T) {
	m := validationModel(t)

	assert.Equal(t, "diagnostics", m.diagTabTitle(), "no tab strip before a validation has run")
	assert.Equal(t, tabScan, m.diagTab)

	runValidation(t, m)

	assert.Contains(t, m.diagTabTitle(), "validation")
	assert.Equal(t, tabValidation, m.diagTab, "the pane switches to what was just asked for")
	assert.True(t, m.validation.Ran)
	assert.NotEmpty(t, m.validation.Findings, "the fixture is invalid on purpose")
}

// TestValidation_FindingsAreShown pins that the findings reach the pane, not just the model.
func TestValidation_FindingsAreShown(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)

	body := testutils.StripANSI(m.diag.Content())

	assert.Contains(t, body, "finding")
	assert.Contains(t, body, "description", "the missing-description error is listed")
}

// TestValidation_RescanRetiresTheTab is the staleness rule.
//
// The findings judged a document that no longer exists, and every row of such a list invites navigating to a node
// that may have moved or gone.
func TestValidation_RescanRetiresTheTab(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)
	require.Equal(t, tabValidation, m.diagTab)

	_, _ = m.Update(scan.ResultMsg{JSON: invalidSpecJSON})

	assert.Equal(t, tabScan, m.diagTab, "a rescan returns the pane to the scan tab")
	assert.False(t, m.validation.Ran, "and drops the verdict on the superseded document")
	assert.Equal(t, "diagnostics", m.diagTabTitle())
}

// TestValidation_TabToggle pins V, and that it cannot switch to a tab that does not exist.
func TestValidation_TabToggle(t *testing.T) {
	m := validationModel(t)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, tabScan, m.diagTab, "nothing to switch to before a validation has run")

	runValidation(t, m)
	require.Equal(t, tabValidation, m.diagTab)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, tabScan, m.diagTab)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, tabValidation, m.diagTab)
}

// TestValidation_LowercaseVDoesNotSwitchTabs is the case-collision guard.
//
// Bindings are matched case-insensitively, so V would otherwise revalidate instead of switching view.
func TestValidation_LowercaseVDoesNotSwitchTabs(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)
	m.diagTab = tabScan

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	assert.NotNil(t, cmd, "lowercase v revalidates")
	assert.Equal(t, tabScan, m.diagTab, "and does not toggle the tab on its way")
}

// TestValidation_EnterGoesToTheNode pins the tab's own tracking.
//
// A finding names a JSON pointer, so it drives the SPEC pane, not the source pane the scan tab drives.
func TestValidation_EnterGoesToTheNode(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)
	m.focused = paneDiag

	// Select a finding that actually carries a location.
	located := -1
	for i, f := range m.validation.Findings {
		if f.Pointer != "" {
			located = i

			break
		}
	}
	require.GreaterOrEqual(t, located, 0, "the fixture produced no locatable finding")
	m.validation.Cursor = located

	_, handled := m.handleDiagNav(tea.KeyMsg{Type: tea.KeyEnter})

	require.True(t, handled)
	assert.Equal(t, paneSpec, m.focused, "Enter goes to the node, so the spec takes focus")
	assert.Contains(t, m.notice, "→ /", "and reports the pointer it landed on")
}

// TestValidation_FollowDrivesTheSpec pins f on this tab.
func TestValidation_FollowDrivesTheSpec(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)
	m.focused = paneDiag

	_, handled := m.handleDiagNav(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	require.True(t, handled)
	assert.Equal(t, followValidation, m.follow)
	assert.Equal(t, paneDiag, m.focused, "the driver keeps focus")
	assert.Contains(t, testutils.StripANSI(m.followBadge()), "VALIDATION ▸ SPEC")
}

// TestValidation_FollowExitsWithTheTab pins that leaving the tab leaves the mode.
//
// A follower mirroring a selection the user can no longer see is worse than no follower.
func TestValidation_FollowExitsWithTheTab(t *testing.T) {
	m := validationModel(t)
	runValidation(t, m)
	m.focused = paneDiag
	_, _ = m.handleDiagNav(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	require.Equal(t, followValidation, m.follow)

	m.toggleDiagTab()

	assert.Equal(t, followOff, m.follow)
}

// TestValidation_NothingToValidate pins the empty case.
func TestValidation_NothingToValidate(t *testing.T) {
	m := testModel(t, sized(120, 40), diagSize(120, 10))
	require.Empty(t, m.scan.JSON)

	_ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	assert.False(t, m.validation.Ran)
	assert.Contains(t, m.notice, "nothing to validate")
}

// TestValidation_RootPointerGoesToTheTop covers a finding about something the document does not have at all.
//
// The validator reports those at the whole document, which RFC 6901 spells as the EMPTY pointer - so an empty pointer
// is a location, not the absence of one. It used to be read as "nowhere" and left the pane where it was, which for the
// commonest finding of all ("info in body is required") meant refusing to navigate to the one place it could.
func TestValidation_RootPointerGoesToTheTop(t *testing.T) {
	m := validationModel(t)
	m.validation = ValidationState{
		Ran:      true,
		Findings: []validation.Finding{{Severity: grammar.SeverityError, Message: "info in body is required"}},
	}
	m.diagTab = tabValidation
	m.spec.SetCursor(12) // somewhere down the document, so going to the top is observable

	target, ok := m.validationTarget()

	require.True(t, ok, "the whole document is somewhere to go")
	assert.Equal(t, validation.RootLabel, target)
	assert.Equal(t, 0, m.spec.CursorLine(), "the top of the pane is the document")
}

// TestValidation_UnresolvablePointerHoldsPosition pins the honest miss, matching how the other followers treat theirs.
//
// A pointer whose every ancestor is missing too has nothing to walk up to.
func TestValidation_UnresolvablePointerHoldsPosition(t *testing.T) {
	m := validationModel(t)
	m.validation = ValidationState{
		Ran: true,
		Findings: []validation.Finding{
			{Severity: grammar.SeverityError, Pointer: "/nowhere/deep", Message: "about a node this view does not have"},
		},
	}
	m.diagTab = tabValidation
	before := m.spec.CursorLine()

	target, ok := m.validationTarget()

	assert.False(t, ok)
	assert.Contains(t, target, "not rendered")
	assert.Equal(t, before, m.spec.CursorLine(), "the spec follower stays put rather than guessing")
}

// TestValidation_RequiredEntryLocatesExactly covers a finding whose subject is an entry of a `required` array.
//
// It lands on the entry rather than on the definition holding it, which is the text a reader has to go and amend.
//
// One fault per spec, deliberately. The validator walks the definitions MAP and breaks out on the first error, so a
// document with two such faults reports whichever Go's map order reached first - and the second only sometimes. One
// fault each keeps this test about locations instead of about iteration order.
//
// The warning case is only visible at all because warnings are now read off the result that carries them.
func TestValidation_RequiredEntryLocatesExactly(t *testing.T) {
	// requiredSpec builds a document whose Pet definition is reachable from a path, since unreferenced definitions are
	// not checked, with `props` as its properties and one required entry naming `required`.
	requiredSpec := func(required, props string) string {
		return `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "getPets",
        "responses": {"200": {"description": "ok", "schema": {"$ref": "#/definitions/Pet"}}}
      }
    }
  },
  "definitions": {
    "Pet": {"type": "object", "required": ["` + required + `"], "properties": {` + props + `}}
  }
}`
	}

	for _, tc := range []struct {
		name     string
		spec     string
		severity grammar.Severity
	}{
		{
			name:     "a required property that is never declared",
			spec:     requiredSpec("notDeclared", `"name": {"type": "string"}`),
			severity: grammar.SeverityError,
		},
		{
			name:     "a property both required and readOnly",
			spec:     requiredSpec("readOnlyToo", `"readOnlyToo": {"type": "string", "readOnly": true}`),
			severity: grammar.SeverityWarning,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const want = "/definitions/Pet/required/0"

			m := testModel(t, sized(120, 40), diagSize(120, 10), withSpecJSON(tc.spec))
			findings, err := validation.Run([]byte(tc.spec))
			require.NoError(t, err)

			i := indexOfPointer(t, findings, want)
			assert.Equal(t, tc.severity, findings[i].Severity)

			m.validation = ValidationState{Ran: true, Findings: findings, Cursor: i}
			landed, ok := m.validationTarget()
			require.True(t, ok)
			assert.Equal(t, want, landed, "a required entry is a real node, so it must land on itself")
		})
	}
}

// TestValidation_PointerResolutionAccuracy records how precisely findings actually locate.
//
// Measured against the validator rather than reasoned about.
//
// Pointers come from the validator's own record of where it was, so every finding lands on the node it names: a deep
// definition path, an indexed one, and an entry of a required array alike.
//
// A finding whose subject is a node's ABSENCE ("schema in body is required") is reported on the value that should hold
// it, and at the top of the document that is the empty pointer - a location, not the lack of one.
//
// This test exists so a change in either direction is noticed.
func TestValidation_PointerResolutionAccuracy(t *testing.T) {
	const spec = `{
  "swagger": "2.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "getPets",
        "parameters": [{"name": "limit", "in": "query", "type": "nope"}],
        "responses": {"200": {}}
      }
    }
  },
  "definitions": {"User": {"type": "object", "properties": {"email": {"type": "bogus"}}}}
}`

	m := testModel(t, sized(120, 40), diagSize(120, 10), withSpecJSON(spec))
	findings, err := validation.Run([]byte(spec))
	require.NoError(t, err)
	require.NotEmpty(t, findings)

	exact, viaAncestor, unlocated := 0, 0, 0
	for i, f := range findings {
		m.validation = ValidationState{Ran: true, Findings: findings, Cursor: i}

		if f.Pointer == "" {
			unlocated++

			continue
		}
		landed, ok := m.validationTarget()
		require.True(t, ok, "a finding with a pointer must resolve somewhere: %q", f.Pointer)

		if landed == f.Pointer {
			exact++

			continue
		}
		viaAncestor++
		assert.True(t, strings.HasPrefix(f.Pointer, landed),
			"an inexact landing must be an ANCESTOR of what was reported, never a sibling: %q vs %q", f.Pointer, landed)
	}

	// The headline claim, and it is now the strong one: every finding lands on the node it names.
	assert.Positive(t, exact, "no finding located exactly; navigation would be useless")
	assert.Zero(t, viaAncestor,
		"a finding landed on an ancestor: validate >= 0.26.3 addresses a node the document holds, so this reads as an "+
			"upstream change rather than a fault here")
	assert.Zero(t, unlocated, "every finding carries a location, the whole document included")
	t.Logf("exact=%d via-ancestor=%d unlocated=%d", exact, viaAncestor, unlocated)

	// Two shapes must be exact. The deep definition path, which nothing about is array-shaped or absent - and an
	// INDEXED one, which is the shape the old dotted notation could not express at all: it reported
	// paths./pets.get.parameters.type for a node living at .../parameters/0/type, so every finding about a parameter
	// landed on the list.
	for _, want := range []string{
		"/paths/~1pets/get/parameters/0/type",     // indexed
		"/definitions/User/properties/email/type", // deep and plain
	} {
		// require, not a bare loop: a shape that stops being reported would otherwise make this pass by never running.
		i := indexOfPointer(t, findings, want)
		m.validation = ValidationState{Ran: true, Findings: findings, Cursor: i}

		landed, ok := m.validationTarget()
		require.True(t, ok)
		assert.Equal(t, want, landed, "this shape must land on the node itself, not on an ancestor")
	}
}

// indexOfPointer finds the finding carrying ptr, failing the test when nothing does.
func indexOfPointer(t *testing.T, findings []validation.Finding, ptr string) int {
	t.Helper()

	for i, f := range findings {
		if f.Pointer == ptr {
			return i
		}
	}
	require.FailNow(t, "no finding reported "+ptr+"; the test would otherwise assert nothing")

	return -1
}
