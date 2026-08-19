// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/scan"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/swag/conv"
)

// Config is the whole command line: the library's knobs, plus this command's.
type Config struct {
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
	packages   *string
	version    *bool
	configPath string
	configSet  []string

	// Observation of the run rather than configuration of it, which is why these are not codescan options.
	profile        *bool
	profileDir     *string
	memProfileRate *int
}

// NewWithFlags declares a [Config] with every flag on fs.
func NewWithFlags(fs *flag.FlagSet) *Config {
	return &Config{
		set:        fs,
		scan:       cliopts.Register(fs),
		configFile: cliconf.Register(fs),

		packages: fs.String("packages", "",
			"comma-separated package patterns (deprecated: name them as arguments instead)"),

		version: fs.Bool("version", false, "print the version and exit"),
		// settings for profiling that are specific to the TUI
		profile: fs.Bool("profile", false,
			"profile every scan (CPU + allocations), reported under m and written for go tool pprof"),
		profileDir: fs.String("profile-dir", "",
			"where -profile writes its .pprof files (default: a fresh temp directory)"),
		memProfileRate: fs.Int("mem-profile-rate", 0,
			"with -profile, runtime.MemProfileRate: 0 samples every 512 KiB, 1 records every allocation exactly"),
	}
}

func (c *Config) WantsVersion() bool {
	return c.version != nil && *c.version
}

func (c *Config) WantsProfile() bool {
	return c.profile != nil && *c.profile
}

func (c *Config) ConfigPath() string {
	return c.configPath
}

func (c *Config) ConfigSet() []string {
	return c.configSet
}

// PrepareScan converts a CLI [Config] into proper [scanner.Options] that the scanner understands.
// At this point, no downstream consumer needs to know where a value came from (file, env, flag...).
func (c *Config) PrepareScan() (options *scanner.Options, profiling *scan.Profiling, err error) {
	// 1. Load the configuration file if available: this presets the flags before parsing the command line.
	applied, configPath, err := configured(c.set, c.configFile)
	if err != nil {
		return nil, nil, err
	}

	c.configPath = configPath
	c.configSet = applied.Set

	// 2. Converts all options: values are checked by cliopts.
	options, err = c.options(c.set.Args())
	if err != nil {
		return nil, nil, err
	}

	// 3. hydrate profiling settings, if desired
	if c.WantsProfile() {
		profiling, err = c.profiling()
		if err != nil {
			return nil, nil, err
		}
	}

	return options, profiling, nil
}

// options assembles the scan configuration the session starts with.
//
// patterns are the positional arguments; -packages stands in for them when there are none.
func (c *Config) options(patterns []string) (*codescan.Options, error) {
	opts := new(codescan.Options)

	opts.Packages = c.patterns(patterns)
	if err := c.scan.Apply(opts); err != nil {
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
func (c *Config) patterns(args []string) []string {
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
// program's lifetime, and the profiles record the rate they were taken under.
//
// Setting it here, once, keeps two runs in the same session comparable.
// That is why this is a CLI flag rather than a row in the options overlay.
func (c *Config) profiling() (*scan.Profiling, error) {
	if rate := conv.Value(c.memProfileRate); rate > 0 {
		runtime.MemProfileRate = rate
	}

	dir := conv.Value(c.profileDir)
	if dir == "" {
		tmp, err := os.MkdirTemp("", "genspec-tui-profile-")
		if err != nil {
			return nil, fmt.Errorf("cannot create a profile directory: %w", err)
		}
		dir = tmp
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	return &scan.Profiling{Enabled: true, Dir: abs}, nil
}
