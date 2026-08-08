// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestAnnotationDoc_EveryKindIsDocumented is the completeness guard.
//
// An annotation added without an entry here would surface in the editor's help as nothing at all — the one place the
// omission is invisible, since a missing entry looks exactly like pressing the key on a line with no annotation.
//
// The range is walked from the first real kind to the last, so a kind appended to the enum is covered without this
// test being touched.
func TestAnnotationDoc_EveryKindIsDocumented(t *testing.T) {
	for a := AnnModel; a <= AnnDescription; a++ {
		t.Run(a.String(), func(t *testing.T) {
			doc, ok := a.Doc()
			require.True(t, ok, "swagger:%s has no reference entry", a)

			assert.NotEmpty(t, doc.Usage)
			assert.NotEmpty(t, doc.Summary)
			assert.NotEmpty(t, doc.Keywords, "say `None.` rather than leaving it blank: silence reads as unfinished")
		})
	}
}

// TestAnnotationDoc_UsageNamesItsOwnAnnotation guards the copy/paste failure this table invites: twenty near-identical
// entries, where a wrong Usage line is still perfectly plausible prose.
func TestAnnotationDoc_UsageNamesItsOwnAnnotation(t *testing.T) {
	for a := AnnModel; a <= AnnDescription; a++ {
		doc, ok := a.Doc()
		require.True(t, ok)

		assert.True(t, strings.HasPrefix(doc.Usage, AnnotationPrefix+a.String()),
			"swagger:%s has the usage line of another annotation: %q", a, doc.Usage)
	}
}

// TestAnnotationDoc_SummaryIsOneSentence keeps the entries to the size the popup was built for.
func TestAnnotationDoc_SummaryIsOneSentence(t *testing.T) {
	for a := AnnModel; a <= AnnDescription; a++ {
		doc, _ := a.Doc()

		assert.LessOrEqual(t, len(doc.Summary), 100, "swagger:%s summary is too long for one line: %q", a, doc.Summary)
		assert.True(t, strings.HasSuffix(doc.Summary, "."), "swagger:%s summary is not a sentence: %q", a, doc.Summary)
	}
}

// TestAnnotationDoc_UnknownHasNone pins the miss: nothing to say about a directive the grammar does not recognise.
func TestAnnotationDoc_UnknownHasNone(t *testing.T) {
	doc, ok := AnnUnknown.Doc()

	assert.False(t, ok)
	assert.Empty(t, doc.Usage)
}

// TestAnnotationDoc_ResolvesFromSourceSpelling ties the table to the way the scanner names annotations, so the lookup
// the editor performs — text on a line → kind → doc — cannot drift from the parser's own vocabulary.
func TestAnnotationDoc_ResolvesFromSourceSpelling(t *testing.T) {
	for _, name := range []string{"model", "route", "parameters", "omit", "patternProperties"} {
		kind := AnnotationKindFromName(name)
		require.NotEqual(t, AnnUnknown, kind, "%q must resolve", name)

		doc, ok := kind.Doc()
		require.True(t, ok)
		assert.Contains(t, doc.Usage, name)
	}
}
