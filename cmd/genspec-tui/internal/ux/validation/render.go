// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// Render composes the validation tab's body.
//
// It also reports the 0-based content line of the selected finding, or -1 when there is none.
//
// Laid out like the scan tab and coloured by the same rule - label and message in the severity's hue, the selected row
// taking the whole line over RAW text - so the two tabs read as two views of one pane rather than two widgets.
func Render(findings []Finding, ran bool, runErr error, selected int, focused bool) (string, int) {
	if runErr != nil {
		return theme.SevError().Render("validation failed: ") + runErr.Error(), -1
	}
	if !ran {
		return "(press v to validate the generated spec)", -1
	}
	if len(findings) == 0 {
		return "(the generated spec is valid)", -1
	}

	var b strings.Builder
	b.WriteString(theme.Status().Render(tally(findings)))

	selectedLine := -1
	for i, f := range findings {
		b.WriteString("\n")
		if i == selected {
			selectedLine = strings.Count(b.String(), "\n")
			if focused {
				b.WriteString(theme.Selected().Render(plainRow(f)))
			} else {
				b.WriteString(theme.Follower().Render(plainRow(f)))
			}

			continue
		}
		b.WriteString(styledRow(f))
	}

	return b.String(), selectedLine
}

// tally summarises the findings by severity.
func tally(findings []Finding) string {
	errs, warns := Tally(findings)

	var parts []string
	for _, p := range []struct {
		n   int
		one string
	}{{errs, "error"}, {warns, "warning"}} {
		if p.n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s%s", p.n, p.one, plural(p.n)))
		}
	}

	noun := "finding" + plural(len(findings))
	if len(parts) == 0 {
		return fmt.Sprintf("%d %s", len(findings), noun)
	}

	return fmt.Sprintf("%d %s (%s)", len(findings), noun, strings.Join(parts, ", "))
}

// styledRow renders one finding coloured by severity.
func styledRow(f Finding) string {
	style := severityStyle(f.Severity)

	return fmt.Sprintf("%s %s: %s",
		theme.Dim().Render(location(f)),
		style.Render(f.Severity.String()),
		style.Bold(false).Render(f.Message),
	)
}

// plainRow is styledRow's unstyled twin, for the selected line's whole-row highlight.
//
// Built from the same pieces in the same order so the two can never show different text depending on where the cursor
// is - the same pairing the scan tab uses, and for the same reason.
func plainRow(f Finding) string {
	return fmt.Sprintf("%s %s: %s", location(f), f.Severity, f.Message)
}

// location is the finding's reported path, or a marker when it named none.
func location(f Finding) string {
	if f.Path == "" {
		return "(spec)"
	}

	return f.Path
}

// severityStyle maps a severity to its pane style.
func severityStyle(s grammar.Severity) lipgloss.Style {
	if s == grammar.SeverityError {
		return theme.SevError()
	}

	return theme.SevWarn()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}

	return "s"
}
