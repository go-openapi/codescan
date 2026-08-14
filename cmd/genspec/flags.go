// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/cliopts"
)

// What -format can be told. "auto" reads the extension of -output, and writes JSON when that says
// nothing - which is what a document going to standard output does.
const (
	formatAuto = "auto"
	formatJSON = "json"
	formatYAML = "yaml"
)

// What -color can be told. "auto" is the only one that looks at where it is writing.
const (
	colorAuto   = "auto"
	colorAlways = "always"
	colorNever  = "never"
)

// What -fail-on can be told: the least serious diagnostic that still makes the command exit
// non-zero.
//
// "never" is the default because a scan that emits warnings is the ordinary case - a codebase with
// nothing to say about it is the exception - and a command that failed the build over one would
// mostly teach people to stop reading it.
const (
	failNever   = "never"
	failError   = "error"
	failWarning = "warning"
)

// config is the whole command line: the library's own knobs, plus this command's.
type config struct {
	set *flag.FlagSet

	// scan is every knob the library takes, declared in internal/cliopts and shared with the other
	// commands, so that a flag means the same thing whichever one you reach for.
	scan *cliopts.Values

	input    *string
	output   *string
	format   *string
	compact  *bool
	validate *bool

	color   *string
	quiet   *bool
	verbose *bool
	failOn  *string

	version *bool
}

func registerFlags(fs *flag.FlagSet) *config {
	return &config{
		set:  fs,
		scan: cliopts.Register(fs),

		input: fs.String("input", "",
			"a specification to merge the scan's discoveries into, as JSON or YAML"),
		output: fs.String("output", "-", `write the specification here ("-" for standard output)`),
		format: fs.String("format", formatAuto,
			`what to write: "json", "yaml", or "auto" to read the extension of -output`),
		compact: fs.Bool("compact", false, "write the document without indentation"),
		validate: fs.Bool("validate", false,
			"check the document against the Swagger 2.0 schema and report what is wrong with it"),

		color: fs.String("color", colorAuto,
			`colored diagnostics: "always", "never", or "auto" for whenever stderr is a terminal`),
		quiet:   fs.Bool("quiet", false, "say nothing on standard error"),
		verbose: fs.Bool("verbose", false, "also report hints, which are muted by default"),
		failOn: fs.String("fail-on", failNever,
			`exit non-zero when anything reported - by the scan or by -validate - reaches this`+"\n"+
				`severity: "error", "warning", or "never"`),

		version: fs.Bool("version", false, "print the version and exit"),
	}
}

// usage is what -h prints.
//
// Written out rather than left to the flag package's default, because the first thing a reader needs
// is that the packages are positional and that the document goes to standard output - neither of
// which is a flag, so neither would appear.
func usage(fs *flag.FlagSet, stderr io.Writer) func() {
	return func() {
		fmt.Fprintln(stderr, "usage: genspec [flags] [packages...]")
		fmt.Fprintln(stderr, "\nScans annotated Go source and writes the Swagger 2.0 specification it describes.")
		fmt.Fprintln(stderr, "\nThe packages are Go patterns, resolved against -workdir. Naming none scans "+
			cliopts.DefaultPatterns)
		fmt.Fprintln(stderr, "The document goes to standard output unless -output names a file; what the scan")
		fmt.Fprintln(stderr, "observed goes to standard error.")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
}

// resolveFormat decides what to write, from what was asked and where it is going.
//
// The extension is consulted only where nothing was said: a caller who writes -format=json to a file
// called spec.yaml has said something contradictory on purpose, and the flag is the one that says it
// out loud.
func resolveFormat(format, output string) (string, error) {
	switch format {
	case formatJSON, formatYAML:
		return format, nil
	case formatAuto:
		if isYAMLName(output) {
			return formatYAML, nil
		}

		return formatJSON, nil
	default:
		return "", fmt.Errorf("%w: -format %q is not one of %s, %s, %s",
			errUsage, format, formatJSON, formatYAML, formatAuto)
	}
}

// isYAMLName reports whether a path names a YAML file.
func isYAMLName(path string) bool {
	lower := strings.ToLower(path)

	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// resolveFailOn reads the threshold as the severity it names.
//
// Reported as a [codescan.Severity] and a flag saying whether there is a threshold at all, rather
// than as a severity meaning "none": the enum belongs to the scanner, and inventing a member of it
// here would be inventing a diagnostic that cannot happen.
func resolveFailOn(failOn string) (codescan.Severity, bool, error) {
	switch failOn {
	case failNever:
		return 0, false, nil
	case failError:
		return codescan.SeverityError, true, nil
	case failWarning:
		return codescan.SeverityWarning, true, nil
	default:
		return 0, false, fmt.Errorf("%w: -fail-on %q is not one of %s, %s, %s",
			errUsage, failOn, failError, failWarning, failNever)
	}
}
