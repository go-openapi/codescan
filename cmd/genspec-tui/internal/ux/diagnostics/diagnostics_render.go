// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// Render composes the diagnostics-pane body for one scan outcome and reports the 0-based content line of the
// selected diagnostic (-1 when none).
//
// A hard error from codescan.Run is shown first — it aborts the whole spec, so it dwarfs everything else.
//
// Soft diagnostics follow a one-line severity tally, one per row in source order, with the severity label and the
// message carrying the severity's colour so the pane can be read for red at a glance; the selected row instead gets the
// whole-line highlight (for diagnostic→source navigation).
// Paths are trimmed to the work dir to keep rows short.
//
// An empty, error-free scan shows the rest state.
// The selected line is counted as the body is built, so it stays correct even when a diagnostic message spans multiple
// lines.
func Render(workdir string, scanErr error, diags []grammar.Diagnostic, selected int, focused bool) (string, int) {
	var b strings.Builder
	selectedLine := -1

	if scanErr != nil {
		b.WriteString(theme.SevError().Render("scan failed: ") + scanErr.Error())
		if len(diags) == 0 {
			return b.String(), -1
		}
		b.WriteString("\n\n")
	}

	if len(diags) == 0 {
		return "(no diagnostics)", -1
	}

	b.WriteString(theme.Status().Render(diagnosticTally(diags)))
	for i, d := range diags {
		b.WriteString("\n")

		var row string
		if i == selected {
			selectedLine = strings.Count(b.String(), "\n") // 0-based line of this row
			// Precedence is the same as the spec pane and the source viewer: the cursor wins the whole line, so the
			// highlight is laid over the RAW text rather than over the severity-coloured one.
			// Wrapping already-styled text would not merely look busy — the inner style's reset terminates the outer
			// background mid-row, and lipgloss does not re-open it, so the bar would visibly stop at the severity label.
			//
			// The strong bar means "you are driving this", the muted tint means "this is where you were": two strong
			// bars on screen at once make it ambiguous which pane a keypress will reach.
			if focused {
				row = theme.Selected().Render(plainDiagnostic(workdir, d))
			} else {
				row = theme.Follower().Render(plainDiagnostic(workdir, d))
			}
		} else {
			row = formatDiagnostic(workdir, d)
		}

		b.WriteString(row)
	}

	return b.String(), selectedLine
}

// diagnosticTally summarizes a diagnostic slice as "N diagnostics (E errors, W warnings, H hints)", omitting any zero
// buckets.
func diagnosticTally(diags []grammar.Diagnostic) string {
	var e, w, h int
	for _, d := range diags {
		switch d.Severity {
		case grammar.SeverityError:
			e++
		case grammar.SeverityWarning:
			w++
		default:
			h++
		}
	}

	var parts []string
	for _, p := range []struct {
		n   int
		one string
	}{{e, "error"}, {w, "warning"}, {h, "hint"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s%s", p.n, p.one, plural(p.n)))
		}
	}

	noun := "diagnostic" + plural(len(diags))
	if len(parts) == 0 {
		return fmt.Sprintf("%d %s", len(diags), noun)
	}
	return fmt.Sprintf("%d %s (%s)", len(diags), noun, strings.Join(parts, ", "))
}

// formatDiagnostic renders one diagnostic as "path:line:col severity: message [code]", coloured by severity.
//
// The label and the message both take the severity's colour — the label alone is too small a target to scan a pane
// for, and the message is the part being read. Only the label is bold, so a wall of errors does not become a wall of
// bold. The trailing [code] is dimmed: it is a lookup key, not prose.
func formatDiagnostic(workdir string, d grammar.Diagnostic) string {
	sev := severityStyle(d.Severity)
	return fmt.Sprintf("%s %s: %s %s",
		diagnosticLoc(workdir, d),
		sev.Render(d.Severity.String()),
		// Bold(false) rather than a separate palette entry: the message wants the severity's hue at normal weight, and
		// deriving it here keeps one definition of what "error red" is.
		sev.Bold(false).Render(d.Message),
		theme.Dim().Render("["+string(d.Code)+"]"),
	)
}

// plainDiagnostic renders the same row as formatDiagnostic with no styling at all, for the selected row to be
// highlighted as a whole.
//
// Kept beside its styled twin, and built from the same pieces in the same order, so the two can never drift into
// showing different text depending on where the cursor is.
func plainDiagnostic(workdir string, d grammar.Diagnostic) string {
	return fmt.Sprintf("%s %s: %s [%s]", diagnosticLoc(workdir, d), d.Severity, d.Message, d.Code)
}

// diagnosticLoc renders a diagnostic's position as "path:line:col", made relative to workdir when it sits inside the
// scanned tree (absolute, or "-" when unknown, otherwise).
func diagnosticLoc(workdir string, d grammar.Diagnostic) string {
	if rel, err := filepath.Rel(workdir, d.Pos.Filename); err == nil && !strings.HasPrefix(rel, "..") {
		return fmt.Sprintf("%s:%d:%d", rel, d.Pos.Line, d.Pos.Column)
	}

	return d.Pos.String()
}

// severityStyle maps a grammar.Severity to its diagnostics-pane style.
func severityStyle(s grammar.Severity) lipgloss.Style {
	switch s {
	case grammar.SeverityError:
		return theme.SevError()
	case grammar.SeverityWarning:
		return theme.SevWarn()
	default:
		return theme.SevHint()
	}
}

// plural returns "s" unless n is exactly 1.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
