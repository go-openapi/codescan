// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The scan options are declared in internal/cliopts and guarded there: the drift guard this command used to carry
// only ever checked its own copy of the table, which is how it came to expose fourteen options out of thirty.
//
// What is left to check here is what this command adds on top, and how the two halves meet.

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("should take the patterns from the arguments", func(t *testing.T) {
		// Positional, as they are for genspec and for every other Go command.
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
	})

	t.Run("should still answer to -packages", func(t *testing.T) {
		// The spelling the README carried since the first release, so it keeps working.
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-packages", "./api/..., ./models"}))

		opts, err := cli.options(cli.set.Args())
		require.NoError(t, err)
		assert.Equal(t, []string{"./api/...", "./models"}, opts.Packages, "entries are trimmed")
	})

	t.Run("should let an argument beat -packages", func(t *testing.T) {
		// Which is the spelling that replaced it.
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-packages", "./old/...", "./new/..."}))

		opts, err := cli.options(cli.set.Args())
		require.NoError(t, err)
		assert.Equal(t, []string{"./new/..."}, opts.Packages)
	})

	t.Run("should resolve -workdir once, here", func(t *testing.T) {
		// The tree, the watcher and every reported position hang off WorkDir, and a relative one would leave each of
		// them to resolve it against whatever they happened to be near.
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-workdir", "."}))

		opts, err := cli.options(nil)
		require.NoError(t, err)

		assert.True(t, filepath.IsAbs(opts.WorkDir), "got %q", opts.WorkDir)
	})

	t.Run("should reach the whole shared surface", func(t *testing.T) {
		// Which is the point of the move: the TUI used to register fourteen of these and silently lack the rest.
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
	})

	t.Run("should refuse a value it does not accept", func(t *testing.T) {
		// Refused here rather than carried into a session, where a scan would fail on every rescan for a reason the
		// command line could have given straight away.
		t.Parallel()

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-loader=cargo"}))

		_, err := cli.options(nil)

		require.ErrorIs(t, err, cliopts.ErrBadFlag)
	})
}

// PrepareScan is the whole command line settled at once, and the only place that knows a value could have come from
// anywhere but a flag.
func TestPrepareScan(t *testing.T) {
	// NOT PARALLEL: the file is found by searching upwards from the working directory.

	t.Run("should settle the file, then what was typed, then the arguments", func(t *testing.T) {
		profiles := t.TempDir()
		cli, path := inConfiguredDir(t, `
scan:
  workdir: ./api
emit:
  scan-models: false
profile:
  profile-dir: `+profiles+`
  profile: true
`)
		require.NoError(t, cli.set.Parse([]string{"-scan-models=true", "./handlers/..."}))

		opts, prof, err := cli.PrepareScan()
		require.NoError(t, err)

		assert.Equal(t, []string{"./handlers/..."}, opts.Packages, "the arguments name what to scan")
		assert.Equal(t, "api", filepath.Base(opts.WorkDir), "the file says from where")
		assert.True(t, filepath.IsAbs(opts.WorkDir), "resolved once, here, got %q", opts.WorkDir)
		assert.True(t, opts.ScanModels, "and what was typed wins over what the file asked for")

		require.NotNil(t, prof, "the file asked for a profiled session")
		assert.True(t, prof.Enabled)
		assert.Equal(t, profiles, prof.Dir, "and said where to keep it")

		requireSameFile(t, path, cli.ConfigPath())
		assert.ElementsMatch(t, []string{"profile", "profile-dir", "workdir"}, cli.ConfigSet(),
			"scan-models is not among them: it was typed, so the file never got to set it - what is reported is "+
				"what the session took from the file, not what the file asked for")
	})

	t.Run("should refuse a file it cannot obey", func(t *testing.T) {
		cli, _ := inConfiguredDir(t, "emit:\n  scan-modles: false\n")
		require.NoError(t, cli.set.Parse(nil))

		_, _, err := cli.PrepareScan()

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
		assert.Empty(t, cli.ConfigPath(), "and reports having read nothing, because it read nothing it could use")
	})

	t.Run("should answer about a command line it was never given", func(t *testing.T) {
		// The version and profile flags are read straight off the parsed command line, and are asked before there is
		// necessarily one - which is what the nil guards on them are for.
		var cli Config

		assert.False(t, cli.WantsVersion())
		assert.False(t, cli.WantsProfile())
		assert.Empty(t, cli.ConfigPath())
		assert.Empty(t, cli.ConfigSet())
	})
}

