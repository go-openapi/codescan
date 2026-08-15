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

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/config"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux"
	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/codescan/cmd/internal/cliopts"
)

const cmd = "genspec-tui"

func main() {
	// Mute the scanner's logging. codescan no longer writes warnings (unsupported type kinds, skipped builtins, ...)
	// through the standard log package, whose default sink is stderr.
	// However, dependent libraries (e.g. spec, validate) still may do, which paints over bubbletea's alt-screen and
	// garbles the TUI.
	//
	// Discard it globally for the lifetime of the program.
	log.SetOutput(io.Discard)

	err := run(os.Args[1:], os.Stderr)
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, cmd+":", err)
	}

	if err != nil {
		// NOTE: unlike genspec, we don't pay much attention to the command exit code in a TUI context.
		os.Exit(1)
	}
}

// run is main's body, with the process left outside.
//
// Everything it touches arrives as an argument, so a test drives the real entry point rather than a rehearsal of it -
// which is what checks the flag surface and the configuration file against what the session actually starts with.
//
// NOTE: [os.Exit] happens in main because exiting from in here would skip the deferred Close, leaving the file watcher
// running until the process dies anyway.
func run(argv []string, stderr io.Writer) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = usage(fs, stderr)

	// 0. Parse CLI args & handles short-circuits (help, version)
	cfg := config.NewWithFlags(fs)
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// -h has already printed the usage, and asking for it is not an error: nothing more to say, and
			// nothing to say it about.
			return nil
		}

		return err
	}

	if cfg.WantsVersion() {
		fmt.Fprintln(os.Stdout, cliopts.Version(cmd))

		return nil
	}

	// 1. Prepare config & scan options
	opts, prof, err := cfg.PrepareScan()
	if err != nil {
		return err
	}

	// 2. Builds the UX model, with scanning and profiling options at startup.
	// NOTE: scanning options may be modified by the user after launch, profiling cannot.
	model := ux.New(ux.Startup{
		Options:   opts, // scan options
		Profiling: prof, // profiling options

		// for reporting only
		ConfigPath: cfg.ConfigPath(),
		ConfigSet:  cfg.ConfigSet(),
	})
	defer model.Close()

	// 3. Run the UX
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()

	return err
}

// usage is what -h prints.
//
// Written out rather than left to the flag package's default, because we want to document positional args.
func usage(fs *flag.FlagSet, stderr io.Writer) func() {
	return func() {
		fmt.Fprintf(stderr, "usage: %s [flags] [packages...]\n", cmd)
		fmt.Fprintln(stderr, "\nBrowses annotated Go source beside the Swagger 2.0 specification it produces,")
		fmt.Fprintln(stderr, "regenerated on every save.")
		fmt.Fprintln(stderr, "\nThe packages are Go patterns, resolved against -workdir. Naming none scans "+
			cliopts.DefaultPatterns)
		fmt.Fprintln(stderr, "\nEvery boolean below can also be toggled during the session with o, which rescans on")
		fmt.Fprintln(stderr, "close - so these decide what the FIRST scan does, not what the session is stuck with.")
		fmt.Fprintln(stderr, "\nFlags may be preset in a "+cliconf.SampleConfigName()+", found by searching upwards from here.")
		fmt.Fprintln(stderr, "Anything typed on the command line wins over it. Keys are grouped into sections and")
		fmt.Fprintln(stderr, "named after the flags they set:")
		fmt.Fprintln(stderr, "\n  scan:\n    workdir: ./api\n  emit:\n    scan-models: true\n  profile:\n    mem-profile-rate: 1")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}
}
