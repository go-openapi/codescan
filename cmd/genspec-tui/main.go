// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command genspec-tui is an interactive terminal front-end for the codescan
// Swagger-spec generator: a source-tree browser (left), the generated spec
// (right, JSON/YAML), and diagnostics (bottom). It regenerates the whole-scope
// spec on any file change.
//
// This is the scaffold: an empty three-panel UX shell. Scanner wiring,
// file-watching, and diagnostics rendering land in later increments.
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

	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux"
)

func main() {
	// Mute the scanner's logging. codescan writes warnings (unsupported type
	// kinds, skipped builtins, …) through the standard log package, whose
	// default sink is stderr — which paints over bubbletea's alt-screen and
	// corrupts the TUI. Discard it globally for the lifetime of the program.
	// (Reflection: codescan should accept an injected sink / route these
	// through OnDiagnostic instead of the global logger — see plan.)
	log.SetOutput(io.Discard)

	workdir := flag.String("workdir", ".", "module directory where scanning runs (codescan WorkDir)")
	packages := flag.String("packages", "./...", "comma-separated package patterns to scan, relative to -workdir")
	scanModels := flag.Bool("scan-models", true, "also emit definitions for swagger:model types")
	flag.Parse()

	dir, err := filepath.Abs(*workdir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genspec-tui:", err)
		os.Exit(1)
	}

	model := ux.New(dir, splitPatterns(*packages), *scanModels)
	defer model.Close()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "genspec-tui:", err)
		os.Exit(1)
	}
}

// splitPatterns parses the comma-separated -packages flag into non-empty,
// trimmed patterns, falling back to "./..." when nothing usable is given.
func splitPatterns(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"./..."}
	}
	return out
}
