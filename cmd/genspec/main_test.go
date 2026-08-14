// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// errUnrecognised stands for an error this command knows nothing about, which is the case the exit
// status has to have an answer for.
var errUnrecognised = errors.New("something else")

// petstore is the corpus's complete example: a meta block, paths, parameters, responses and models.
// A document produced from it exercises every part of this command that is not about flags.
func petstore(t *testing.T) string {
	t.Helper()

	return filepath.Join(fixtures(t), "goparsing", "petstore")
}

// unannotated is a fixture with no swagger:meta block, so the document it produces has no info
// section - which is what makes it invalid on its own, and what makes an -input overlay visible.
func unannotated(t *testing.T) string {
	t.Helper()

	return filepath.Join(fixtures(t), "enhancements", "additional-properties")
}

// fixtures locates the repository's corpus from this file, rather than from the working directory,
// which `go test` is free to set to wherever the package lives.
func fixtures(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve this test's own path")

	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "fixtures"))
}

// exec runs the command as the process would, and reports what it wrote and what it returned.
func exec(t *testing.T, argv ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errs bytes.Buffer
	err = run(argv, &out, &errs)

	return out.String(), errs.String(), err
}

// scanned is exec on a fixture, decoded, for the tests that care about the document rather than
// about the bytes.
func scanned(t *testing.T, dir string, argv ...string) map[string]any {
	t.Helper()

	// Flags first, patterns last: everything after the first positional argument is a pattern, which
	// is the flag package's rule and not one this command is free to soften.
	argv = append([]string{"-workdir", dir, "-quiet", "-config=off"}, argv...)

	stdout, _, err := exec(t, append(argv, "./...")...)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &document))

	return document
}

func TestRunWritesTheDocumentToStandardOutput(t *testing.T) {
	t.Parallel()

	document := scanned(t, petstore(t))

	assert.Equal(t, "2.0", document["swagger"])
	assert.NotEmpty(t, document["paths"], "the petstore describes paths")
	assert.NotEmpty(t, document["definitions"], "and the models they carry")
}

func TestRunWritesToAFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "swagger.json")

	stdout, _, err := exec(t, "-workdir", petstore(t), "-quiet", "-config=off", "-output", path, "./...")
	require.NoError(t, err)
	assert.Empty(t, stdout, "the document went to the file, so nothing goes to standard output")

	written, err := os.ReadFile(path)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(written, &document))
	assert.Equal(t, "2.0", document["swagger"])
}

// TestRunInfersYAMLFromTheOutputName is the half of -format=auto that a caller never states: naming
// a file spec.yaml and getting JSON in it would be a surprise, and the extension is the only place
// the intention was written down.
func TestRunInfersYAMLFromTheOutputName(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "swagger.yaml")

	_, _, err := exec(t, "-workdir", petstore(t), "-quiet", "-config=off", "-output", path, "./...")
	require.NoError(t, err)

	written, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(string(written), "basePath:"),
		"a YAML document, not a JSON one: %.40s", written)
}

func TestRunCompactDropsTheIndentation(t *testing.T) {
	t.Parallel()

	stdout, _, err := exec(t, "-workdir", petstore(t), "-quiet", "-config=off", "-compact", "./...")
	require.NoError(t, err)

	assert.NotContains(t, stdout, "\n  ", "compact means one line, not a differently indented one")
	assert.True(t, json.Valid([]byte(stdout)))
}

// TestRunOverlaysTheInputSpecification covers what -input is for: everything a scanner cannot know.
// The fixture has no meta block, so an info section in the document can only have come from the
// input, and definitions can only have come from the scan.
func TestRunOverlaysTheInputSpecification(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "base.json")
	require.NoError(t, os.WriteFile(base, []byte(
		`{"swagger":"2.0","host":"example.com","info":{"title":"Written by hand","version":"1.0"}}`,
	), 0o600))

	document := scanned(t, unannotated(t), "-input", base)

	assert.Equal(t, "example.com", document["host"], "the input's own content survives the merge")
	assert.NotEmpty(t, document["definitions"], "and the scan's discoveries are written on top of it")
}

