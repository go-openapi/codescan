// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
)

// sectionProfile is where this command's own flags are addressed.
//
// The library's flags bring their own sections (scan, go, load, emit).
// What the TUI adds specifically is about observing a run: it reads as profile.* rather than tui.*.
const sectionProfile = "profile"

// commandSections says where each of this command's flags is addressed.
//
//nolint:gochecknoglobals // the schema, read once at startup
var commandSections = cliconf.Schema{
	// NOTE: TestEveryFlagIsAddressableInAConfigFile is what keeps this list in step.
	"profile":          sectionProfile,
	"profile-dir":      sectionProfile,
	"mem-profile-rate": sectionProfile,
}

// notConfigurable are the flags a file deliberately cannot set, with the reason.
//
//nolint:gochecknoglobals // table for the drift guard
var notConfigurable = map[string]string{
	cliconf.Flag:      "names the file itself, which cannot be read from inside it",
	cliconf.ShortFlag: "the short form of -" + cliconf.Flag,
	cliconf.NoFlag:    "decides whether there is a file at all, from outside it",
	"packages": "the historic spelling of the positional patterns, kept working for scripts; a file " +
		"that could set it would be reviving the flag rather than retiring it",
	"version": "asks the command a question rather than configuring it",
}

// configSchema is the whole of what a configuration file may address here.
func configSchema() (cliconf.Schema, error) {
	schema, err := cliopts.ConfigSchema().Merge(commandSections)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

// configured reads a configuration file, if there is one, and presets the flags with it.
//
// Called after parsing and before anything reads a flag, so that everything downstream sees one settled command line
// and needs to know nothing about where a value came from.
//
// What the file decides is what the session STARTS with.
// The options overlay owns everything after that, which is why nothing here is consulted a second time.
func configured(fs *flag.FlagSet, which *cliconf.Flags) (cliconf.Result, string, error) {
	var applied cliconf.Result

	// Searched from where the command was run, not from -workdir: the file is free to set -workdir, and a file found
	// through the very value it sets would be reasoning in a circle.
	here, err := os.Getwd()
	if err != nil {
		return applied, "", fmt.Errorf("cannot tell where this is running from: %w", err)
	}

	path, err := which.Discover(here)
	if err != nil || path == "" {
		return applied, "", err
	}

	values, err := read(path)
	if err != nil {
		return applied, path, err
	}

	schema, err := configSchema()
	if err != nil {
		return applied, path, err
	}

	applied, err = cliconf.Apply(fs, values, schema)
	if err != nil {
		return applied, path, fmt.Errorf("%s: %w", path, err)
	}

	return applied, path, nil
}

// read loads a configuration file.
//
// Through cliconf's own parser rather than koanf, which cmd/genspec uses:
// koanf earns its place there for what it will be asked for next: environment variables and several sources merged.
//
// This command reads one file, once, to decide what the first scan of a session starts with.
// Both produce the same flat, section-qualified map, so the choice costs nothing but a dependency.
func read(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", cliconf.ErrBadConfig, err)
	}

	values, err := cliconf.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", cliconf.ErrBadConfig, path, err)
	}

	return values, nil
}
