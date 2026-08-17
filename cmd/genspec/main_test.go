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
	"strings"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec/internal/clitest/fixtures"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/internal/testloader"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// errUnrecognised stands for an error this command knows nothing about,
// which is the case the exit status has to have an answer for.
var errUnrecognised = errors.New("something else")

func TestRun(t *testing.T) {
	t.Run("should write document to stdout", func(t *testing.T) {
		t.Parallel()

		document := scanned(t, fixtures.Petstore(t))

		assert.Equal(t, "2.0", document["swagger"])
		assert.NotEmpty(t, document["paths"], "the petstore describes paths")
		assert.NotEmpty(t, document["definitions"], "and the models they carry")
	})

	t.Run("should write document to a file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "swagger.json")

		stdout, _, err := exec(t, "-workdir", fixtures.Petstore(t), "-quiet", "-no-config", "-output", path, "./...")
		require.NoError(t, err)
		assert.Empty(t, stdout, "the document went to the file, so nothing goes to standard output")

		written, err := os.ReadFile(path)
		require.NoError(t, err)

		var document map[string]any
		require.NoError(t, json.Unmarshal(written, &document))
		assert.Equal(t, "2.0", document["swagger"])
	})

	t.Run("should infer YAML from the output", func(t *testing.T) {
		// This is the half of -format=auto that a caller never states: naming a file spec.yaml and getting JSON in it
		// would be a surprise, and the extension is the only place the intention was written down.
		t.Parallel()

		path := filepath.Join(t.TempDir(), "swagger.yaml")

		_, _, err := exec(t, "-workdir", fixtures.Petstore(t), "-quiet", "-no-config", "-output", path, "./...")
		require.NoError(t, err)

		written, err := os.ReadFile(path)
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(string(written), "basePath:"),
			"a YAML document, not a JSON one: %.40s", written)
	})

	t.Run("should drop indentation on compact", func(t *testing.T) {
		t.Parallel()

		stdout, _, err := exec(t, "-workdir", fixtures.Petstore(t), "-quiet", "-no-config", "-compact", "./...")
		require.NoError(t, err)

		assert.NotContains(t, stdout, "\n  ", "compact means one line, not a differently indented one")
		assert.True(t, json.Valid([]byte(stdout)))
	})

	t.Run("should overlay input spec", func(t *testing.T) {
		// Covers what -input is for: everything a scanner cannot know.
		// The fixture has no meta block, so an info section in the document can only have come from the
		// input, and definitions can only have come from the scan.
		t.Parallel()

		base := filepath.Join(t.TempDir(), "base.json")
		require.NoError(t, os.WriteFile(base, []byte(
			`{"swagger":"2.0","host":"example.com","info":{"title":"Written by hand","version":"1.0"}}`,
		), 0o600))

		document := scanned(t, fixtures.Unannotated(t), "-input", base)

		assert.Equal(t, "example.com", document["host"], "the input's own content survives the merge")
		assert.NotEmpty(t, document["definitions"], "and the scan's discoveries are written on top of it")
	})

	t.Run("should refuse a directory as input", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-workdir", fixtures.Petstore(t), "-quiet", "-no-config", "-input", t.TempDir(), "./...")

		require.ErrorIs(t, err, sentinel.ErrUsage)
		assert.Equal(t, exitUsage, exitStatus(err))
	})

	t.Run("should fail when -validate an invalid spec", func(t *testing.T) {
		// TestRunValidateFailsOnAnInvalidDocument is the contract of -validate: the document is still written,
		// and the status says what is wrong with it.
		t.Parallel()

		stdout, stderr, err := exec(t, "-workdir", fixtures.Unannotated(t), "-validate", "-no-config", "-color=never", "./...")

		require.ErrorIs(t, err, sentinel.ErrInvalidSpec)
		assert.Equal(t, exitInvalidSpec, exitStatus(err))
		assert.True(t, json.Valid([]byte(stdout)), "the document is written even when it does not validate")
		assert.Contains(t, stderr, "info in body is required", "and what is wrong with it is reported")
	})

	t.Run("should fail when on -fail-on is given", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-workdir", fixtures.Unannotated(t), "-fail-on=warning", "-no-config", "-color=never", "./...")

		require.ErrorIs(t, err, sentinel.ErrDiagnostics)
		assert.Equal(t, exitDiagnostics, exitStatus(err))
	})

	t.Run("should remain silent on clean scan", func(t *testing.T) {
		// States what a command in a pipeline can count on: nothing on standard error unless there is something to say.
		t.Parallel()

		_, stderr, err := exec(t, "-workdir", fixtures.Petstore(t), "-color=never", "-no-config", "./...")
		require.NoError(t, err)

		if loaderHasSomethingToSay() {
			assert.Contains(t, stderr, "finished hints=1")
		} else {
			assert.Empty(t, stderr)
		}
	})

	t.Run("should remain silent when quiet", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := exec(t, "-workdir", fixtures.Unannotated(t), "-quiet", "-no-config", "-verbose", "./...")

		require.NoError(t, err)
		assert.NotEmpty(t, stdout, "-quiet is about the diagnostics, not about the document")
		assert.Empty(t, stderr)
	})

	t.Run("should report with positions relative to the working directory", func(t *testing.T) {
		t.Parallel()

		_, stderr, err := exec(t, "-workdir", fixtures.Unannotated(t), "-color=never", "-no-config", "./...")

		require.NoError(t, err)
		require.NotEmpty(t, stderr, "this fixture has something to say about itself")
		assert.Contains(t, stderr, "file=api.go", "a path the caller could open, not an absolute one")
	})

	t.Run("should print version", func(t *testing.T) {
		t.Parallel()

		stdout, _, err := exec(t, "-version")

		require.NoError(t, err)
		assert.Contains(t, stdout, "genspec")
	})

	t.Run("should provide help without error", func(t *testing.T) {
		// Keeps -h at status 0. flag reports it as an error, and treating it as one
		// makes a command that cannot be asked what it does without failing.
		//
		// The parser raises an error, but the exit code is 0.
		t.Parallel()

		_, stderr, err := exec(t, "-h")

		require.ErrorIs(t, err, flag.ErrHelp)
		assert.Contains(t, stderr, "usage: genspec")
		assert.Contains(t, stderr, "packages are Go patterns")

		require.Equal(t, 0, exitStatus(err))
	})

	t.Run("should reject an unknown flag", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-not-a-flag")

		require.ErrorIs(t, err, sentinel.ErrUsage)
		assert.Equal(t, exitUsage, exitStatus(err))
	})

	t.Run("should reject an unreadable workdir", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-workdir", filepath.Join(t.TempDir(), "nowhere"), "-quiet", "-no-config", "./...")

		require.Error(t, err)
		assert.Equal(t, exitFailed, exitStatus(err))
	})
}

func TestExitStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, exitOK, exitStatus(nil))
	assert.Equal(t, exitUsage, exitStatus(sentinel.ErrUsage))
	assert.Equal(t, exitDiagnostics, exitStatus(sentinel.ErrDiagnostics))
	assert.Equal(t, exitInvalidSpec, exitStatus(sentinel.ErrInvalidSpec))
	assert.Equal(t, exitFailed, exitStatus(errUnrecognised))
}

func TestConfigFileSetsFlags(t *testing.T) {
	t.Parallel()

	t.Run("should set flags from config file", func(t *testing.T) {
		t.Parallel()
		path := fixtures.ConfigFile(t, `
scan:
  workdir: `+fixtures.Petstore(t)+`
emit:
  scan-models: false
document:
  compact: true
diagnostics:
  quiet: true
`)

		stdout, _, err := exec(t, "-config", path, "./...")
		require.NoError(t, err)

		assert.NotContains(t, stdout, "\n  ", "document.compact came from the file")
		assert.True(t, json.Valid([]byte(stdout)))
	})

	t.Run("flag should beat config file", func(t *testing.T) {
		// This is the precedence rule, end to end.
		t.Parallel()

		path := fixtures.ConfigFile(t, `
document:
  compact: true
diagnostics:
  quiet: true
`)

		stdout, _, err := exec(t, "-config", path, "-workdir", fixtures.Petstore(t), "-compact=false", "./...")
		require.NoError(t, err)

		assert.Contains(t, stdout, "\n  ", "-compact=false was typed, so the file does not get a say")
	})

	t.Run("with flag -noconfig should ignore the config file", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "document:\n  compact: true\n")

		stdout, _, err := exec(t, "-no-config", "-workdir", fixtures.Petstore(t), "-quiet", "./...")
		require.NoError(t, err)
		require.FileExists(t, path)

		assert.Contains(t, stdout, "\n  ", "the file exists, and was not consulted")
	})

	t.Run("should not report about config when quiet", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "scan:\n  workdir: "+fixtures.Petstore(t)+"\ndocument:\n  compact: true\n")

		stdout, _, err := exec(t, "-c", path, "-quiet", "./...")

		require.NoError(t, err)
		assert.NotContains(t, stdout, "\n  ", "the file -c named was read")
	})

	t.Run("should error on contradictory config flags", func(t *testing.T) {
		// the pair of flags can be asked something with no answer
		t.Parallel()

		path := fixtures.ConfigFile(t, "scan:\n  workdir: .\n")

		_, _, err := exec(t, "-c", path, "-no-config", "./...")

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
		assert.Equal(t, exitUsage, exitStatus(err))
	})

	t.Run("should refuse unknown key", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "scan:\n  workdirs: ./api\n")

		_, _, err := exec(t, "-config", path, "./...")

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
		assert.Equal(t, exitUsage, exitStatus(err), "a file that cannot be obeyed is a usage error")
		assert.Contains(t, err.Error(), "scan.workdirs")
	})

	t.Run("should refuse invalid value", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "emit:\n  scan-models: sometimes\n")

		_, _, err := exec(t, "-config", path, "./...")

		require.ErrorIs(t, err, cliconf.ErrBadValue)
		assert.Equal(t, exitUsage, exitStatus(err))
	})

	t.Run("file must exist if named explicitly", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", filepath.Join(t.TempDir(), "absent.yaml"), "./...")

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
		assert.Equal(t, exitUsage, exitStatus(err))
	})

	t.Run("should report whas is skipped", func(t *testing.T) {
		// This is how a section meant for another command - or a misspelled one - stays findable.
		t.Parallel()

		path := fixtures.ConfigFile(t, `
scan:
  workdir: `+fixtures.Petstore(t)+`
tui:
  theme: dark
`)

		_, stderr, err := exec(t, "-config", path, "-verbose", "-color=never", "./...")

		require.NoError(t, err)
		assert.Contains(t, stderr, "configuration read")
		assert.Contains(t, stderr, "tui.theme")
	})

	t.Run("should not report about config when not verbose", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "scan:\n  workdir: "+fixtures.Petstore(t)+"\n")

		_, stderr, err := exec(t, "-config", path, "-color=never", "./...")

		require.NoError(t, err)
		if loaderHasSomethingToSay() {
			assert.Contains(t, stderr, "finished hints=1")
		} else {
			assert.Empty(t, stderr, "reading a configuration file is not news")
		}
	})

	t.Run("should reject config pointing to another config", func(t *testing.T) {
		t.Parallel()

		path := fixtures.ConfigFile(t, "scan:\n  config: elsewhere.yaml\n")

		_, _, err := exec(t, "-config", path, "./...")

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
	})
}

