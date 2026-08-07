// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// TestRenderDiagnostics checks the three render states: clean, hard-error, and a soft-diagnostic list with a severity
// tally and relative paths.
func TestRenderDiagnostics(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		if got, _ := Render("/work", nil, nil, 0, true); got != "(no diagnostics)" {
			t.Errorf("clean scan: got %q", got)
		}
	})

	t.Run("hard error", func(t *testing.T) {
		got, _ := Render("/work", codescan.ErrCodeScan, nil, 0, true)
		if !strings.Contains(got, "scan failed") || !strings.Contains(got, codescan.ErrCodeScan.Error()) {
			t.Errorf("hard error not surfaced: %q", got)
		}
	})

	t.Run("soft diagnostics", func(t *testing.T) {
		diags := []grammar.Diagnostic{
			grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
			grammar.Warnf(pos("/work/models/a.go", 20, 5), grammar.CodeAmbiguousEmbed, "ambiguous"),
		}
		got, _ := Render("/work", nil, diags, 0, true)

		for _, want := range []string{
			"2 diagnostics (1 error, 1 warning)",
			filepath.FromSlash("models/a.go") + ":12:3", // trimmed to workdir, native separators
			"bad maximum",
			string(grammar.CodeInvalidNumber),
			"error",
			"warning",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("rendered diagnostics missing %q in:\n%s", want, got)
			}
		}
		if strings.Contains(got, filepath.FromSlash("/work/models")) {
			t.Errorf("absolute path leaked into rendered diagnostics:\n%s", got)
		}
	})
}

// TestSelectedRowKeepsItsHighlight is the regression witness for a selected row losing its highlight partway across.
//
// The selection bar is a background colour opened once at the start of the row and closed once at its end. Any style
// applied INSIDE it emits a reset of its own, and a reset clears the background too — lipgloss does not re-open it — so
// a row highlighted over already-coloured text renders its bar only as far as the first coloured run (here, the
// severity label) and then stops.
//
// Counting resets is the falsifiable form of that: exactly one, at the end.
func TestSelectedRowKeepsItsHighlight(t *testing.T) {
	diags := []grammar.Diagnostic{
		grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
	}

	for _, focused := range []bool{true, false} {
		got, line := Render("/work", nil, diags, 0, focused)
		if line < 0 {
			t.Fatalf("focused=%v: selected line not reported", focused)
		}

		row := strings.Split(got, "\n")[line]
		if n := strings.Count(row, ansiReset); n != 1 {
			t.Errorf("focused=%v: selected row has %d resets, want 1 (the bar breaks at each extra one):\n%q",
				focused, n, row)
		}
		if !strings.HasSuffix(row, ansiReset) {
			t.Errorf("focused=%v: selected row does not close its style at the end:\n%q", focused, row)
		}
	}
}

// TestSelectedAndUnselectedRowsCarrySameText pins that the highlight changes only how a row is painted, never what it
// says.
//
// The selected row is built by a different function from the unselected one (styling a whole line and styling its parts
// cannot be the same code), so this is the guard against those two drifting.
func TestSelectedAndUnselectedRowsCarrySameText(t *testing.T) {
	diags := []grammar.Diagnostic{
		grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
		grammar.Warnf(pos("/work/models/a.go", 20, 5), grammar.CodeAmbiguousEmbed, "ambiguous"),
	}

	// Render twice, moving the selection; each row is then available in both roles. Content line 0 is the tally, so
	// diagnostic i sits on line i+1.
	selFirst := strings.Split(mustRender(t, diags, 0), "\n")
	selSecond := strings.Split(mustRender(t, diags, 1), "\n")

	for i, both := range []struct{ selected, plain string }{
		{selected: selFirst[1], plain: selSecond[1]},
		{selected: selSecond[2], plain: selFirst[2]},
	} {
		if got, want := stripANSI(both.selected), stripANSI(both.plain); got != want {
			t.Errorf("row %d reads differently selected vs not:\n selected: %q\n plain   : %q", i, got, want)
		}
	}
}

func mustRender(t *testing.T, diags []grammar.Diagnostic, selected int) string {
	t.Helper()
	got, line := Render("/work", nil, diags, selected, true)
	if line < 0 {
		t.Fatalf("selected=%d: no selected line reported", selected)
	}

	return got
}

// TestUnselectedRowIsColoredBySeverity pins that severity reaches the message, not just the label.
//
// The label alone is a few characters; colouring only it does not let the pane be scanned for red, which is the point
// of colouring at all.
func TestUnselectedRowIsColoredBySeverity(t *testing.T) {
	diags := []grammar.Diagnostic{
		// Selection is on row 0, so row 1 is the one rendered in its severity colour.
		grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
		grammar.Warnf(pos("/work/models/a.go", 20, 5), grammar.CodeAmbiguousEmbed, "ambiguous message"),
	}
	got, _ := Render("/work", nil, diags, 0, true)
	row := strings.Split(got, "\n")[2]

	// The message must sit inside a styled run rather than trailing the label's reset as plain text.
	before, _, found := strings.Cut(row, "ambiguous message")
	if !found {
		t.Fatalf("message missing from row: %q", row)
	}
	if !strings.HasSuffix(before, "m") || !strings.Contains(before, "\x1b[") {
		t.Errorf("message is not opened by a style sequence — severity colour stops at the label:\n%q", row)
	}
}

const ansiReset = "\x1b[0m"

// stripANSI removes CSI escape sequences, leaving the text a user actually reads.
//
// Hand-rolled rather than pulled from a dependency: it is a handful of lines, and promoting an indirect module to a
// direct one for a test assertion is a poor trade.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] == ';' || (s[i] >= '0' && s[i] <= '9')) {
				i++
			}
			if i < len(s) {
				i++ // the final byte (e.g. 'm')
			}

			continue
		}
		b.WriteByte(s[i])
		i++
	}

	return b.String()
}

func pos(file string, line, col int) token.Position {
	return token.Position{Filename: file, Line: line, Column: col}
}
