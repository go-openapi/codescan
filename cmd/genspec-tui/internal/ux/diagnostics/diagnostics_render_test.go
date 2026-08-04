// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
)

// TestRenderDiagnostics checks the three render states: clean, hard-error, and a soft-diagnostic list with a severity
// tally and relative paths.
func TestRenderDiagnostics(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		if got, _ := Render("/work", nil, nil, 0, true); got != "(no diagnostics)" {
			t.Errorf("clean scan: got %q", got)
		}
	})

	t.Run("hard error", func(t *testing.T) {
		got, _ := Render("/work", codescan.ErrCodeScan, nil, 0, true)
		if !strings.Contains(got, "scan failed") || !strings.Contains(got, codescan.ErrCodeScan.Error()) {
			t.Errorf("hard error not surfaced: %q", got)
		}
	})

	t.Run("soft diagnostics", func(t *testing.T) {
		diags := []grammar.Diagnostic{
			grammar.Errorf(pos("/work/models/a.go", 12, 3), grammar.CodeInvalidNumber, "bad maximum"),
			grammar.Warnf(pos("/work/models/a.go", 20, 5), grammar.CodeAmbiguousEmbed, "ambiguous"),
		}
		got, _ := Render("/work", nil, diags, 0, true)

		for _, want := range []string{
			"2 diagnostics (1 error, 1 warning)",
			filepath.FromSlash("models/a.go") + ":12:3", // trimmed to workdir, native separators
			"bad maximum",
			string(grammar.CodeInvalidNumber),
			"error",
			"warning",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("rendered diagnostics missing %q in:\n%s", want, got)
			}
		}
		if strings.Contains(got, filepath.FromSlash("/work/models")) {
			t.Errorf("absolute path leaked into rendered diagnostics:\n%s", got)
		}
	})
}

func pos(file string, line, col int) token.Position {
	return token.Position{Filename: file, Line: line, Column: col}
}
