// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func newLoadedSpec() Spec {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}")
	return sp
}

func TestSpec_HighlightLine(t *testing.T) {
	sp := newLoadedSpec()
	assert.Equal(t, -1, sp.xrefLine, "no highlight on a fresh spec")

	sp.HighlightLine(2)
	assert.Equal(t, 2, sp.xrefLine, "cross-ref highlight set")
	assert.Contains(t, sp.Content(), "\"b\": 2", "raw content is unchanged (highlight is view-only)")

	sp.ClearHighlight()
	assert.Equal(t, -1, sp.xrefLine, "highlight cleared")
}

func TestSpec_HighlightInvalidatedBySearchAndContent(t *testing.T) {
	sp := newLoadedSpec()

	sp.HighlightLine(1)
	sp.Search("b")
	assert.Equal(t, -1, sp.xrefLine, "a search supersedes the cross-ref highlight")

	sp.HighlightLine(3)
	sp.SetContent("{\n  \"x\": 9\n}")
	assert.Equal(t, -1, sp.xrefLine, "new content invalidates the highlight")
}

func TestSpec_RenderPreservesContent(t *testing.T) {
	sp := newLoadedSpec()
	sp.HighlightLine(2) // forces the styled render path
	// The viewport render must still show every source line.
	view := sp.vp.View()
	for _, want := range []string{"\"a\": 1", "\"b\": 2", "\"c\": 3"} {
		assert.Contains(t, view, want)
	}
}
