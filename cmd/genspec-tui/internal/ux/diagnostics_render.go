// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// renderDiagnostics composes the diagnostics-pane body for one scan outcome and
// reports the 0-based content line of the selected diagnostic (-1 when none).
//
// A hard error from codescan.Run is shown first — it aborts the whole spec, so
// it dwarfs everything else. Soft diagnostics follow a one-line severity tally,
// one per row in source order, colored by severity; the selected row gets the
// whole-line highlight (for diagnostic→source navigation). Paths are trimmed to
// the work dir to keep rows short. An empty, error-free scan shows the rest
// state. The selected line is counted as the body is built, so it stays correct
// even when a diagnostic message spans multiple lines.
func renderDiagnostics(workdir string, scanErr error, diags []grammar.Diagnostic, selected int) (string, int) {
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
		row := formatDiagnostic(workdir, d)
		if i == selected {
			selectedLine = strings.Count(b.String(), "\n") // 0-based line of this row
			row = theme.Selected().Render(row)
		}
		b.WriteString(row)
	}
	return b.String(), selectedLine
}

// diagnosticTally summarizes a diagnostic slice as "N diagnostics (E errors, W
// warnings, H hints)", omitting any zero buckets.
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

// formatDiagnostic renders one diagnostic as "path:line:col severity: message
// [code]", with the severity label colored and the path made relative to
// workdir when it sits inside the scanned tree.
func formatDiagnostic(workdir string, d grammar.Diagnostic) string {
	loc := d.Pos.String() // absolute "file:line:col" (or "-" when unknown)
	if rel, err := filepath.Rel(workdir, d.Pos.Filename); err == nil && !strings.HasPrefix(rel, "..") {
		loc = fmt.Sprintf("%s:%d:%d", rel, d.Pos.Line, d.Pos.Column)
	}
	sev := severityStyle(d.Severity).Render(d.Severity.String())
	return fmt.Sprintf("%s %s: %s [%s]", loc, sev, d.Message, d.Code)
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
