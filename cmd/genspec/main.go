// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec/internal/config"
	"github.com/go-openapi/codescan/cmd/genspec/internal/render"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
)

// Exit statuses. See the package documentation.
const (
	exitOK          = 0
	exitFailed      = 1
	exitUsage       = 2
	exitDiagnostics = 3
	exitInvalidSpec = 4
)

const cmd = "genspec"

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, cmd+":", err)
	}

	if status := exitStatus(err); status != exitOK {
		os.Exit(status)
	}
}

// run the command, with the process left outside.
func run(argv []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = usage(fs, stderr)

	// 0. Parse CLI args & handles short-circuits (help, version)
	cfg := config.NewWithFlags(fs, stdout, stderr)
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// -h has already printed the usage, and asking for it is not an error: carried out only so that
			// exitStatus can answer 0 for it.
			return err
		}

		return fmt.Errorf("%w: %w", sentinel.ErrUsage, err)
	}

	if cfg.WantsVersion() {
		fmt.Fprintln(stdout, cliopts.Version(cmd))

		return nil
	}

	// 1. Prepare config & scan options
	opts, report, err := cfg.PrepareScan()
	if err != nil {
		return err
	}

	// 2. Code scanning stage
	doc, err := codescan.Run(opts)
	if err != nil {
		return err
	}

	// 3. Renders the spec to a file or stdout
	asJSON, err := render.Spec(doc, render.Config{
		CompactJSON: cfg.WantsCompactJSON(), // ignored if wants YAML
		WantsYAML:   cfg.Format() == config.FormatYAML,
		Output:      cfg.Output(), // file or stdout if "-"
		Stdout:      stdout,
	})
	if err != nil {
		return err
	}

	// 4. Spec validation stage
	// Validation comes after the document is written, so that an invalid one is still there to be read.
	//
	/// It is better to be told about what is wrong with a specification you can look at.
	var invalid error
	if cfg.WantsValidation() {
		invalid = report.ValidateSpec(asJSON)
	}

	if tripped := report.Summarize(); invalid == nil {
		return tripped
	}

	return invalid
}

// usage is what -h prints.
//
// Written out rather than left to the flag package, because we need to document positional args, not just flags.
func usage(fs *flag.FlagSet, stderr io.Writer) func() {
	return func() {
		fmt.Fprintf(stderr, "usage: %s [flags] [packages...]\n", cmd)
		fmt.Fprintln(stderr, "\nScans annotated Go source and writes the Swagger 2.0 specification it describes.")
		fmt.Fprintln(stderr, "\nThe packages are Go patterns, resolved against -workdir. Naming none scans "+
			cliopts.DefaultPatterns)
		fmt.Fprintln(stderr, "The document goes to standard output unless -output names a file; what the scan")
		fmt.Fprintln(stderr, "observed goes to standard error.")
		fmt.Fprintln(stderr, "\nFlags may be preset in a "+cliconf.SampleConfigName()+", found by searching upwards from here.")
		fmt.Fprintln(stderr, "Anything typed on the command line wins over it. Keys are grouped into sections and")
		fmt.Fprintln(stderr, "named after the flags they set:")
		fmt.Fprintln(stderr, "\n  scan:\n    workdir: ./api\n  emit:\n    scan-models: true")
		fmt.Fprintln(stderr, "\nFlags:")

		fs.PrintDefaults()
	}
}

// exitStatus maps what run reported onto what the shell sees.
//
// Anything unrecognised is a failed scan: the specific statuses are promises about states that the
// command distinguishes, and a new kind of error is not one of them until it is added here.
func exitStatus(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.Is(err, sentinel.ErrUsage), errors.Is(err, cliopts.ErrBadFlag):
		return exitUsage
	case errors.Is(err, sentinel.ErrDiagnostics):
		return exitDiagnostics
	case errors.Is(err, sentinel.ErrInvalidSpec):
		return exitInvalidSpec
	default:
		return exitFailed
	}
}
