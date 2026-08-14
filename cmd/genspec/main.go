// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/cliopts"
)

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "genspec:", err)
	}

	if status := exitStatus(err); status != exitOK {
		os.Exit(status)
	}
}

// run is the whole command, with the process left outside.
//
// Everything it touches arrives as an argument, so a test drives the real entry point rather than a
// rehearsal of it - which is the only way the flag surface, the exit statuses and what lands on
// which stream are checked at all.
func run(argv []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("genspec", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = usage(fs, stderr)

	cfg := registerFlags(fs)
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}

		return fmt.Errorf("%w: %w", errUsage, err)
	}

	if *cfg.version {
		fmt.Fprintln(stdout, version())

		return nil
	}

	// Before anything reads a flag, so that everything downstream sees one settled command line and
	// needs to know nothing about where a value came from.
	applied, configPath, err := configured(fs, *cfg.configFile)
	if err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}

	format, err := resolveFormat(*cfg.format, *cfg.output)
	if err != nil {
		return err
	}

	report, err := newReporter(cfg, stderr)
	if err != nil {
		return err
	}
	// Said now rather than when it was read: the file is free to decide how loud this command is, so
	// there was nothing to say it through until here.
	report.configuration(configPath, applied)

	opts, err := cfg.options(fs.Args(), report)
	if err != nil {
		return err
	}

	doc, err := codescan.Run(opts)
	if err != nil {
		return err
	}

	rendered, asJSON, err := render(doc, format, *cfg.compact)
	if err != nil {
		return err
	}

	if err := write(rendered, *cfg.output, stdout); err != nil {
		return err
	}

	// Validation comes after the document is written, so that an invalid one is still there to be
	// read. Being told what is wrong with a specification you cannot look at is not much of a report.
	var invalid error
	if *cfg.validate {
		invalid = checkValid(asJSON, report)
	}

	if tripped := report.summarize(); invalid == nil {
		return tripped
	}

	return invalid
}

// options assembles what the scan runs with.
func (c *config) options(patterns []string, report *reporter) (*codescan.Options, error) {
	opts := &codescan.Options{Packages: cliopts.Patterns(patterns)}
	if err := c.scan.Apply(opts); err != nil {
		return nil, err
	}

	input, err := loadInput(*c.input)
	if err != nil {
		return nil, err
	}
	opts.InputSpec = input

	// Positions are reported relative to where the scan ran, which has to be settled before anything
	// is reported - and resolved, since -workdir is free to be relative and a diagnostic is not.
	root, err := filepath.Abs(opts.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve -workdir %q: %w", opts.WorkDir, err)
	}
	report.root = root

	opts.OnDiagnostic = report.onDiagnostic

	return opts, nil
}

// version reports what this build is, as the module system recorded it.
//
// A binary installed with `go install ...@latest` carries its version; one built from a working copy
// carries the revision instead, and says so rather than claiming a release it is not.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "genspec (unknown version)"
	}

	if v := info.Main.Version; v != "" && v != "(devel)" {
		return "genspec " + v
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return "genspec (devel, " + setting.Value + ")"
		}
	}

	return "genspec (devel)"
}
