// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan/internal/cliconf"
)

// The scan options are declared in internal/cliopts and guarded there: the drift guard this command used to carry
// only ever checked its own copy of the table, which is how it came to expose fourteen options out of thirty.
//
// What is left to check here is what this command adds on top, and how the two halves meet.

func TestEveryFlagIsAddressableInAConfigFile(t *testing.T) {
	t.Parallel()

	cli := newTestFlags(t)

	schema, err := configSchema()
	require.NoError(t, err)

	cli.set.VisitAll(func(f *flag.Flag) {
		if reason, excused := notConfigurable[f.Name]; excused {
			assert.NotContainsf(t, schema, f.Name,
				"-%s is excused from configuration (%s) but the schema addresses it anyway", f.Name, reason)

			return
		}

		assert.Containsf(t, schema, f.Name,
			"flag -%s is addressed in no configuration section. Add it to commandSections, or excuse it in "+
				"notConfigurable with a reason.", f.Name)
	})
}

// The patterns are positional, as they are for genspec and for every other Go command.
func TestPatternsComeFromTheArguments(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name string
		args []string
		want []string
	}{
		{"named as arguments", []string{"./api/...", "./models"}, []string{"./api/...", "./models"}},
		{"none names everything", nil, []string{"./..."}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestFlags(t)
			require.NoError(t, cli.set.Parse(c.args))

			opts, err := cli.options(cli.set.Args())
			require.NoError(t, err)
			assert.Equal(t, c.want, opts.Packages)
		})
	}
}

// -packages is the spelling the README carried since the first release, so it keeps working - and loses to an
// argument, which is the spelling that replaced it.
func TestPackagesFlagStillWorks(t *testing.T) {
	t.Parallel()

	t.Run("stands in when nothing is named", func(t *testing.T) {
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-packages", "./api/..., ./models"}))

		opts, err := cli.options(cli.set.Args())
		require.NoError(t, err)
		assert.Equal(t, []string{"./api/...", "./models"}, opts.Packages, "entries are trimmed")
	})

	t.Run("an argument wins", func(t *testing.T) {
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-packages", "./old/...", "./new/..."}))

		opts, err := cli.options(cli.set.Args())
		require.NoError(t, err)
		assert.Equal(t, []string{"./new/..."}, opts.Packages)
	})
}

// The tree, the watcher and every reported position hang off WorkDir, and a relative one would leave each of them to
// resolve it against whatever they happened to be near.
func TestWorkdirIsResolved(t *testing.T) {
	t.Parallel()

	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse([]string{"-workdir", "."}))

	opts, err := cli.options(nil)
	require.NoError(t, err)

	assert.True(t, filepath.IsAbs(opts.WorkDir), "got %q", opts.WorkDir)
}

// The whole shared surface is reachable here, which is the point of the move: the TUI used to register fourteen of
// these and silently lack the rest.
func TestTheSharedOptionsAreAllRegistered(t *testing.T) {
	t.Parallel()

	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse([]string{
		"-scan-models=false",
		"-prune-unused-models",
		"-emit-ref-siblings",
		"-clean-go-doc",
		"-name-from-tags", "form,json",
		"-loader", "own",
	}))

	opts, err := cli.options(nil)
	require.NoError(t, err)

	assert.False(t, opts.ScanModels)
	assert.True(t, opts.PruneUnusedModels)
	assert.True(t, opts.EmitRefSiblings)
	assert.True(t, opts.CleanGoDoc)
	assert.Equal(t, []string{"form", "json"}, opts.NameFromTags)
	assert.True(t, opts.ToolchainFreeLoader, "-loader own is the tri-state spelling of it")
}

func TestProfilingIsOffUnlessAsked(t *testing.T) {
	t.Parallel()

	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse(nil))

	prof, err := cli.profiling()
	require.NoError(t, err)

	assert.False(t, prof.Enabled)
	assert.Empty(t, prof.Dir, "nothing is created for a session that did not ask to be profiled")
}

func TestProfilingTakesADirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", dir}))

	prof, err := cli.profiling()
	require.NoError(t, err)

	assert.True(t, prof.Enabled)
	assert.Equal(t, dir, prof.Dir)
}

