// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/txtar"

	"github.com/go-openapi/codescan/internal/packages"
)

// The go command's script tests carry hundreds of package trees that somebody thought hard about:
// nested modules, workspaces, vendor directories, `...` in awkward places. That corpus is the
// expensive half of testing pattern resolution, and it already exists.
//
// What is NOT reused is the assertions. Running the scripts as written would mean implementing their
// shell DSL, and there is no need: a real go command is available, so the second strategy IS the
// oracle. We take the trees and compare the two strategies against each other.
//
// The trees are read from the Go checkout and never copied into this repository — they are
// BSD-licensed test data, and vendoring several hundred of them to assert nothing about their
// contents would be a poor trade.

// families of script tests that touch package resolution. Everything else in the corpus is about
// building, testing, downloading — surfaces this loader does not have.
var families = []string{"list_", "mod_", "work_", "vendor"} //nolint:gochecknoglobals // a table

// networkVerbs mark a script that expects the test module proxy. Their trees are incomplete without
// it, so they are skipped rather than run and reported as failures of ours.
var networkVerbs = []string{ //nolint:gochecknoglobals // a table
	"go mod download", "go get", "go mod tidy", "rsc.io", "golang.org/x/",
}

// Permissions for the throwaway trees this tool materialises.
const (
	dirPerm  = 0o750
	filePerm = 0o600
)

// outcome is what happened to one script's tree.
type outcome struct {
	script string
	status string // agree, differ, skip, error
	detail string
}

func runCorpus(goRoot string, args []string) error {
	fs := flag.NewFlagSet("corpus", flag.ContinueOnError)
	only := fs.String("only", "", "run only scripts whose name contains this substring")
	verbose := fs.Bool("v", false, "print every script, not only the ones that differ")
	writeLedger := fs.Bool("write-ledger", false, "rewrite hack/go-loader/ledger.md with the results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := filepath.Join(goRoot, "src", "cmd", "go", "testdata", "script")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read script corpus: %w", err)
	}

	var outcomes []outcome
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".txt") || !inFamily(name) {
			continue
		}
		if *only != "" && !strings.Contains(name, *only) {
			continue
		}
		outcomes = append(outcomes, runScript(filepath.Join(dir, name), name))
	}

	report(outcomes, *verbose)

	if *writeLedger {
		if err := writeLedgerFile(goRoot, outcomes); err != nil {
			return err
		}
		fmt.Println("\nwrote hack/go-loader/ledger.md")
	}

	return nil
}

// runScript materialises one script's tree and compares the two strategies over it.
func runScript(path, name string) outcome {
	blob, err := os.ReadFile(path)
	if err != nil {
		return outcome{name, "error", err.Error()}
	}
	archive := txtar.Parse(blob)

	if reason, skip := skipReason(string(archive.Comment), archive); skip {
		return outcome{name, "skip", reason}
	}

	root, err := os.MkdirTemp("", "go-loader-*")
	if err != nil {
		return outcome{name, "error", err.Error()}
	}
	defer func() { _ = os.RemoveAll(root) }()

	for _, f := range archive.Files {
		dest := filepath.Join(root, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(dest), dirPerm); err != nil {
			return outcome{name, "error", err.Error()}
		}
		if err := os.WriteFile(dest, f.Data, filePerm); err != nil {
			return outcome{name, "error", err.Error()}
		}
	}

	native, nErr := describe(root, packages.StrategyGoPackages)
	own, oErr := describe(root, packages.StrategyToolchainFree)

	switch {
	case nErr != nil && oErr != nil:
		// Neither loader will look at this tree, so it says nothing about either. Many of these fixtures
		// exist precisely to make the go command refuse.
		return outcome{name, "skip", "both strategies reject the tree"}

	case nErr != nil:
		// The go command refuses and we do not. That is the documented direction of this loader — it
		// reads trees with no module, with another module's internal packages imported, with vendoring
		// the go command calls inconsistent — so it is reported apart from real disagreements rather
		// than counted as one.
		return outcome{name, "go-rejects", firstLine(nErr.Error())}

	case oErr != nil:
		return outcome{name, "differ", "toolchain-free failed: " + firstLine(oErr.Error())}

	case len(native) == 0 && len(own) > 0:
		// go list matched nothing here — usually the workspace-root case, where "./..." is not a legal
		// pattern for the go command at all. Again our own direction, not a disagreement.
		return outcome{name, "go-rejects", "go list matched no package; we found " + strconv.Itoa(len(own))}

	case !slices.Equal(native, own):
		return outcome{name, "differ", diff(native, own)}

	default:
		return outcome{name, "agree", fmt.Sprintf("%d package(s)", len(native))}
	}
}

