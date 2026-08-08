// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/theme"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The common case, and it is common by construction rather than by luck.
//
// A diagnostic and a lexical run both address a token, so they agree on where it starts.
func TestDiagMarks_ExactHitRestylesTheRun(t *testing.T) {
	marked := MarkDiagnostics(lexicalSpans(), []DiagMark{
		{Line: 4, Col: 5, Kind: theme.SyntaxDiagError},
	})

	assert.Equal(t, []theme.Span{
		{Col: 2, Kind: theme.SyntaxComment},
		{Col: 5, Kind: theme.SyntaxDiagError},
		{Col: 7, Kind: theme.SyntaxComment},
	}, marked[4], "the keyword's own run carries the diagnostic")
}

// Not every diagnostic lands on a token boundary.
//
// codescan reports an invalid enum option at the space before the value.
//
// Opening a run there over-reaches rather than pointing somewhere false.
func TestDiagMarks_InsideARunOpensANewOne(t *testing.T) {
	marked := MarkDiagnostics(lexicalSpans(), []DiagMark{
		{Line: 4, Col: 6, Kind: theme.SyntaxDiagWarn},
	})

	require.Len(t, marked[4], 4)
	assert.Equal(t, theme.Span{Col: 6, Kind: theme.SyntaxDiagWarn}, marked[4][2])
	for i := 1; i < len(marked[4]); i++ {
		assert.Greater(t, marked[4][i].Col, marked[4][i-1].Col, "runs must stay ordered")
	}
}

// Lexical spans are rebuilt when the BUFFER changes; marks change on every rescan.
//
// Mutating the input would make a rescan's marks accumulate on top of the previous scan's.
func TestDiagMarks_LeavesTheInputAlone(t *testing.T) {
	spans := lexicalSpans()

	_ = MarkDiagnostics(spans, []DiagMark{{Line: 4, Col: 5, Kind: theme.SyntaxDiagError}})

	assert.Equal(t, theme.SyntaxKeyword, spans[4][1].Kind, "the lexical spans are untouched")
}

func TestDiagMarks_Edges(t *testing.T) {
	assert.Nil(t, MarkDiagnostics(nil, []DiagMark{{Line: 0, Col: 1, Kind: theme.SyntaxDiagError}}),
		"a file we do not tokenize gets no invented runs")

	spans := lexicalSpans()
	assert.Equal(t, spans, MarkDiagnostics(spans, nil), "no marks, nothing to do")

	marked := MarkDiagnostics(lexicalSpans(), []DiagMark{
		{Line: -1, Col: 5, Kind: theme.SyntaxDiagError},
		{Line: 4, Col: 0, Kind: theme.SyntaxDiagError},
	})
	assert.Equal(t, lexicalSpans(), marked, "positionless marks are dropped, not clamped onto line 0")
}

// A diagnostic on a line with no lexical runs still marks it - a blank or unclassified line can carry one.
func TestDiagMarks_LineWithNoRuns(t *testing.T) {
	marked := MarkDiagnostics(lexicalSpans(), []DiagMark{
		{Line: 9, Col: 3, Kind: theme.SyntaxDiagHint},
	})

	assert.Equal(t, []theme.Span{{Col: 3, Kind: theme.SyntaxDiagHint}}, marked[9])
	assert.Len(t, marked[4], 3, "other lines are unaffected")
}

// lexicalSpans is what an annotated in: formData line tokenizes to.
//
// The comment markers, the keyword, then the value back in prose.
func lexicalSpans() map[int][]theme.Span {
	return map[int][]theme.Span{
		4: {
			{Col: 2, Kind: theme.SyntaxComment},
			{Col: 5, Kind: theme.SyntaxKeyword},
			{Col: 7, Kind: theme.SyntaxComment},
		},
	}
}