// The heap sampling rate is a property of the process, so it is applied once, here, rather than per scan.
func TestProfilingAppliesTheSamplingRate(t *testing.T) {
	saved := runtime.MemProfileRate
	t.Cleanup(func() { runtime.MemProfileRate = saved })

	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", t.TempDir(), "-mem-profile-rate", "1"}))

	_, err := cli.profiling()
	require.NoError(t, err)

	assert.Equal(t, 1, runtime.MemProfileRate)
}

func TestConfigFileSetsFlags(t *testing.T) {
	cli, path := inConfiguredDir(t, `
scan:
  workdir: ./api
emit:
  scan-models: false
profile:
  mem-profile-rate: 1
  profile: true
`)
	require.NoError(t, cli.set.Parse(nil))

	applied, got, err := configured(cli.set, cli.configFile)
	require.NoError(t, err)
	requireSameFile(t, path, got)
	assert.ElementsMatch(t, []string{"workdir", "scan-models", "profile", "mem-profile-rate"}, applied.Set)

	opts, err := cli.options(nil)
	require.NoError(t, err)
	assert.Equal(t, "api", filepath.Base(opts.WorkDir), "a section the library owns")
	assert.False(t, opts.ScanModels)
	assert.True(t, *cli.profile, "and a section this command owns")
}

// The precedence rule: a file presets, a command line decides.
func TestTypedFlagsBeatTheConfigFile(t *testing.T) {
	cli, _ := inConfiguredDir(t, "emit:\n  scan-models: false\n")
	require.NoError(t, cli.set.Parse([]string{"-scan-models=true"}))

	_, _, err := configured(cli.set, cli.configFile)
	require.NoError(t, err)

	opts, err := cli.options(nil)
	require.NoError(t, err)
	assert.True(t, opts.ScanModels)
}

func TestNoConfigIgnoresTheFile(t *testing.T) {
	cli, path := inConfiguredDir(t, "emit:\n  scan-models: false\n")
	require.NoError(t, cli.set.Parse([]string{"-no-config"}))

	applied, got, err := configured(cli.set, cli.configFile)
	require.NoError(t, err)
	require.FileExists(t, path, "the file is there; it was simply not consulted")

	assert.Empty(t, got)
	assert.Empty(t, applied.Set)
}

// A key nobody can act on reads exactly like a setting that quietly never applied, which is what the schema is for.
func TestConfigFileRefusesAnUnknownKey(t *testing.T) {
	cli, _ := inConfiguredDir(t, "emit:\n  scan-modles: false\n")
	require.NoError(t, cli.set.Parse(nil))

	_, _, err := configured(cli.set, cli.configFile)

	require.ErrorIs(t, err, cliconf.ErrUnknownKey)
}

// Another command's half of a shared file is skipped rather than refused - that is what makes the file shareable.
func TestConfigFileSkipsAnotherCommandsSection(t *testing.T) {
	cli, _ := inConfiguredDir(t, "document:\n  output: spec.json\nemit:\n  scan-models: false\n")
	require.NoError(t, cli.set.Parse(nil))

	applied, _, err := configured(cli.set, cli.configFile)
	require.NoError(t, err)

	assert.Equal(t, []string{"scan-models"}, applied.Set)
	assert.Equal(t, []string{"document.output"}, applied.Ignored, "genspec's half, reported rather than dropped")
}

// inConfiguredDir writes a configuration file into a fresh directory and runs the test from inside it, which is how
// the file is found: the search walks up from where the command was started.
func inConfiguredDir(t *testing.T, content string) (*config, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	t.Chdir(dir)

	return newTestFlags(t), path
}

// requireSameFile says the discovery found the file the test wrote, without saying how it is spelled.
//
// The search joins names onto the working directory as the process reports it and resolves nothing, so the spelling is
// the one os.Getwd hands back - which on macOS is the temporary directory reached through the /var symlink, where
// t.TempDir names the same file under /private/var. Neither is more correct than the other, and which one comes back
// is not something this package decides.
func requireSameFile(t *testing.T, want, got string) {
	t.Helper()

	wantInfo, err := os.Stat(want)
	require.NoError(t, err)

	gotInfo, err := os.Stat(got)
	require.NoError(t, err)

	require.True(t, os.SameFile(wantInfo, gotInfo), "%q and %q should name the same file", want, got)
}

func newTestFlags(t *testing.T) *config {
	t.Helper()

	fs := flag.NewFlagSet("genspec-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return registerFlags(fs)
}
