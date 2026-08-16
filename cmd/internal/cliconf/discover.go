// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// The flags every command registers for its configuration file.
//
// Three spellings of two questions: which file, and whether to read one at all. -c is the short form
// of -config, because naming a file is the thing done often enough to be worth two keystrokes;
// -no-config is the way to say none, and it is a switch rather than a magic value so that it reads
// as what it is at a glance.
const (
	// Flag names the file to read.
	Flag = "config"
	// ShortFlag is the short form of [Flag].
	ShortFlag = "c"
	// NoFlag says to read no configuration file at all.
	NoFlag = "no-config"
)

// SampleConfigName returns the first default supported default config file.
func SampleConfigName() string {
	return Names[0]
}

// Names are the file names looked for, in order, when -config says nothing.
//
// One name for every command rather than one per command: the sections are what tell them apart, and
// a project configuring a scan has configured it for all of them. JSON is in the list because the
// parser reads it anyway, and a generated file is as likely to be JSON as YAML.
var Names = []string{ //nolint:gochecknoglobals // the search list, read once at startup
	".codescan.yaml",
	".codescan.yml",
	".codescan.json",
}

// Flags is where the configuration-file flags land.
//
// A type rather than a pair of pointers, because the two questions can be answered in ways that
// contradict each other, and something has to hold the answer to "then what".
type Flags struct {
	// long and short hold the same answer, separately, so that giving both can be noticed rather
	// than resolved by whichever the parser happened to read last.
	long  *string
	short *string

	none *bool
}

// Register declares the configuration-file flags and returns where their values land.
func Register(fs *flag.FlagSet) *Flags {
	return &Flags{
		long: fs.String(Flag, "",
			"configuration file to read before the flags (default: the nearest .codescan.yaml,\n"+
				"searching upwards)"),
		short: fs.String(ShortFlag, "", "shorthand for -"+Flag),
		none:  fs.Bool(NoFlag, false, "read no configuration file, whatever is lying around"),
	}
}

// Discover reports which file to read, if any.
//
// A named file must exist: a caller who named one meant that file, and silently searching for
// another - or for none - would answer a question they did not ask. Searching, on the other hand, is
// allowed to find nothing, which is the ordinary case.
//
// The search walks up from start, so running a command from anywhere inside a project finds the
// project's own file. It stops at the first hit rather than merging what it passes: a file
// half-overridden by one three directories up is not something anybody can read off the page.
func (f *Flags) Discover(start string) (string, error) {
	named, err := f.named()
	if err != nil {
		return "", err
	}

	if *f.none {
		if named != "" {
			// Named by either spelling, so the message names both rather than guessing which was typed.
			return "", fmt.Errorf("%w: -%s/-%s names %q, and -%s says to read none",
				ErrBadConfig, Flag, ShortFlag, named, NoFlag)
		}

		return "", nil
	}

	if named == "" {
		return search(start)
	}

	if _, err := os.Stat(named); err != nil {
		return "", fmt.Errorf("%w: -%s %q: %w", ErrBadConfig, Flag, named, err)
	}

	return named, nil
}

// named reports the file the caller asked for, by either spelling.
func (f *Flags) named() (string, error) {
	long, short := *f.long, *f.short

	if long != "" && short != "" && long != short {
		return "", fmt.Errorf("%w: -%s says %q and -%s says %q", ErrBadConfig, Flag, long, ShortFlag, short)
	}
	if long != "" {
		return long, nil
	}

	return short, nil
}

// search walks up from dir looking for a file named like a configuration.
func search(dir string) (string, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("%w: cannot resolve %q: %w", ErrBadConfig, dir, err)
	}

	for {
		for _, name := range Names {
			candidate := filepath.Join(current, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}