func TestProfiling(t *testing.T) {
	// NOT PARALLEL: the sampling rate below is the process's, not a scan's.

	t.Run("should be disabled unless it is asked for", func(t *testing.T) {
		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse(nil))

		_, prof, err := cli.PrepareScan()
		require.NoError(t, err)

		assert.Nil(t, prof)
	})

	t.Run("should keep the profiles where -profile-dir says", func(t *testing.T) {
		dir := t.TempDir()
		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", dir}))

		_, prof, err := cli.PrepareScan()
		require.NoError(t, err)
		require.NotNil(t, prof)
		assert.True(t, prof.Enabled)
		assert.Equal(t, dir, prof.Dir)
	})

	t.Run("should resolve a relative -profile-dir", func(t *testing.T) {
		// The profiles are written after a scan, and a relative directory would be read against wherever the process
		// stands when it comes to write them.
		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", "profiles"}))

		prof, err := cli.profiling()
		require.NoError(t, err)

		assert.True(t, filepath.IsAbs(prof.Dir), "got %q", prof.Dir)
		assert.Equal(t, "profiles", filepath.Base(prof.Dir))
	})

	t.Run("should make a directory when none is named", func(t *testing.T) {
		// -profile alone has to be enough: the point is to be able to ask for a profile without first deciding where
		// to keep it.
		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-profile"}))

		prof, err := cli.profiling()
		require.NoError(t, err)
		t.Cleanup(func() { _ = os.RemoveAll(prof.Dir) })

		assert.True(t, filepath.IsAbs(prof.Dir), "got %q", prof.Dir)
		assert.DirExists(t, prof.Dir, "made now rather than at the first scan, so that a run that cannot write "+
			"anywhere says so before it is measured")
	})

	t.Run("should apply the sampling rate to the process", func(t *testing.T) {
		// The heap sampling rate is a property of the process, so it is applied once, here, rather than per scan.
		saved := runtime.MemProfileRate
		t.Cleanup(func() { runtime.MemProfileRate = saved })

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", t.TempDir(), "-mem-profile-rate", "1"}))

		_, err := cli.profiling()
		require.NoError(t, err)

		assert.Equal(t, 1, runtime.MemProfileRate)
	})

	t.Run("should leave the process's own rate alone when none is named", func(t *testing.T) {
		// Zero is the flag's "nothing said", not a rate: setting it would turn -profile into an instruction to sample
		// nothing and report an empty allocation table.
		saved := runtime.MemProfileRate
		t.Cleanup(func() { runtime.MemProfileRate = saved })

		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-profile", "-profile-dir", t.TempDir()}))

		_, err := cli.profiling()
		require.NoError(t, err)

		assert.Equal(t, saved, runtime.MemProfileRate)
	})
}

func TestConfigFile(t *testing.T) {
	// NOT PARALLEL: the file is found by searching upwards from the working directory.

	t.Run("should preset the flags it names", func(t *testing.T) {
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
	})

	t.Run("should lose to anything typed", func(t *testing.T) {
		// The precedence rule: a file presets, a command line decides.
		cli, _ := inConfiguredDir(t, "emit:\n  scan-models: false\n")
		require.NoError(t, cli.set.Parse([]string{"-scan-models=true"}))

		_, _, err := configured(cli.set, cli.configFile)
		require.NoError(t, err)

		opts, err := cli.options(nil)
		require.NoError(t, err)
		assert.True(t, opts.ScanModels)
	})

	t.Run("should not be consulted at all under -no-config", func(t *testing.T) {
		cli, path := inConfiguredDir(t, "emit:\n  scan-models: false\n")
		require.NoError(t, cli.set.Parse([]string{"-no-config"}))

		applied, got, err := configured(cli.set, cli.configFile)
		require.NoError(t, err)
		require.FileExists(t, path, "the file is there; it was simply not consulted")

		assert.Empty(t, got)
		assert.Empty(t, applied.Set)
	})

	t.Run("should refuse a key nobody can act on", func(t *testing.T) {
		// Which reads exactly like a setting that quietly never applied, and is what the schema is for.
		cli, _ := inConfiguredDir(t, "emit:\n  scan-modles: false\n")
		require.NoError(t, cli.set.Parse(nil))

		_, _, err := configured(cli.set, cli.configFile)

		require.ErrorIs(t, err, cliconf.ErrUnknownKey)
	})

	t.Run("should refuse what it cannot parse", func(t *testing.T) {
		// A different failure from a file whose contents cannot be obeyed, and it says so: the keys are never
		// reached, so naming one would be guesswork.
		cli, _ := inConfiguredDir(t, "emit:\n\tscan-models: false\n") // tabs do not indent YAML
		require.NoError(t, cli.set.Parse(nil))

		_, _, err := configured(cli.set, cli.configFile)

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
		assert.Contains(t, err.Error(), cliconf.Names[0],
			"and names the file, since the search decided which one that is and the caller may not know")
	})

	t.Run("should refuse what it cannot read", func(t *testing.T) {
		// Named rather than discovered: the search skips a directory, so this is only reachable by asking for one.
		cli := newTestFlags(t)
		require.NoError(t, cli.set.Parse([]string{"-config", t.TempDir()}))

		_, _, err := configured(cli.set, cli.configFile)

		require.ErrorIs(t, err, cliconf.ErrBadConfig)
	})

	t.Run("should skip another command's section", func(t *testing.T) {
		// Skipped rather than refused - that is what makes the file shareable.
		cli, _ := inConfiguredDir(t, "document:\n  output: spec.json\nemit:\n  scan-models: false\n")
		require.NoError(t, cli.set.Parse(nil))

		applied, _, err := configured(cli.set, cli.configFile)
		require.NoError(t, err)

		assert.Equal(t, []string{"scan-models"}, applied.Set)
		assert.Equal(t, []string{"document.output"}, applied.Ignored, "genspec's half, reported rather than dropped")
	})
}

/********************************/
/* test helpers */
/********************************/

// inConfiguredDir writes a configuration file into a fresh directory and runs the test from inside it, which is how
// the file is found: the search walks up from where the command was started.
func inConfiguredDir(t *testing.T, content string) (*Config, string) {
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

func newTestFlags(t *testing.T) *Config {
	t.Helper()

	fs := flag.NewFlagSet("genspec-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return NewWithFlags(fs)
}
