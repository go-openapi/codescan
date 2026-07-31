// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package theme holds the lipgloss styles shared by the model and its panels:
// a rounded-border panel box (bright when focused, dim otherwise), a panel
// title, and the status line. Kept tiny and dependency-free so both ux and
// panels can import it without a cycle.
package theme

import "github.com/charmbracelet/lipgloss"

// The palette. Constants rather than variables: a theme nothing can reassign at
// runtime is one less thing a rendering bug can be.
const (
	colorActive   = lipgloss.Color("170")
	colorInactive = lipgloss.Color("240")
	colorTitle    = lipgloss.Color("213")
	colorDim      = lipgloss.Color("245")
	colorError    = lipgloss.Color("203")
	colorWarn     = lipgloss.Color("214")
	colorHint     = lipgloss.Color("110")
	colorFollower = lipgloss.Color("53") // muted shade of colorActive, for mirrored lines

	colorSyntaxKey     = lipgloss.Color("81")
	colorSyntaxString  = lipgloss.Color("114")
	colorSyntaxNumber  = lipgloss.Color("215")
	colorSyntaxKeyword = lipgloss.Color("176")
	colorSyntaxPunct   = lipgloss.Color("242")
	colorSyntaxComment = lipgloss.Color("244")
)

// SyntaxKind classifies a lexical run for highlighting. It is deliberately
// source-language-neutral: the JSON/YAML lexers and go/scanner both map onto it,
// so the renderer and the palette are shared rather than duplicated per pane.
type SyntaxKind uint8

// The syntax classes. Plain is the zero value, so an unmapped token simply
// renders unstyled rather than wrong.
//
// The Diag* classes are the exception to "lexical": they are not what a token
// IS but what the scanner said ABOUT it, overlaid on the run the diagnostic
// points at. They ride the same span mechanism because a diagnostic and a token
// address the same thing — a (line, column) run — and giving them a second
// mechanism would mean two ways to paint one line.
const (
	SyntaxPlain SyntaxKind = iota
	SyntaxKey
	SyntaxString
	SyntaxNumber
	SyntaxKeyword
	SyntaxPunct
	SyntaxComment
	SyntaxDiagError
	SyntaxDiagWarn
	SyntaxDiagHint
)

// Span is one lexical run on a rendered line, identified by where it STARTS.
// A run extends to the next span's column (or the end of the line), which is
// what lets the renderer slice raw text at known boundaries instead of
// truncating already-coloured output — the operation that corrupts escapes.
type Span struct {
	Col  int // 1-based column of the run's first character
	Kind SyntaxKind
}

// Syntax returns the style for a syntax class. The diagnostic classes underline
// as well as recolour: underline is the terminal's squiggle, and it survives a
// palette where the severity colour is close to a syntax one.
func Syntax(k SyntaxKind) lipgloss.Style {
	switch k {
	case SyntaxKey:
		return lipgloss.NewStyle().Foreground(colorSyntaxKey)
	case SyntaxString:
		return lipgloss.NewStyle().Foreground(colorSyntaxString)
	case SyntaxNumber:
		return lipgloss.NewStyle().Foreground(colorSyntaxNumber)
	case SyntaxKeyword:
		return lipgloss.NewStyle().Foreground(colorSyntaxKeyword)
	case SyntaxPunct:
		return lipgloss.NewStyle().Foreground(colorSyntaxPunct)
	case SyntaxComment:
		return lipgloss.NewStyle().Foreground(colorSyntaxComment).Italic(true)
	case SyntaxDiagError:
		return lipgloss.NewStyle().Foreground(colorError).Underline(true)
	case SyntaxDiagWarn:
		return lipgloss.NewStyle().Foreground(colorWarn).Underline(true)
	case SyntaxDiagHint:
		return lipgloss.NewStyle().Foreground(colorHint).Underline(true)
	case SyntaxPlain:
		return lipgloss.NewStyle()
	default:
		return lipgloss.NewStyle()
	}
}

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

// Selected styles the cursor row in a navigable panel — the DRIVER line, i.e.
// the one the user is actually moving. Strong reverse-video bar (design §6.5).
func Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("231")).
		Background(colorActive)
}

// Follower styles a cross-ref line in the pane that is MIRRORING the driver.
// A muted tint of Selected, so at a glance it is obvious which pane leads and
// which one is being dragged along (design §6.5).
func Follower() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(colorFollower)
}

// Chip styles a small standing badge in the header — currently the "h: help"
// hint, which has to survive a crowded header line without being mistaken for
// one more status field.
func Chip() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("16")).
		Background(colorHint).
		Bold(true)
}

// Gutter styles the link markers in the spec pane's and source viewer's gutter.
// Dim on purpose: they are a hint about what is navigable, not content.
func Gutter() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorHint)
}

// Stale styles the follow-mode badge shown while the source buffer has unsaved
// edits, i.e. while cross-ref positions are older than what is on screen.
func Stale() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("16")).
		Background(colorWarn).
		Bold(true)
}
