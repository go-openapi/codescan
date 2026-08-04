// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package panels

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

func newLoadedFileView() FileView {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "line1\nline2\nline3\nline4")
	return fv
}

func TestFileView_ReadOnlyNav(t *testing.T) {
	fv := newLoadedFileView()

	assert.False(t, fv.Editing(), "opens read-only")
	assert.Equal(t, 0, fv.CurrentLine(), "starts at the top")

	fv.NavDown()
	fv.NavDown()
	assert.Equal(t, 2, fv.CurrentLine())

	fv.NavUp()
	assert.Equal(t, 1, fv.CurrentLine())

	// Clamped at both ends.
	fv.NavUp()
	fv.NavUp()
	assert.Equal(t, 0, fv.CurrentLine(), "clamped at the first line")

	for range 10 {
		fv.NavDown()
	}
	assert.Equal(t, 3, fv.CurrentLine(), "clamped at the last line (4 lines, 0-based)")
}

// textarea NORMALISES the text it is given — tabs become spaces — so comparing the buffer against the file marked
// every tab-indented Go file as edited the moment it was opened: a ● in the title, and a STALE badge on follow, with
// nothing touched.
//
// The buffer is the baseline, not the bytes on disk.
func TestFileView_NotDirtyOnOpen(t *testing.T) {
	fv := NewFileView()
	fv.SetSize(40, 10)
	fv.SetFile("x.go", "package p\n\ntype T struct {\n\tID int64\n}\n")

	assert.False(t, fv.Dirty(), "a freshly opened file has no unsaved edits")
	assert.NotContains(t, fv.View(false, false), "●")
}

func TestFileView_GotoLine(t *testing.T) {
	fv := newLoadedFileView()
	fv.GotoLine(2)
	assert.Equal(t, 2, fv.CurrentLine())

	// Out-of-range is clamped, not an error.
	fv.GotoLine(99)
	assert.Equal(t, 3, fv.CurrentLine())
	fv.GotoLine(-5)
	assert.Equal(t, 0, fv.CurrentLine())
}

func TestFileView_EditToggle(t *testing.T) {
	fv := newLoadedFileView()
	fv.GotoLine(2)

	_ = fv.StartEdit()
	assert.True(t, fv.Editing(), "i enters edit mode")
	assert.Equal(t, 2, fv.CurrentLine(), "editor cursor lands on the nav line")

	fv.StopEdit()
	assert.False(t, fv.Editing(), "esc leaves edit mode")
	assert.Equal(t, 2, fv.CurrentLine(), "nav line parks where the cursor was")
}

func TestFileView_ViewRendersGutterAndHighlight(t *testing.T) {
	fv := newLoadedFileView()
	fv.GotoLine(1)

	out := fv.View(true, true)
	assert.Contains(t, out, "line2", "shows file content")
	assert.Contains(t, out, "view", "title marks read-only mode")

	_ = fv.StartEdit()
	assert.Contains(t, fv.View(true, true), "edit", "title marks edit mode")
}
