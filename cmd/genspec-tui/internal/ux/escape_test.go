// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/testutils"
)

// The payload every assertion below is written against.
//
// The marker is chosen so that "the raw form is absent" says something about the scanned tree rather than about the
// TUI's own colours, which this package's TestMain deliberately turns on.
const (
	rawSGR     = "\x1b[31mPWNED"
	encodedSGR = "␛[31mPWNED"
)

// assertNoLeak checks that a rendered view carries the picture and not the command.
func assertNoLeak(t *testing.T, out string) {
	t.Helper()

	assert.NotContains(t, out, rawSGR, "a control sequence from the scanned tree reached the terminal")
	assert.Contains(t, out, encodedSGR, "the control sequence was dropped instead of being shown")
}

// craftedFile writes a Go file whose NAME carries a control sequence, and returns its path.
//
// Skips where the filesystem will not take the name, which is what a windows runner does.
func craftedFile(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "evil"+rawSGR+".go")
	if err := os.WriteFile(path, []byte("package p\n"), 0o600); err != nil {
		t.Skipf("this filesystem will not hold a name with a control character: %v", err)
	}

	return path
}

// TestOpenFile_CraftedNameDoesNotReachTheTerminal walks the path the report describes, end to end.
//
// A repository names a file, the reader opens it from the tree, and the name is then drawn in three places at once:
// the panel title, the status notice, and (once an edit is pending) the confirmation prompt.
func TestOpenFile_CraftedNameDoesNotReachTheTerminal(t *testing.T) {
	dir := t.TempDir()
	path := craftedFile(t, dir)

	m := testModelIn(t, dir, openFile(path))

	t.Run("panel title", func(t *testing.T) {
		assertNoLeak(t, m.leftView(true))
	})

	t.Run("status notice", func(t *testing.T) {
		require.NotNil(t, m.reloadFile())
		assertNoLeak(t, m.statusLine())
	})

	t.Run("confirmation prompt", func(t *testing.T) {
		m.fileView.StartEdit()
		m.fileView.Update(testutils.KeyRune('x'))
		require.True(t, m.fileView.Dirty(), "the buffer must be dirty for the prompt to be asked")

		require.Nil(t, m.requestReload())
		assertNoLeak(t, m.confirm.View())
	})
}

// TestHeaderLine_CraftedWorkDirDoesNotReachTheTerminal covers the one elastic field of the top chrome.
func TestHeaderLine_CraftedWorkDirDoesNotReachTheTerminal(t *testing.T) {
	m := testModel(t, sized(200, 24))
	m.cfg.WorkDir = "/src/" + rawSGR

	assertNoLeak(t, m.headerLine())
}

// TestSpecRender_EscapesControlCharacters pins the assumption the spec pane rests on.
//
// That pane draws thousands of lines and re-draws them on every cursor move, so it does not sanitize: it is fed only
// the serializers' output, and both of them encode a control character rather than pass it through. If that ever
// stopped being true the pane would need the same treatment as the rest, so it is asserted rather than assumed.
func TestSpecRender_EscapesControlCharacters(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module evil\n\ngo 1.25.0\n"), 0o600))

	src := "// Package evil.\n" +
		"//\n" +
		"// Version: 1.0.0\n" +
		"//\n" +
		"// swagger:meta\n" +
		"package evil\n" +
		"\n" +
		"// Pet is a pet " + rawSGR + ".\n" +
		"//\n" +
		"// swagger:model Pet\n" +
		"type Pet struct {\n" +
		"\t// Name of the pet " + rawSGR + "\n" +
		"\tName string `json:\"name\"`\n" +
		"}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api.go"), []byte(src), 0o600))

	res := scan.Do(testutils.ApplyLoader(&codescan.Options{
		WorkDir:    dir,
		Packages:   []string{"./..."},
		GOWORK:     "off",
		ScanModels: true,
	}), nil)
	require.NoError(t, res.Err)
	require.Contains(t, res.JSON, "PWNED", "the crafted text never reached the document, so nothing is being tested")

	yamlBody, err := scan.RenderYAML(res.JSON)
	require.NoError(t, err)

	assert.NotContains(t, res.JSON, "\x1b", "the JSON render carries a raw escape")
	assert.NotContains(t, yamlBody, "\x1b", "the YAML render carries a raw escape")

	m := testModelIn(t, dir, withRenders(res.JSON, yamlBody))
	assert.NotContains(t, m.spec.View(true), rawSGR)
}

// TestNotify_EncodesControlSequences covers the funnel every transient status message goes through.
func TestNotify_EncodesControlSequences(t *testing.T) {
	m := testModel(t)

	require.NotNil(t, m.notify("no matches: %s", rawSGR))

	assert.NotContains(t, m.notice, rawSGR)
	assert.True(t, strings.Contains(m.notice, encodedSGR), "notice %q", m.notice)
	assertNoLeak(t, m.statusLine())
}