func TestRunRefusesADirectoryAsInput(t *testing.T) {
	t.Parallel()

	_, _, err := exec(t, "-workdir", petstore(t), "-quiet", "-config=off", "-input", t.TempDir(), "./...")

	require.ErrorIs(t, err, errUsage)
	assert.Equal(t, exitUsage, exitStatus(err))
}

// TestRunValidateFailsOnAnInvalidDocument is the contract of -validate: the document is still
// written, and the status says what is wrong with it.
func TestRunValidateFailsOnAnInvalidDocument(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := exec(t, "-workdir", unannotated(t), "-validate", "-config=off", "-color=never", "./...")

	require.ErrorIs(t, err, errInvalidSpec)
	assert.Equal(t, exitInvalidSpec, exitStatus(err))
	assert.True(t, json.Valid([]byte(stdout)), "the document is written even when it does not validate")
	assert.Contains(t, stderr, "info in body is required", "and what is wrong with it is reported")
}

func TestRunFailsOnTheThresholdItWasGiven(t *testing.T) {
	t.Parallel()

	_, _, err := exec(t, "-workdir", unannotated(t), "-fail-on=warning", "-config=off", "-color=never", "./...")

	require.ErrorIs(t, err, errDiagnostics)
	assert.Equal(t, exitDiagnostics, exitStatus(err))
}

// TestRunIsSilentByDefaultOnACleanScan states what a command in a pipeline can count on: nothing on
// standard error unless there is something to say.
func TestRunIsSilentByDefaultOnACleanScan(t *testing.T) {
	t.Parallel()

	_, stderr, err := exec(t, "-workdir", petstore(t), "-color=never", "-config=off", "./...")

	require.NoError(t, err)
	assert.Empty(t, stderr)
}

func TestRunQuietSaysNothing(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := exec(t, "-workdir", unannotated(t), "-quiet", "-config=off", "-verbose", "./...")

	require.NoError(t, err)
	assert.NotEmpty(t, stdout, "-quiet is about the diagnostics, not about the document")
	assert.Empty(t, stderr)
}

func TestRunReportsPositionsRelativeToTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	_, stderr, err := exec(t, "-workdir", unannotated(t), "-color=never", "-config=off", "./...")

	require.NoError(t, err)
	require.NotEmpty(t, stderr, "this fixture has something to say about itself")
	assert.Contains(t, stderr, "file=api.go", "a path the caller could open, not an absolute one")
}

func TestRunPrintsItsVersion(t *testing.T) {
	t.Parallel()

	stdout, _, err := exec(t, "-version")

	require.NoError(t, err)
	assert.Contains(t, stdout, "genspec")
}

// TestRunHelpIsNotAFailure keeps -h at status 0. flag reports it as an error, and treating it as one
// makes a command that cannot be asked what it does without failing.
func TestRunHelpIsNotAFailure(t *testing.T) {
	t.Parallel()

	_, stderr, err := exec(t, "-h")

	require.ErrorIs(t, err, flag.ErrHelp)
	assert.Contains(t, stderr, "usage: genspec")
	assert.Contains(t, stderr, "packages are Go patterns")
}

func TestRunRejectsAnUnknownFlag(t *testing.T) {
	t.Parallel()

	_, _, err := exec(t, "-not-a-flag")

	require.ErrorIs(t, err, errUsage)
	assert.Equal(t, exitUsage, exitStatus(err))
}

func TestRunRejectsAnUnreadableWorkdir(t *testing.T) {
	t.Parallel()

	_, _, err := exec(t, "-workdir", filepath.Join(t.TempDir(), "nowhere"), "-quiet", "-config=off", "./...")

	require.Error(t, err)
	assert.Equal(t, exitFailed, exitStatus(err))
}

func TestExitStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, exitOK, exitStatus(nil))
	assert.Equal(t, exitUsage, exitStatus(errUsage))
	assert.Equal(t, exitDiagnostics, exitStatus(errDiagnostics))
	assert.Equal(t, exitInvalidSpec, exitStatus(errInvalidSpec))
	assert.Equal(t, exitFailed, exitStatus(errUnrecognised))
}
