// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Flag is what the configuration file is named on the command line.
const Flag = "config"

// Off is what -config is given to use no file at all, whatever is lying around.
//
// Spelled as a value rather than as a second flag, the way GOWORK spells it: a run that must be
// reproducible says so in one place, and there is no pair of flags that can disagree.
const Off = "off"

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

const flagHelp = `configuration file to read before the flags ("off" for none; default: the nearest ` +
	`.codescan.yaml, searching upwards)`

// Register declares the -config flag and returns where its value lands.
func Register(fs *flag.FlagSet) *string {
	return fs.String(Flag, "", flagHelp)
}

// Discover reports which file to read, if any.
//
// A named file must exist: a caller who said -config meant that file, and silently scanning for
// another one - or for none - would answer a question they did not ask. Searching, on the other
// hand, is allowed to find nothing, which is the ordinary case.
//
// The search walks up from start, so running a command from anywhere inside a project finds the
// project's own file. It stops at the first hit rather than merging what it passes: a file that is
// half-overridden by one three directories up is not something anybody can read off the page.
func Discover(explicit, start string) (string, error) {
	switch explicit {
	case Off:
		return "", nil
	case "":
		return search(start)
	}

	if _, err := os.Stat(explicit); err != nil {
		return "", fmt.Errorf("%w: -%s %q: %w", ErrBadConfig, Flag, explicit, err)
	}

	return explicit, nil
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
