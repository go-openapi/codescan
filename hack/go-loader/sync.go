// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// copied names a declaration taken verbatim out of the Go tree, and where each copy lives.
//
// Copying is a deliberate trade — see internal/packages/list/pkgpattern.go for why — and its whole cost is
// that upstream can move underneath us. This is what makes that visible: a drift report is cheap, and
// silently diverging from the semantics we claim to reproduce is not.
type copied struct {
	upstream string // path under the Go checkout
	local    string // path in this repository
	decls    []string
}

// copies is everything this repository has taken from the Go distribution.
var copies = []copied{ //nolint:gochecknoglobals // a table, read once
	{
		upstream: "src/cmd/internal/pkgpattern/pkgpattern.go",
		local:    "internal/packages/list/pkgpattern.go",
		decls:    []string{"MatchPattern", "matchPatternInternal", "replaceVendor"},
	},
	{
		upstream: "src/cmd/internal/pkgpattern/pat_test.go",
		local:    "internal/packages/list/pkgpattern_test.go",
		decls:    []string{"matchPatternTests", "testPatterns"},
	},
}

func runSync(goRoot string, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	update := fs.Bool("update", false, "rewrite the local copies from upstream instead of only reporting")
	if err := fs.Parse(args); err != nil {
		return err
	}

	drifted := 0
	for _, c := range copies {
		up, err := os.ReadFile(filepath.Join(goRoot, c.upstream))
		if err != nil {
			return fmt.Errorf("read upstream %s: %w", c.upstream, err)
		}
		local, err := os.ReadFile(c.local)
		if err != nil {
			return fmt.Errorf("read local %s: %w", c.local, err)
		}

		for _, decl := range c.decls {
			wantDecl, ok := declaration(string(up), decl)
			if !ok {
				fmt.Printf("GONE     %s: %s no longer exists upstream\n", c.local, decl)
				drifted++

				continue
			}
			gotDecl, ok := declaration(string(local), decl)
			if !ok {
				fmt.Printf("MISSING  %s: %s is not in our copy\n", c.local, decl)
				drifted++

				continue
			}
			if gotDecl != wantDecl {
				fmt.Printf("DRIFTED  %s: %s differs from upstream\n", c.local, decl)
				drifted++

				if *update {
					if err := replaceDeclaration(c.local, decl, wantDecl); err != nil {
						return err
					}
					fmt.Printf("         updated in place\n")
				}

				continue
			}
			fmt.Printf("ok       %s: %s\n", c.local, decl)
		}
	}

	if drifted > 0 && !*update {
		return fmt.Errorf("%w: %d declaration(s) drifted; re-run with -update to refresh",
			errDrift, drifted)
	}

	return nil
}

// errDrift reports that a copied declaration no longer matches upstream.
var errDrift = errors.New("copied declarations are out of date")

// declaration extracts a top-level declaration by name, from its doc comment through its closing
// brace or backquote.
//
// Deliberately textual rather than a go/ast walk: the point is to compare the bytes a reader would
// diff, comments included, not the semantics.
func declaration(src, name string) (string, bool) {
	for _, prefix := range []string{"func " + name + "(", "var " + name + " ="} {
		i := strings.Index(src, "\n"+prefix)
		if i < 0 {
			continue
		}
		start := i + 1

		// Walk back over any doc comment attached to it.
		for {
			prev := strings.LastIndex(src[:start-1], "\n")
			line := src[prev+1 : start-1]
			if !strings.HasPrefix(line, "//") {
				break
			}
			start = prev + 1
		}

		end := strings.Index(src[i:], "\n}\n")
		if strings.HasPrefix(prefix, "var ") {
			end = strings.Index(src[i:], "\n`\n")
		}
		if end < 0 {
			return "", false
		}

		return src[start : i+end+3], true
	}

	return "", false
}

// replaceDeclaration swaps one declaration in a file for another.
func replaceDeclaration(path, name, want string) error {
	// path is always a literal from the copies table above, never anything a caller supplies.
	blob, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	got, ok := declaration(string(blob), name)
	if !ok {
		return fmt.Errorf("%w: %s not found in %s", errDrift, name, path)
	}

	//nolint:gosec // path is a literal from the copies table, not caller input
	return os.WriteFile(path, []byte(strings.Replace(string(blob), got, want, 1)), filePerm)
}
