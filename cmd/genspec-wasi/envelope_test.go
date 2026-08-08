// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// source is the module every test here scans.
//
// It is written out rather than pointed at a fixture because the assertions are about POSITIONS,
// and a fixture that someone reformats moves them without breaking anything visible.
const source = `package models

import "time"

// Doc is a document.
//
// swagger:model doc
type Doc struct {
	// When it happened
	At time.Time ` + "`json:\"at\"`" + `

	// A duration, which is a named type and so lands behind a $ref
	For time.Duration ` + "`json:\"for\"`" + `
}
`

// lineOf reports the 1-based line holding needle, so an assertion names what it is looking for
// rather than a number that drifts when the fixture above is edited.
func lineOf(t *testing.T, needle string) int {
	t.Helper()

	for i, line := range strings.Split(source, "\n") {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	require.Fail(t, "no line contains "+needle)

	return 0
}

func moduleDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/env\n\ngo 1.25.0\n"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "models"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "models", "m.go"), []byte(source), 0o600))

	return dir
}

// scanJSON runs the command as a caller would and returns the parsed envelope.
func scanJSON(t *testing.T, dir string) envelope {
	t.Helper()

	var stdout, stderr bytes.Buffer
	require.NoError(t, run([]string{
		"-format=json", "-loader=own", "-workdir", dir, "./...",
	}, &stdout, &stderr))

	assert.Empty(t, stderr.String(), "json mode carries diagnostics in the document, so stderr stays clean")

	var got envelope
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))

	return got
}

// TestJSONFormatAnchorsNodesToTheirSource is the property the playground's cross-highlight rests on:
// a pointer into the emitted document, and the line that produced it.
func TestJSONFormatAnchorsNodesToTheirSource(t *testing.T) {
	t.Parallel()

	got := scanJSON(t, moduleDir(t))

	anchors := map[string]jsonAnchor{}
	for _, a := range got.Provenance {
		anchors[a.Pointer] = a
	}

	decl, ok := anchors["/definitions/doc"]
	require.True(t, ok, "no anchor for the definition; got %v", anchors)
	assert.Equal(t, "models/m.go", decl.File)
	assert.Equal(t, lineOf(t, "type Doc struct"), decl.Line)

	field, ok := anchors["/definitions/doc/properties/at"]
	require.True(t, ok, "no anchor for the field; got %v", anchors)
	assert.Equal(t, "models/m.go", field.File)
	assert.Equal(t, lineOf(t, "At time.Time"), field.Line)
}

// TestJSONFormatPositionsDiagnostics checks the other half: a diagnostic that a consumer can put a
// mark against, rather than a sentence with a path printed in it.
func TestJSONFormatPositionsDiagnostics(t *testing.T) {
	t.Parallel()

	got := scanJSON(t, moduleDir(t))
	require.NotEmpty(t, got.Diagnostics, "the $ref'd field's dropped description should be reported")

	d := got.Diagnostics[0]
	assert.Equal(t, "hint", d.Severity)
	assert.Equal(t, "validate.dropped-ref-sibling", d.Code)
	assert.Equal(t, "models/m.go", d.File)

	// The mark lands on the prose that was dropped, not on the field it was attached to. Worth
	// pinning: a consumer drawing a gutter mark from this puts it a line above the declaration, and
	// that is right rather than off by one.
	assert.Equal(t, lineOf(t, "A duration, which is a named type"), d.Line)
}

// TestJSONFormatLeavesOutOfTreePositionsAbsolute pins the deliberate exception. time.Duration is
// declared in GOROOT, so relativising it would produce a chain of ".." that names it no better -
// and a consumer tells the two apart by whether the path matches a file it holds.
func TestJSONFormatLeavesOutOfTreePositionsAbsolute(t *testing.T) {
	t.Parallel()

	got := scanJSON(t, moduleDir(t))

	for _, a := range got.Provenance {
		if a.Pointer != "/definitions/Duration" {
			continue
		}
		assert.True(t, filepath.IsAbs(a.File), "a GOROOT position should stay absolute, got %q", a.File)
		assert.Contains(t, a.File, "time")

		return
	}
	require.Fail(t, "no anchor for the stdlib type pulled in behind the $ref")
}

// TestSpecFormatIsUnchanged is the compatibility guard: the default writes the bare document, which
// is what the WASI tests, the browser probes and any pipeline already consume.
func TestSpecFormatIsUnchanged(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	require.NoError(t, run([]string{
		"-quiet", "-loader=own", "-workdir", moduleDir(t), "./...",
	}, &stdout, &stderr))

	var doc map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc))

	assert.Contains(t, doc, "definitions")
	assert.NotContains(t, doc, "spec", "the bare document must not be wrapped")
	assert.NotContains(t, doc, "provenance")
}

func TestUnknownFormatIsRefused(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := run([]string{"-format=yaml", "-workdir", moduleDir(t), "./..."}, &stdout, &stderr)

	require.Error(t, err)
	assert.ErrorIs(t, err, errBadFlag)
	assert.Empty(t, stdout.String(), "a refused format must not half-write a document")
}

// TestCollectorEmitsEmptyArraysNotNull keeps a consumer from having to check before ranging.
func TestCollectorEmitsEmptyArraysNotNull(t *testing.T) {
	t.Parallel()

	blob, err := json.Marshal((&collector{}).result(map[string]any{}))
	require.NoError(t, err)

	assert.Contains(t, string(blob), `"diagnostics":[]`)
	assert.Contains(t, string(blob), `"provenance":[]`)
}
