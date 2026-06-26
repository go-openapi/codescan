// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// followFixture builds a Model wired with a known spec/source index pair, ready
// to drive follow mode without a real scan.
func followFixture(t *testing.T) *Model {
	t.Helper()
	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.spec.SetSize(60, 20)
	m.fileView.SetSize(60, 20)
	m.spec.SetContent("line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8")
	m.specIndex = newSpecIndex(
		map[int]string{7: "/definitions/User/properties/email"},
		map[string]int{"/definitions/User/properties/email": 7},
	)
	return m
}

func TestFollow_SourceDriven(t *testing.T) {
	m := followFixture(t)
	m.srcIndex = BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User/properties/email", Pos: token.Position{Filename: "user.go", Line: 5}},
	})
	m.currentFile = "user.go"
	m.fileView.SetFile("user.go", "a\nb\nc\nd\ne\nf")
	m.fileView.GotoLine(4) // 0-based row 4 == source line 5
	m.focused = paneTree
	m.leftMode = modeView

	m.toggleFollow(followSource)
	assert.Equal(t, followSource, m.follow, "f enters source-driven follow")
	assert.Equal(t, "/definitions/User/properties/email", m.followTarget)

	// A line with no anchor at or above reports honestly.
	m.fileView.GotoLine(0) // source line 1, before the first anchor (line 5)
	m.syncFollowIfActive()
	assert.Equal(t, "(no spec node)", m.followTarget)

	// Moving focus off the driver exits follow.
	m.focused = paneSpec
	m.syncFollowIfActive()
	assert.Equal(t, followOff, m.follow, "leaving the driver pane exits follow")
}

func TestFollow_SpecDriven(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "user.go")
	require.NoError(t, os.WriteFile(src, []byte("package p\n\ntype User struct{}\n"), 0o600))

	m := &Model{spec: panels.NewSpec(), fileView: panels.NewFileView()}
	m.cfg.WorkDir = dir
	m.spec.SetSize(60, 20)
	m.fileView.SetSize(60, 20)
	m.spec.SetContent("{\n  \"definitions\": {}\n}")
	// The node maps to the top line of the viewport (YOffset 0).
	m.specIndex = newSpecIndex(
		map[int]string{0: "/definitions/User"},
		map[string]int{"/definitions/User": 0},
	)
	m.srcIndex = BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: token.Position{Filename: src, Line: 3}},
	})
	m.focused = paneSpec

	m.toggleFollow(followSpec)
	assert.Equal(t, followSpec, m.follow)
	assert.Equal(t, src, m.currentFile, "the source follower loads the producing file")
	assert.Equal(t, paneSpec, m.focused, "the driver keeps focus")
	assert.Contains(t, m.followTarget, "/definitions/User")
	assert.Contains(t, m.followTarget, "user.go:3")

	// f again toggles off.
	m.toggleFollow(followSpec)
	assert.Equal(t, followOff, m.follow)
	assert.Empty(t, m.followTarget)
}

func TestFollow_ExitClearsState(t *testing.T) {
	m := followFixture(t)
	m.srcIndex = BuildSourceIndex(nil)
	m.follow = followSpec
	m.followTarget = "something"

	m.exitFollow()
	assert.Equal(t, followOff, m.follow)
	assert.Empty(t, m.followTarget)
	// Idempotent.
	m.exitFollow()
	assert.Equal(t, followOff, m.follow)
}
