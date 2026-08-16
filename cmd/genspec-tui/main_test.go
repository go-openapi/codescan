// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// What a test can drive of this command stops where the session begins: a run that gets as far as its options takes
// over the terminal it was started from and does not come back until the user quits.
//
// So every case here has to be one that answers, or refuses, BEFORE that - which is also why each says which error it
// expects rather than merely that there was one. A case that quietly stopped refusing would otherwise still pass, on
// the TTY error a terminal-less CI machine happens to raise, while doing the one thing these tests must not do.
//
// What happens once a session is running is the ux package's to test, and it tests it through the model rather than
// through a program.

func TestRun(t *testing.T) {
	t.Run("should print version", func(t *testing.T) {
		t.Parallel()

		stdout, _, err := exec(t, "-version")

		require.NoError(t, err)
		assert.Contains(t, stdout, cmd)
	})

	t.Run("should provide help without error", func(t *testing.T) {
		// Unlike genspec, which carries flag.ErrHelp out so that its exit-status ladder can answer 0 for it, this
		// command has no ladder: asking what it does is not an error and does not read as one.
		t.Parallel()

		stdout, stderr, err := exec(t, "-h")

		require.NoError(t, err)
		assert.Empty(t, stdout, "the usage is a message about the command, not its output")
		assert.Contains(t, stderr, "usage: "+cmd)
		assert.Contains(t, stderr, "packages are Go patterns")
		assert.Contains(t, stderr, cliconf.SampleConfigName(), "and it says a file can preset all this")
	})

	t.Run("should reject an unknown flag", func(t *testing.T) {
		t.Parallel()

		_, stderr, err := exec(t, "-not-a-flag")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not-a-flag")
		assert.Contains(t, stderr, "usage: "+cmd, "which is when the usage is worth printing unasked")
	})

	t.Run("should reject a flag value it does not accept", func(t *testing.T) {
		// The shared table validates its own values, and the command carries the verdict out rather than starting a
		// session that would scan through a loader nobody named.
		t.Parallel()

		_, _, err := exec(t, "-no-config", "-loader", "cargo")

		require.ErrorIs(t, err, cliopts.ErrBadFlag)
	})
}

// A file this command cannot obey is refused before the session starts, rather than half-applied to it: the status
// line reports what a file set, and a report of a file that was only partly read would be worse than no file.
func TestRunConfigFile(t *testing.T) {
	t.Run("should refuse an unknown key", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", configFile(t, "scan:\n  workdirs: ./api\n"))

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
		assert.Contains(t, err.Error(), "scan.workdirs")
	})

	t.Run("should refuse a value of the wrong kind", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", configFile(t, "emit:\n  scan-models: sometimes\n"))

		require.ErrorIs(t, err, cliconf.ErrBadValue)
	})

	t.Run("should refuse a file that is not there", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", filepath.Join(t.TempDir(), "absent.yaml"))

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
	})

	t.Run("should refuse being asked for a file and for none", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", configFile(t, "scan:\n  workdir: .\n"), "-no-config")

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
	})

	t.Run("should refuse a file naming another file", func(t *testing.T) {
		t.Parallel()

		_, _, err := exec(t, "-config", configFile(t, "scan:\n  config: elsewhere.yaml\n"))

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
	})
}

/********************************/
/* exec helpers */
/********************************/

// exec runs the command as the process would, and reports what it wrote and what it returned.
//
// NOTE: main's os.Exit is not simulated - a non-nil error is what the process exits 1 on.
func exec(t *testing.T, argv ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errs bytes.Buffer
	err = run(argv, &out, &errs)

	return out.String(), errs.String(), err
}

// configFile writes a configuration file in a fresh directory and reports its path.
//
// Named on the command line rather than left to be discovered, so that a test says which file it means and none of
// these depends on where it was run from.
func configFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}
