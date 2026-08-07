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

// TestValidation_UnlocatableFindingHoldsPosition pins the honest miss, matching how the other followers treat theirs.
func TestValidation_UnlocatableFindingHoldsPosition(t *testing.T) {
	m := validationModel(t)
	m.validation = ValidationState{
		Ran:      true,
		Findings: []validation.Finding{{Severity: grammar.SeverityError, Message: "something global"}},
	}
	m.diagTab = tabValidation
	before := m.spec.CursorLine()

	target, ok := m.validationTarget()

	assert.False(t, ok)
	assert.Contains(t, target, "names no location")
	assert.Equal(t, before, m.spec.CursorLine(), "the spec follower stays put rather than guessing")
}

// TestValidation_PointerResolutionAccuracy records how precisely findings actually locate.
//
// Measured against the validator rather than reasoned about.
//
// The conversion from the validator's dotted notation is exact in the ordinary case - a deep definition path lands on
// the node itself. Two things cost precision, and neither is the notation:
//
//   - the validator omits ARRAY INDICES, so a finding about one parameter reports ...parameters.type where the node is
//     at .../parameters/0/type. It therefore lands on the array;
//   - a "required but missing" finding names a node that by definition is not there, so its parent is the only honest
//     landing.
//
// Both degrade to the enclosing node, which is why the walk-up exists. This test exists so a change in either direction
// is noticed.
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

	// The headline claim: the notation is usable, not merely lossy.
	assert.Positive(t, exact, "no finding located exactly; the conversion would be useless")
	t.Logf("exact=%d via-ancestor=%d unlocated=%d", exact, viaAncestor, unlocated)

	// The deep definition path is the one that must be exact - nothing about it is array-shaped or absent.
	m.validation = ValidationState{Ran: true, Findings: findings}
	for i, f := range findings {
		if f.Pointer != "/definitions/User/properties/email/type" {
			continue
		}
		m.validation.Cursor = i
		landed, ok := m.validationTarget()
		require.True(t, ok)
		assert.Equal(t, f.Pointer, landed, "a plain object path must land on the node itself")

		return
	}
	t.Fatal("the fixture produced no definition-path finding; the assertion above checked nothing")
}
