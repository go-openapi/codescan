// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/internal/cliconf"
)

// configFile writes a configuration file in a fresh directory and reports its path.
func configFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// TestEveryFlagIsAddressableInAConfigFile is the config-file half of the coverage guard, for the
// flags this command adds to the shared ones.
//
// A flag with no section can be typed but not configured, and nothing would notice: from the outside
// that reads exactly like the option not existing.
func TestEveryFlagIsAddressableInAConfigFile(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("genspec", flag.ContinueOnError)
	fs.SetOutput(nil)
	registerFlags(fs)

	schema, err := configSchema()
	require.NoError(t, err)

	fs.VisitAll(func(f *flag.Flag) {
		if reason, excused := notConfigurable[f.Name]; excused {
			assert.NotContainsf(t, schema, f.Name,
				"-%s is excused from configuration (%s) but the schema addresses it anyway", f.Name, reason)

			return
		}

		assert.Containsf(t, schema, f.Name,
			"flag -%s is addressed in no configuration section. Add it to commandSections, or excuse "+
				"it in notConfigurable with a reason.", f.Name)
	})
}

func TestConfigFileSetsFlags(t *testing.T) {
	t.Parallel()

	path := configFile(t, `
scan:
  workdir: `+petstore(t)+`
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
}

// TestTypedFlagsBeatTheConfigFile is the precedence rule, end to end.
func TestTypedFlagsBeatTheConfigFile(t *testing.T) {
	t.Parallel()

	path := configFile(t, `
document:
  compact: true
diagnostics:
  quiet: true
`)

	stdout, _, err := exec(t, "-config", path, "-workdir", petstore(t), "-compact=false", "./...")
	require.NoError(t, err)

	assert.Contains(t, stdout, "\n  ", "-compact=false was typed, so the file does not get a say")
}

func TestNoConfigIgnoresTheFile(t *testing.T) {
	t.Parallel()

	path := configFile(t, "document:\n  compact: true\n")

	stdout, _, err := exec(t, "-no-config", "-workdir", petstore(t), "-quiet", "./...")
	require.NoError(t, err)
	require.FileExists(t, path)

	assert.Contains(t, stdout, "\n  ", "the file exists, and was not consulted")
}

// TestShortConfigFlagPinsTheFile covers the spelling a caller reaches for often enough to want short.
func TestShortConfigFlagPinsTheFile(t *testing.T) {
	t.Parallel()

	path := configFile(t, "scan:\n  workdir: "+petstore(t)+"\ndocument:\n  compact: true\n")

	stdout, _, err := exec(t, "-c", path, "-quiet", "./...")

	require.NoError(t, err)
	assert.NotContains(t, stdout, "\n  ", "the file -c named was read")
}

// TestConfigAndNoConfigContradictEachOther: the pair of flags can be asked something with no
// answer, and saying so beats picking one.
func TestConfigAndNoConfigContradictEachOther(t *testing.T) {
	t.Parallel()

	path := configFile(t, "scan:\n  workdir: .\n")

	_, _, err := exec(t, "-c", path, "-no-config", "./...")

	require.ErrorIs(t, err, cliconf.ErrBadConfig)
	assert.Equal(t, exitUsage, exitStatus(err))
}

// TestConfigFileIsFoundByWalkingUp covers the path a user actually takes: a file in the project,
// and a command run from inside it.
//
// Not parallel: it is about the process's working directory, which is not a thing a test can hold
// an opinion on privately.
func TestConfigFileIsFoundByWalkingUp(t *testing.T) {
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, cliconf.Names[0]),
		[]byte("scan:\n  workdir: "+petstore(t)+"\ndiagnostics:\n  quiet: true\n"), 0o600))

	deep := filepath.Join(project, "cmd", "api")
	require.NoError(t, os.MkdirAll(deep, 0o750))
	t.Chdir(deep)

	stdout, _, err := exec(t, "./...")

	require.NoError(t, err)
	assert.Contains(t, stdout, `"swagger": "2.0"`, "the scan ran where the file said")
}

func TestConfigFileRefusesAnUnknownKey(t *testing.T) {
	t.Parallel()

	path := configFile(t, "scan:\n  workdirs: ./api\n")

	_, _, err := exec(t, "-config", path, "./...")

	require.ErrorIs(t, err, cliconf.ErrUnknownKey)
	assert.Equal(t, exitUsage, exitStatus(err), "a file that cannot be obeyed is a usage error")
	assert.Contains(t, err.Error(), "scan.workdirs")
}

func TestConfigFileRefusesAValueTheFlagWillNotTake(t *testing.T) {
	t.Parallel()

	path := configFile(t, "emit:\n  scan-models: sometimes\n")

	_, _, err := exec(t, "-config", path, "./...")

	require.ErrorIs(t, err, cliconf.ErrBadValue)
	assert.Equal(t, exitUsage, exitStatus(err))
}

func TestConfigFileMustExistWhenNamed(t *testing.T) {
	t.Parallel()

	_, _, err := exec(t, "-config", filepath.Join(t.TempDir(), "absent.yaml"), "./...")

	require.ErrorIs(t, err, cliconf.ErrBadConfig)
	assert.Equal(t, exitUsage, exitStatus(err))
}

// TestConfigFileReportsWhatItSkipped is how a section meant for another command - or a misspelled
// one - stays findable.
func TestConfigFileReportsWhatItSkipped(t *testing.T) {
	t.Parallel()

	path := configFile(t, `
scan:
  workdir: `+petstore(t)+`
tui:
  theme: dark
`)

	_, stderr, err := exec(t, "-config", path, "-verbose", "-color=never", "./...")

	require.NoError(t, err)
	assert.Contains(t, stderr, "configuration read")
	assert.Contains(t, stderr, "tui.theme")
}

func TestConfigFileIsSilentWithoutVerbose(t *testing.T) {
	t.Parallel()

	path := configFile(t, "scan:\n  workdir: "+petstore(t)+"\n")

	_, stderr, err := exec(t, "-config", path, "-color=never", "./...")

	require.NoError(t, err)
	assert.Empty(t, stderr, "reading a configuration file is not news")
}

// TestConfigFileCannotNameAnotherConfigFile pins the one flag a file may not set.
func TestConfigFileCannotNameAnotherConfigFile(t *testing.T) {
	t.Parallel()

	path := configFile(t, "scan:\n  config: elsewhere.yaml\n")

	_, _, err := exec(t, "-config", path, "./...")

	require.ErrorIs(t, err, cliconf.ErrUnknownKey)
}

func TestConfigSchemaCoversBothHalves(t *testing.T) {
	t.Parallel()

	schema, err := configSchema()
	require.NoError(t, err)

	assert.Equal(t, "scan", schema["workdir"], "the library's own flags")
	assert.Equal(t, "emit", schema["scan-models"])
	assert.Equal(t, sectionDocument, schema["output"], "and this command's")
	assert.Equal(t, sectionDiagnostics, schema["fail-on"])
	assert.Equal(t,
		[]string{sectionDiagnostics, sectionDocument, "emit", "go", "load", "scan"},
		schema.Sections())
}
