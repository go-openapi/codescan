// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genspec-tui is an interactive terminal front-end for the codescan
// Swagger-spec generator: a source-tree browser (left), the generated spec
// (right, JSON/YAML), and diagnostics (bottom). It regenerates the whole-scope
// spec on any file change.
//
// The scan is configured from two places. Boolean knobs are toggled live in the
// options overlay (`o`), which re-runs the scan on close; the value-typed ones
// — build tags, package and tag filters, naming — are command-line flags, since
// a checkbox list cannot express them.
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

// cliFlags holds the raw flag values. Registration is separated from parsing so
// TestFlags_CoverEveryValueTypedOption can inspect the flag set without running
// the program.
type cliFlags struct {
	set *flag.FlagSet

	workdir          *string
	packages         *string
	scanModels       *bool
	buildTags        *string
	include          *string
	exclude          *string
	includeTags      *string
	excludeTags      *string
	nameFromTags     *string
	nameConcatBudget *float64
}

// registerFlags declares every flag on fs.
//
// Each value-typed field of codescan.Options belongs here; the drift guard in
// main_test.go fails when one is added without a flag, which is how the CLI
// came to expose three options out of ten.
func registerFlags(fs *flag.FlagSet) *cliFlags {
	return &cliFlags{
		set:        fs,
		workdir:    fs.String("workdir", ".", "module directory where scanning runs (codescan WorkDir)"),
		packages:   fs.String("packages", "./...", "comma-separated package patterns to scan, relative to -workdir"),
		scanModels: fs.Bool("scan-models", true, "also emit definitions for swagger:model types"),
		buildTags:  fs.String("build-tags", "", "comma-separated go build tags to apply while loading"),
		include:    fs.String("include", "", "comma-separated patterns; only matching packages are scanned"),
		exclude:    fs.String("exclude", "", "comma-separated patterns; matching packages are skipped"),
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
		WorkDir:          workDir,
		Packages:         splitPatterns(*c.packages),
		ScanModels:       *c.scanModels,
		BuildTags:        *c.buildTags,
		Include:          splitList(*c.include),
		Exclude:          splitList(*c.exclude),
		IncludeTags:      splitList(*c.includeTags),
		ExcludeTags:      splitList(*c.excludeTags),
		NameFromTags:     resolveNameFromTags(*c.nameFromTags, c.passed("name-from-tags")),
		NameConcatBudget: *c.nameConcatBudget,
	}
}

// passed reports whether the named flag appeared on the command line, as
// opposed to holding its default.
func (c *cliFlags) passed(name string) bool {
	seen := false
	c.set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})

	return seen
}

// resolveNameFromTags maps the flag onto NameFromTags's three-way contract:
// nil (unset) keeps the historic ["json"] behaviour, while an empty but NON-nil
// slice means "consult no struct tag, use the Go field name". Collapsing those
// two would make `-name-from-tags=` silently mean the opposite of what it says.
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
	// Mute the scanner's logging. codescan writes warnings (unsupported type
	// kinds, skipped builtins, …) through the standard log package, whose
	// default sink is stderr — which paints over bubbletea's alt-screen and
	// corrupts the TUI. Discard it globally for the lifetime of the program.
	// (Reflection: codescan should accept an injected sink / route these
	// through OnDiagnostic instead of the global logger — see plan.)
	log.SetOutput(io.Discard)

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genspec-tui:", err)
		os.Exit(1)
	}
}

// run is main's body, split out so that os.Exit happens with no deferred call
// pending: exiting from inside main skipped `defer model.Close()`, leaving the
// file watcher running until the process died anyway.
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

// splitList parses a comma-separated flag into trimmed, non-empty entries,
// returning nil when there is nothing usable — nil being what codescan reads as
// "no filter".
func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

// splitPatterns parses the comma-separated -packages flag into non-empty,
// trimmed patterns, falling back to "./..." when nothing usable is given.
func splitPatterns(s string) []string {
	if out := splitList(s); len(out) > 0 {
		return out
	}

	return []string{"./..."}
}
