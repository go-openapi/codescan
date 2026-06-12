// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package theme holds the lipgloss styles shared by the model and its panels:
// a rounded-border panel box (bright when focused, dim otherwise), a panel
// title, and the status line. Kept tiny and dependency-free so both ux and
// panels can import it without a cycle.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	colorActive   = lipgloss.Color("170")
	colorInactive = lipgloss.Color("240")
	colorTitle    = lipgloss.Color("213")
	colorDim      = lipgloss.Color("245")
	colorError    = lipgloss.Color("203")
	colorWarn     = lipgloss.Color("214")
	colorHint     = lipgloss.Color("110")
)

// Panel returns a rounded-border box style whose OUTER dimensions are w×h
// (the border consumes one cell on each side). The border is bright when
// focused and dim otherwise.
func Panel(w, h int, focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(max(w-2, 0)).
		Height(max(h-2, 0))
	if focused {
		return s.BorderForeground(colorActive)
	}
	return s.BorderForeground(colorInactive)
}

// Title styles a panel's header line.
func Title(focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().Bold(true)
	if focused {
		return s.Foreground(colorTitle)
	}
	return s.Foreground(colorDim)
}

// Status styles the bottom status/help line.
func Status() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorDim)
}

// Accent styles the app name / emphasised header text.
func Accent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorActive).Bold(true)
}

// Match styles a search hit in the spec pane.
func Match() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("16")).
		Background(lipgloss.Color("226"))
}

// Modal styles a centered popup box (e.g. the scanner-options dialog).
func Modal() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorActive).
		Padding(1, 3)
}

// SevError, SevWarn, and SevHint style a diagnostic's severity label in the
// diagnostics pane (red / amber / blue), matching grammar.Severity order.
func SevError() lipgloss.Style { return lipgloss.NewStyle().Foreground(colorError).Bold(true) }

// SevWarn styles a warning-severity diagnostic label.
func SevWarn() lipgloss.Style { return lipgloss.NewStyle().Foreground(colorWarn) }

// SevHint styles a hint-severity diagnostic label.
func SevHint() lipgloss.Style { return lipgloss.NewStyle().Foreground(colorHint) }

// Dir styles a directory row in the source tree.
func Dir() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorTitle)
}

// Selected styles the cursor row in a navigable panel.
func Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("231")).
		Background(colorActive)
}
