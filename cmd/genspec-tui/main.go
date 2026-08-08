// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux"
)

// cliFlags holds the raw flag values.
//
// Registration is separated from parsing so TestFlags_CoverEveryValueTypedOption can inspect the flag set without
// running the program.
type cliFlags struct {
	set *flag.FlagSet

	workdir             *string
	packages            *string
	scanModels          *bool
	buildTags           *string
	goos                *string
	goarch              *string
	goflags             *string
	gowork              *string
	goexperiment        *string
	toolchainFreeLoader *bool
	stubStdlib          *bool
	include             *string
	exclude             *string
	includeTags         *string
	excludeTags         *string
	nameFromTags        *string
	nameConcatBudget    *float64
}

// registerFlags declares every flag on fs.
//
// Each value-typed field of codescan.Options belongs here; the drift guard in main_test.go fails when one is added
// without a flag, which is how the CLI came to expose three options out of ten.
func registerFlags(fs *flag.FlagSet) *cliFlags {
	return &cliFlags{
		set:        fs,
		workdir:    fs.String("workdir", ".", "module directory where scanning runs (codescan WorkDir)"),
		packages:   fs.String("packages", "./...", "comma-separated package patterns to scan, relative to -workdir"),
		scanModels: fs.Bool("scan-models", true, "also emit definitions for swagger:model types"),
		buildTags:  fs.String("build-tags", "", "comma-separated go build tags to apply while loading"),
		goos:       fs.String("goos", "", "GOOS the scanned code is built for (default: this machine's)"),
		goarch:     fs.String("goarch", "", "GOARCH the scanned code is built for (default: this machine's)"),
		goflags: fs.String("goflags", "",
			"default go command flags, as GOFLAGS (e.g. \"-tags=integration\"); -build-tags wins"),
		gowork: fs.String("gowork", "",
			`workspace selection, as GOWORK: "off" to disable, a path to a go.work, empty to search upwards`),
		goexperiment: fs.String("goexperiment", "",
			"toolchain experiments to enable, as GOEXPERIMENT (e.g. \"jsonv2\")"),
		toolchainFreeLoader: fs.Bool("toolchain-free-loader", false,
			"load packages with codescan's own loader instead of the go command (experimental)"),
		stubStdlib: fs.Bool("stub-stdlib", false,
			"synthesize the standard library instead of reading GOROOT (needs -toolchain-free-loader)"),
		include: fs.String("include", "", "comma-separated patterns; only matching packages are scanned"),
		exclude: fs.String("exclude", "", "comma-separated patterns; matching packages are skipped"),
		includeTags: fs.String("include-tags", "",
			"comma-separated swagger tags; only matching operations are emitted"),
		excludeTags: fs.String("exclude-tags", "",
			"comma-separated swagger tags; matching operations are skipped"),
		nameFromTags: fs.String("name-from-tags", "",
			`ordered struct tags a field's name derives from (default "json"; pass empty to use the Go field name)`),
		nameConcatBudget: fs.Float64("name-concat-budget", 0,
			"readability cutoff for collision-renaming by concatenation (0 = codescan's default of 0.65)"),
	}
}

// options assembles the scan config. workDir is passed in already absolute.
func (c *cliFlags) options(workDir string) codescan.Options {
	return codescan.Options{
		WorkDir:             workDir,
		Packages:            splitPatterns(*c.packages),
		ScanModels:          *c.scanModels,
		BuildTags:           *c.buildTags,
		GOOS:                *c.goos,
		GOARCH:              *c.goarch,
		GOFLAGS:             *c.goflags,
		GOWORK:              *c.gowork,
		GOEXPERIMENT:        *c.goexperiment,
		ToolchainFreeLoader: *c.toolchainFreeLoader,
		StubStdlib:          *c.stubStdlib,
		Include:             splitList(*c.include),
		Exclude:             splitList(*c.exclude),
		IncludeTags:         splitList(*c.includeTags),
		ExcludeTags:         splitList(*c.excludeTags),
		NameFromTags:        resolveNameFromTags(*c.nameFromTags, c.passed("name-from-tags")),
		NameConcatBudget:    *c.nameConcatBudget,
	}
}

// passed reports whether the named flag appeared on the command line, as opposed to holding its default.
func (c *cliFlags) passed(name string) bool {
	seen := false
	c.set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})

	return seen
}

// resolveNameFromTags maps the flag onto NameFromTags.
//
// This is a three-way contract.
//
//   - nil (unset) keeps the historic ["json"] behaviour
//   - an empty but NON-nil slice means "consult no struct tag, use the Go field name"
//
// Collapsing those two would make -name-from-tags= silently mean the opposite of what it says.
func resolveNameFromTags(raw string, passed bool) []string {
	if !passed {
		return nil
	}
	if list := splitList(raw); list != nil {
		return list
	}

	return []string{}
}

func main() {
	// Mute the scanner's logging. codescan writes warnings (unsupported type kinds, skipped builtins, ...) through the
	// standard log package, whose default sink is stderr - which paints over bubbletea's alt-screen and corrupts the TUI.
	//
	// Discard it globally for the lifetime of the program.
	//
	// Proposal for enhancement(maintainers): codescan should accept an injected sink and route these through
	// OnDiagnostic instead of the global logger. This is probably already done, so this discarding is probably not
	// useful any longer.
	log.SetOutput(io.Discard)

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genspec-tui:", err)
		os.Exit(1)
	}
}

// run is main's body, split out so that os.Exit happens with no deferred call pending.
//
// Exiting from inside main skipped defer model.Close(), leaving the file watcher running until the process died anyway.
func run() error {
	cli := registerFlags(flag.CommandLine)
	flag.Parse()

	dir, err := filepath.Abs(*cli.workdir)
	if err != nil {
		return err
	}

	model := ux.New(cli.options(dir))
	defer model.Close()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()

	return err
}

// splitList parses a comma-separated flag into trimmed, non-empty entries.
//
// It returns nil when there is nothing usable - nil being what codescan reads as "no filter".
func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

// splitPatterns parses the comma-separated -packages flag into non-empty, trimmed patterns.
//
// It falls back to "./..." when nothing usable is given.
func splitPatterns(s string) []string {
	if out := splitList(s); len(out) > 0 {
		return out
	}

	return []string{"./..."}
}