func TestConfigFileWalkUp(t *testing.T) {
	// NOT PARALLEL

	t.Run("should find config file by walking up", func(t *testing.T) {
		// Covers the path a user actually takes: a file in the project, and a command run from inside it.
		//
		// Not parallel: it is about the process's working directory, which is not a thing a test can hold an opinion on privately.
		project := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(project, cliconf.Names[0]),
			[]byte("scan:\n  workdir: "+fixtures.Petstore(t)+"\ndiagnostics:\n  quiet: true\n"), 0o600))

		deep := filepath.Join(project, "cmd", "api")
		require.NoError(t, os.MkdirAll(deep, 0o750))
		t.Chdir(deep)

		stdout, _, err := exec(t, "./...")

		require.NoError(t, err)
		assert.Contains(t, stdout, `"swagger": "2.0"`, "the scan ran where the file said")
	})
}

/********************************/
/* exec helpers */
/********************************/

// loaderHasSomethingToSay reports whether the loader this run uses reports a hint of its own on a
// clean scan, which the silence assertions have to allow for.
//
// Both non-default loaders do, for unrelated reasons: compiled dependencies announce that the
// request was met, and the toolchain-free loader says it did not run the cgo tool. Neither is a
// finding about the scanned code, which is what those assertions are really about -- so the
// exception is the loader having spoken, not any particular thing it said.
func loaderHasSomethingToSay() bool {
	return testloader.Selected() != testloader.LoaderSource
}

// exec runs the command as the process would, and reports what it wrote and what it returned.
//
// NOTE: this does not simulate the exit code returned to the shell: you have to check exitStatus() for that.
func exec(t *testing.T, argv ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errs bytes.Buffer
	const maxExtraArgs = 4
	argsWithLoader := make([]string, 0, len(argv)+maxExtraArgs)
	argsWithLoader = append(argsWithLoader, testloader.Flags()...)
	argsWithLoader = append(argsWithLoader, argv...)
	err = run(argsWithLoader, &out, &errs)

	return out.String(), errs.String(), err
}

// scanned is exec on a fixture, decoded, for the tests that care about the document rather than about the bytes.
func scanned(t *testing.T, dir string, argv ...string) map[string]any {
	t.Helper()

	// Flags first, patterns last: everything after the first positional argument is a pattern, which
	// is the flag package's rule and not one this command is free to soften.
	argv = append([]string{"-workdir", dir, "-quiet", "-no-config"}, argv...)

	stdout, _, err := exec(t, append(argv, "./...")...)
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &document))

	return document
}
