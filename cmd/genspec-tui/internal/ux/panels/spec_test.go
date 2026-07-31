// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func newLoadedSpec() Spec {
	sp := NewSpec()
	sp.SetSize(40, 12)
	sp.SetContent("{\n  \"a\": 1,\n  \"b\": 2,\n  \"c\": 3\n}")
	return sp
}

func TestSpec_CursorStartsAtTheTop(t *testing.T) {
	sp := newLoadedSpec()
	assert.Equal(t, 0, sp.CursorLine(), "a fresh spec puts the cursor on the first line")
}

func TestSpec_JumpToMovesTheCursor(t *testing.T) {
	sp := newLoadedSpec()

	sp.JumpTo(2)

	assert.Equal(t, 2, sp.CursorLine())
	assert.Contains(t, sp.Content(), "\"b\": 2",
		"raw content is unchanged — the cursor is view-only")
}

func TestSpec_CursorClamps(t *testing.T) {
	sp := newLoadedSpec() // 5 lines

	sp.SetCursor(99)
	assert.Equal(t, sp.LastLine(), sp.CursorLine(), "clamped at the last line")

	sp.SetCursor(-5)
	assert.Equal(t, 0, sp.CursorLine(), "clamped at the first")

	sp.MoveCursor(+2)
	assert.Equal(t, 2, sp.CursorLine())
	sp.MoveCursor(-99)
	assert.Equal(t, 0, sp.CursorLine())
}

// Searching parks the cursor ON the match, so that follow, find-references and
// go-to-definition all act on what was just searched for.
func TestSpec_SearchMovesTheCursorToTheMatch(t *testing.T) {
	sp := newLoadedSpec()

	require.Equal(t, 1, sp.Search("b"))

	assert.Equal(t, 2, sp.CursorLine(), `the line holding "b": 2`)
}

// New content CLAMPS the cursor rather than resetting it: a rescan re-renders
// nearly the same document, and dropping to line 0 on every save would make the
// live-reload loop unusable. Restoring the same NODE is the caller's job.
func TestSpec_SetContentClampsRatherThanResets(t *testing.T) {
	sp := newLoadedSpec()
	sp.JumpTo(3)

	sp.SetContent("{\n  \"x\": 9,\n  \"y\": 8\n}")
	assert.Equal(t, 3, sp.CursorLine(), "still in range, so kept")

	sp.SetContent("{\n}")
	assert.Equal(t, 1, sp.CursorLine(), "clamped into the shorter document")
}

func TestSpec_RenderPreservesContent(t *testing.T) {
	sp := newLoadedSpec()
	sp.JumpTo(2) // forces the styled render path
	// The viewport render must still show every source line.
	view := sp.vp.View()
	for _, want := range []string{"\"a\": 1", "\"b\": 2", "\"c\": 3"} {
		assert.Contains(t, view, want)
	}
}
