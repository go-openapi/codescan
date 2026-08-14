// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/internal/cliconf"
	"github.com/go-openapi/codescan/internal/cliopts"
)

// config is the whole command line: the library's knobs, plus this command's.
type config struct {
	set *flag.FlagSet

	// scan is every knob the library takes, declared in internal/cliopts and shared with the other commands, so that a
	// flag means the same thing whichever one you reach for. It used to be a table of its own here, which is how the
	// TUI came to expose fourteen options out of thirty.
	scan *cliopts.Values

	// configFile is which file is read before the flags, if any. Its own field rather than one of the above: it decides
	// where the others come from, so it cannot itself come from there.
	configFile *cliconf.Flags

	// packages is the historic spelling of the positional patterns.
	//
	// Kept working, no longer advertised: `genspec-tui ./api/...` is how every other Go command reads and how genspec
	// reads, but a flag that has been in the README since the first release is in somebody's editor task by now.
	packages *string

	// Observation of the run rather than configuration of it, which is why these are not codescan options.
	profile        *bool
	profileDir     *string
	memProfileRate *int
}

// registerFlags declares every flag on fs.
func registerFlags(fs *flag.FlagSet) *config {
	return &config{
		set:        fs,
		scan:       cliopts.Register(fs),
		configFile: cliconf.Register(fs),

		packages: fs.String("packages", "",
			"comma-separated package patterns (deprecated: name them as arguments instead)"),

		profile: fs.Bool("profile", false,
			"profile every scan (CPU + allocations), reported under m and written for go tool pprof"),
		profileDir: fs.String("profile-dir", "",
			"where -profile writes its .pprof files (default: a fresh temp directory)"),
		memProfileRate: fs.Int("mem-profile-rate", 0,
			"with -profile, runtime.MemProfileRate: 0 samples every 512 KiB, 1 records every allocation exactly"),
	}
}

// usage is what -h prints.
//
// Written out rather than left to the flag package's default, because the first thing a reader needs is that the
// packages are positional and that the options are also reachable from inside the session - neither of which is a flag,
// so neither would appear.
func usage(fs *flag.FlagSet, stderr io.Writer) func() {
	return func() {
		fmt.Fprintln(stderr, "usage: genspec-tui [flags] [packages...]")
		fmt.Fprintln(stderr, "\nBrowses annotated Go source beside the Swagger 2.0 specification it produces,")
		fmt.Fprintln(stderr, "regenerated on every save.")
		fmt.Fprintln(stderr, "\nThe packages are Go patterns, resolved against -workdir. Naming none scans "+
			cliopts.DefaultPatterns)
		fmt.Fprintln(stderr, "\nEvery boolean below can also be toggled during the session with o, which rescans on")
		fmt.Fprintln(stderr, "close - so these decide what the FIRST scan does, not what the session is stuck with.")
		fmt.Fprintln(stderr, "\nFlags may be preset in a "+cliconf.Names[0]+", found by searching upwards from here.")
		fmt.Fprintln(stderr, "Anything typed on the command line wins over it. Keys are grouped into sections and")
		fmt.Fprintln(stderr, "named after the flags they set:")
		fmt.Fprintln(stderr, "\n  scan:\n    workdir: ./api\n  emit:\n    scan-models: true\n  profile:\n    mem-profile-rate: 1")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
}

// options assembles the scan configuration the session starts with.
//
// patterns are the positional arguments; -packages stands in for them when there are none.
func (c *config) options(patterns []string) (codescan.Options, error) {
	var opts codescan.Options

	opts.Packages = c.patterns(patterns)
	if err := c.scan.Apply(&opts); err != nil {
		return opts, err
	}

	// The source tree, the file watcher and every diagnostic position are rooted here, and -workdir is free to be
	// relative - so it is resolved once, before anything is built on it.
	root, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return opts, fmt.Errorf("cannot resolve -workdir %q: %w", opts.WorkDir, err)
	}
	opts.WorkDir = root

	return opts, nil
}

// patterns reports what to scan: the arguments, the deprecated flag, or everything under the working directory.
func (c *config) patterns(args []string) []string {
	if len(args) == 0 {
		if list := cliopts.SplitList(*c.packages); len(list) > 0 {
			return list
		}
	}

	return cliopts.Patterns(args)
}

// profiling reads the profile flags, and applies the ones the runtime wants set before anything is measured.
//
// The heap sampling rate is a property of the process, not of a scan: the runtime expects it constant for the
// program's lifetime, and the profiles record the rate they were taken under. Setting it here, once, is what makes two
// runs in the same session comparable - and is why this is a flag rather than a row in the options overlay.
func (c *config) profiling() (scan.Profiling, error) {
	if !*c.profile {
		return scan.Profiling{}, nil
	}

	if rate := *c.memProfileRate; rate > 0 {
		runtime.MemProfileRate = rate
	}

	dir := *c.profileDir
	if dir == "" {
		tmp, err := os.MkdirTemp("", "genspec-tui-profile-")
		if err != nil {
			return scan.Profiling{}, fmt.Errorf("cannot create a profile directory: %w", err)
		}
		dir = tmp
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return scan.Profiling{}, err
	}

	return scan.Profiling{Enabled: true, Dir: abs}, nil
}

func main() {
	// Mute the scanner's logging. codescan writes warnings (unsupported type kinds, skipped builtins, ...) through the
	// standard log package, whose default sink is stderr - which paints over bubbletea's alt-screen and corrupts the TUI.
	//
	// Discard it globally for the lifetime of the program.
	log.SetOutput(io.Discard)

	err := run(os.Args[1:], os.Stderr)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "genspec-tui:", err)
	}
	if err != nil {
		os.Exit(1)
	}
}

// run is main's body, with the process left outside.
//
// Everything it touches arrives as an argument, so a test drives the real entry point rather than a rehearsal of it -
// which is what checks the flag surface and the configuration file against what the session actually starts with. It is
// also why os.Exit happens in main: exiting from in here would skip the deferred Close, leaving the file watcher
// running until the process died anyway.
func run(argv []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("genspec-tui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = usage(fs, stderr)

	cli := registerFlags(fs)
	if err := fs.Parse(argv); err != nil {
		return err
	}

	// Before anything reads a flag, so that everything downstream sees one settled command line and needs to know
	// nothing about where a value came from.
	applied, configPath, err := configured(fs, cli.configFile)
	if err != nil {
		return err
	}

	opts, err := cli.options(fs.Args())
	if err != nil {
		return err
	}

	prof, err := cli.profiling()
	if err != nil {
		return err
	}

	model := ux.New(ux.Startup{
		Options:    opts,
		Profiling:  prof,
		ConfigPath: configPath,
		ConfigSet:  applied.Set,
	})
	defer model.Close()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()

	return err
}
