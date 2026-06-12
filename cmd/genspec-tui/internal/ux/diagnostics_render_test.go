// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// badMaximumFixture is the minimal diagnostic trigger: a swagger:model whose
// field carries a non-numeric `maximum:`. The parser drops the keyword from the
// spec and emits grammar.CodeInvalidNumber — exactly the soft-diagnostic shape
// the pane is built to surface. Kept inline so the test owns its input and the
// lean TUI module needs no root test-fixture dependency.
const badMaximumFixture = `package diagfixture

// BadMaximum has an invalid maximum: value.
//
// swagger:model BadMaximum
type BadMaximum struct {
	// Count holds an arbitrary count.
	//
	// maximum: notanumber
	Count int ` + "`json:\"count\"`" + `
}
`

// TestDoScanCollectsDiagnostics is the end-to-end wiring proof: a malformed
// numeric validation surfaces the parser's CodeInvalidNumber through
// Options.OnDiagnostic into scanResultMsg.diags, without failing the scan
// (diagnostics never abort the build).
func TestDoScanCollectsDiagnostics(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod":   "module diagfixture\n\ngo 1.25\n",
		"types.go": badMaximumFixture,
	})

	res := doScan(codescan.Options{
		WorkDir:    dir,
		Packages:   []string{"."},
		ScanModels: true,
	})

	if res.err != nil {
		t.Fatalf("scan should not hard-fail on soft diagnostics: %v", res.err)
	}
	if len(res.diags) == 0 {
		t.Fatal("expected at least one diagnostic from the malformed fixture")
	}

	found := false
	for _, d := range res.diags {
		if d.Code == grammar.CodeInvalidNumber {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a %s diagnostic; got %v", grammar.CodeInvalidNumber, codes(res.diags))
	}
}

// TestRenderDiagnostics checks the three render states: clean, hard-error, and
// a soft-diagnostic list with a severity tally and relative paths.
func TestRenderDiagnostics(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		if got, _ := renderDiagnostics("/work", nil, nil, 0); got != "(no diagnostics)" {
			t.Errorf("clean scan: got %q", got)
		}
	})

	t.Run("hard error", func(t *testing.T) {
		got, _ := renderDiagnostics("/work", codescan.ErrCodeScan, nil, 0)
		if !strings.Contains(got, "scan failed") || !strings.Contains(got, codescan.ErrCodeScan.Error()) {
			t.Errorf("hard error not surfaced: %q", got)
		}
	})

	t.Run("soft diagnostics", func(t *testing.T) {
		diags := []grammar.Diagnostic{
			grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
			grammar.Warnf(pos("/work/models/a.go", 20, 5), grammar.CodeAmbiguousEmbed, "ambiguous"),
		}
		got, _ := renderDiagnostics("/work", nil, diags, 0)

		for _, want := range []string{
			"2 diagnostics (1 error, 1 warning)",
			"models/a.go:12:3", // path trimmed to workdir
			"bad maximum",
			string(grammar.CodeInvalidNumber),
			"error",
			"warning",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("rendered diagnostics missing %q in:\n%s", want, got)
			}
		}
		if strings.Contains(got, "/work/models") {
			t.Errorf("absolute path leaked into rendered diagnostics:\n%s", got)
		}
	})
}

// writeModule materializes files (relative path → content) under a fresh temp
// dir and returns it. codescan scans it as a standalone module (it forces
// GOWORK=off), so no go.sum or workspace entry is needed for a stdlib-only tree.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return dir
}

func pos(file string, line, col int) token.Position {
	return token.Position{Filename: file, Line: line, Column: col}
}

func codes(diags []grammar.Diagnostic) []grammar.Code {
	out := make([]grammar.Code, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Code)
	}
	return out
}
