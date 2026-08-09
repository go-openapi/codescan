// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	stderrors "errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestMain forces a colour profile, as ux, ux/panels and ux/diagnostics do.
//
// Without it lipgloss emits plain text under go test and every styling assertion below would pass against a renderer
// that applied none - which is exactly how a selected row's broken highlight went unnoticed in the scan tab.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

const ansiReset = "\x1b[0m"

// errValidatorBroke stands in for the validator failing outright, as opposed to reporting findings.
var errValidatorBroke = stderrors.New("the validator could not run")

func twoFindings() []Finding {
	return []Finding{
		{Severity: grammar.SeverityError, Pointer: "/paths/~1pets/get/responses/200", Message: "description is required"},
		{Severity: grammar.SeverityWarning, Pointer: "/definitions/User", Message: "unused definition"},
	}
}

func TestRender_States(t *testing.T) {
	t.Run("before anything ran", func(t *testing.T) {
		body, line := Render(nil, false, nil, 0, true)

		assert.Contains(t, body, "press v")
		assert.Equal(t, -1, line)
	})

	t.Run("valid spec", func(t *testing.T) {
		body, line := Render(nil, true, nil, 0, true)

		assert.Contains(t, body, "is valid")
		assert.Equal(t, -1, line)
	})

	t.Run("the validator itself failed", func(t *testing.T) {
		body, line := Render(nil, true, errValidatorBroke, 0, true)

		assert.Contains(t, body, "validation failed")
		assert.Equal(t, -1, line)
	})

	t.Run("findings", func(t *testing.T) {
		body, line := Render(twoFindings(), true, nil, 0, true)

		assert.Contains(t, body, "2 findings (1 error, 1 warning)")
		assert.Contains(t, body, "description is required")
		assert.Contains(t, body, "/paths/~1pets/get/responses/200",
			"the row shows the pointer, which is where enter will go")
		assert.Equal(t, 1, line, "the selected row sits under the tally")
	})
}

// TestRender_SelectedRowKeepsItsHighlight is the same regression guard the scan tab carries.
//
// The selection bar is a background opened once per row; any style applied inside it emits a reset that closes the
// background early, and lipgloss does not re-open it - so the bar would stop partway across.
func TestRender_SelectedRowKeepsItsHighlight(t *testing.T) {
	for _, focused := range []bool{true, false} {
		body, line := Render(twoFindings(), true, nil, 0, focused)
		require.GreaterOrEqual(t, line, 0)

		row := strings.Split(body, "\n")[line]

		assert.Equal(t, 1, strings.Count(row, ansiReset),
			"focused=%v: the highlight breaks at each extra reset:\n%q", focused, row)
	}
}

// TestRender_UnselectedRowIsColouredBySeverity pins that severity reaches the message, not just the label.
//
// The point of colouring at all is that the pane can be read for red at a glance.
func TestRender_UnselectedRowIsColouredBySeverity(t *testing.T) {
	body, _ := Render(twoFindings(), true, nil, 0, true) // row 0 selected, so row 1 is styled
	row := strings.Split(body, "\n")[2]

	before, _, found := strings.Cut(row, "unused definition")
	require.True(t, found, "message missing from %q", row)
	assert.True(t, strings.HasSuffix(before, "m") && strings.Contains(before, "\x1b["),
		"the severity colour stops at the label:\n%q", row)
}

// TestRender_SelectedAndPlainRowsCarrySameText guards the styled and plain pair against drifting.
//
// The two are built by different functions.
func TestRender_SelectedAndPlainRowsCarrySameText(t *testing.T) {
	f := twoFindings()[0]

	assert.Equal(t, stripANSI(styledRow(f)), plainRow(f))
}

// TestRender_RootFindingRendersALabel pins that a finding about the whole document renders a row rather than a blank
// column, since the pointer addressing the document is the empty string.
func TestRender_RootFindingRendersALabel(t *testing.T) {
	body, _ := Render([]Finding{{Severity: grammar.SeverityError, Message: "info in body is required"}}, true, nil, -1, true)

	assert.Contains(t, body, RootLabel)
}

// stripANSI removes CSI escapes, leaving the text a user reads.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] == ';' || (s[i] >= '0' && s[i] <= '9')) {
				i++
			}
			if i < len(s) {
				i++
			}

			continue
		}
		b.WriteByte(s[i])
		i++
	}

	return b.String()
}