// describe loads "./..." and renders each package as "importpath file,file" — the two things a
// resolver decides, and the two the emitted spec depends on.
func describe(root string, strategy packages.Strategy) ([]string, error) {
	pkgs, err := packages.NewLoader(packages.WithStrategy(strategy)).
		Load(&packages.Config{Dir: root}, "./...")
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		// go list -e reports an unmatched pattern as a package with the pattern for a name and no files
		// at all. It is an error placeholder, not something a scan would ever read, and counting it as a
		// package makes every such tree look like a disagreement.
		if len(p.GoFiles) == 0 {
			continue
		}

		files := make([]string, 0, len(p.GoFiles))
		for _, f := range p.GoFiles {
			files = append(files, filepath.Base(f))
		}
		sort.Strings(files)
		out = append(out, p.PkgPath+" "+strings.Join(files, ","))
	}
	sort.Strings(out)

	return out, nil
}

// skipReason reports why a script's tree cannot stand on its own.
func skipReason(preamble string, archive *txtar.Archive) (string, bool) {
	for _, verb := range networkVerbs {
		if strings.Contains(preamble, verb) {
			return "needs the module proxy (" + verb + ")", true
		}
	}
	// GO111MODULE=off is GOPATH mode, which this loader deliberately does not implement.
	if strings.Contains(preamble, "GO111MODULE=off") {
		return "GOPATH mode, not supported", true
	}

	hasGoMod, hasGo := false, false
	for _, f := range archive.Files {
		switch {
		case f.Name == "go.mod" || strings.HasSuffix(f.Name, "/go.mod"):
			hasGoMod = true
		case strings.HasSuffix(f.Name, ".go"):
			hasGo = true
		}
	}
	if !hasGoMod {
		return "no go.mod in the tree", true
	}
	if !hasGo {
		return "no Go source in the tree", true
	}

	return "", false
}

func inFamily(name string) bool {
	for _, f := range families {
		if strings.HasPrefix(name, f) {
			return true
		}
	}

	return false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")

	return line
}

// diff renders the first few asymmetries between two package descriptions.
func diff(native, own []string) string {
	var only []string
	for _, s := range native {
		if !slices.Contains(own, s) {
			only = append(only, "go/packages only: "+s)
		}
	}
	for _, s := range own {
		if !slices.Contains(native, s) {
			only = append(only, "toolchain-free only: "+s)
		}
	}
	const maxShown = 3
	if len(only) > maxShown {
		only = append(only[:maxShown], fmt.Sprintf("(+%d more)", len(only)-maxShown))
	}

	return strings.Join(only, "; ")
}

func report(outcomes []outcome, verbose bool) {
	counts := map[string]int{}
	for _, o := range outcomes {
		counts[o.status]++
		if o.status == "differ" || o.status == "error" || verbose {
			fmt.Printf("%-8s %-44s %s\n", strings.ToUpper(o.status), o.script, o.detail)
		}
	}

	fmt.Printf("\nscripts=%d agree=%d differ=%d go-rejects=%d skip=%d error=%d\n",
		len(outcomes), counts["agree"], counts["differ"], counts["go-rejects"], counts["skip"], counts["error"])
}
