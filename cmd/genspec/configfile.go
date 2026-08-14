// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"

	"github.com/go-openapi/codescan/internal/cliconf"
	"github.com/go-openapi/codescan/internal/cliopts"
)

// The sections this command's own flags are addressed in.
//
// The library's flags bring their own (scan, go, load, emit); these are what genspec adds on top:
// the document it reads and writes, and how loud it is about what it saw.
const (
	sectionDocument    = "document"
	sectionDiagnostics = "diagnostics"
)

// commandSections says where each of this command's flags is addressed.
//
// A literal rather than something derived, because these flags are registered inline: there is no
// table to read a section off. TestEveryFlagIsAddressableInAConfigFile is what keeps it in step.
var commandSections = cliconf.Schema{ //nolint:gochecknoglobals // the schema, read once at startup
	"input":   sectionDocument,
	"output":  sectionDocument,
	"format":  sectionDocument,
	"compact": sectionDocument,

	"color":    sectionDiagnostics,
	"quiet":    sectionDiagnostics,
	"verbose":  sectionDiagnostics,
	"validate": sectionDiagnostics,
	"fail-on":  sectionDiagnostics,
}

// notConfigurable are the flags a file deliberately cannot set, with the reason.
var notConfigurable = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	cliconf.Flag:      "names the file itself, which cannot be read from inside it",
	cliconf.ShortFlag: "the short form of -" + cliconf.Flag,
	cliconf.NoFlag:    "decides whether there is a file at all, from outside it",
	"version":         "asks the command a question rather than configuring it",
}

// configSchema is the whole of what a configuration file may address here.
func configSchema() (cliconf.Schema, error) {
	schema, err := cliopts.ConfigSchema().Merge(commandSections)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUsage, err)
	}

	return schema, nil
}

// configured reads a configuration file, if there is one, and presets the flags with it.
//
// Called after parsing and before anything reads a flag, so that everything downstream sees one
// settled command line and needs to know nothing about where a value came from.
func configured(fs *flag.FlagSet, which *cliconf.Flags) (cliconf.Result, string, error) {
	var applied cliconf.Result

	// Searched from where the command was run, not from -workdir: the file is free to set -workdir,
	// and a file found through the very value it sets would be reasoning in a circle.
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

// read loads a configuration file through koanf.
//
// koanf for what it will be asked for next - environment variables, a second format, several sources
// merged - rather than for reading one YAML file, which cliconf can do by itself. What it hands back
// is the same flat, section-qualified map [cliconf.Parse] produces, so the two are interchangeable
// and the commands that cannot take the dependency lose nothing.
//
// The bytes are read here rather than by koanf's file provider, which brings a filesystem watcher
// with it for a file this command reads once and never looks at again.
func read(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", cliconf.ErrBadConfig, err)
	}

	loaded := koanf.New(cliconf.Delimiter)
	if err := loaded.Load(rawbytes.Provider(data), cliconf.YAML{}); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", cliconf.ErrBadConfig, path, err)
	}

	return loaded.All(), nil
}
