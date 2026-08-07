// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"slices"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
)

// DiagMark is one diagnostic located in the DISPLAYED text.
//
// A 0-based line and a 1-based rune column, already translated out of the file coordinates the scanner reports in.
//
// Kind is the severity class to paint.
type DiagMark struct {
	Line, Col int
	Kind      theme.SyntaxKind
}

// MarkDiagnostics overlays diagnostic marks onto lexical spans.
//
// It returns a new map and leaves the input untouched.
// The lexical spans are rebuilt only when the buffer changes, while the marks change on every rescan.
//
// A diagnostic wins over the token it lands on.
// The scanner's opinion is why the pane is open, so it takes the run rather than tinting around it.
//
// Nil spans stay nil: a file with no lexical runs is one we do not tokenize (not Go).
// Inventing runs for it would colour text that nobody classified.
func MarkDiagnostics(spans map[int][]theme.Span, marks []DiagMark) map[int][]theme.Span {
	if spans == nil || len(marks) == 0 {
		return spans
	}

	out := make(map[int][]theme.Span, len(spans))
	for line, runs := range spans {
		out[line] = slices.Clone(runs)
	}

	for _, mark := range marks {
		if mark.Line < 0 || mark.Col < 1 {
			continue
		}
		out[mark.Line] = markRun(out[mark.Line], mark)
	}

	return out
}

// markRun restyles the run that BEGINS at the mark's column, or opens one there when the mark falls inside a run.
//
// The exact hit is the common case rather than a lucky one: a diagnostic and a lexical run both address a token, so
// they agree on where it starts - a keyword-level diagnostic lands on the keyword's run, a declaration-level one on
// the identifier's. Opening a run mid-token is the honest fallback: it paints from the reported column to the next run,
// which over-reaches rather than pointing somewhere false.
func markRun(runs []theme.Span, mark DiagMark) []theme.Span {
	at, found := slices.BinarySearchFunc(runs, mark.Col, func(s theme.Span, col int) int {
		return s.Col - col
	})
	if found {
		runs[at].Kind = mark.Kind

		return runs
	}

	return slices.Insert(runs, at, theme.Span{Col: mark.Col, Kind: mark.Kind})
}
