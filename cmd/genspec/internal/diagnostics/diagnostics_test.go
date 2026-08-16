// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import (
	"bytes"
	"go/token"
	"iter"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

func TestReporter(t *testing.T) {
	t.Parallel()

	t.Run("should write what it is given", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})
		report.OnDiagnostic(diagnostic(codescan.SeverityWarning, "something to say"))

		assert.Contains(t, out.String(), "something to say")
		assert.Contains(t, out.String(), "WARN")
		assert.Contains(t, out.String(), "code=scan.test")
		assert.Contains(t, out.String(), "line=7")
	})

	t.Run("should mute hints but count them", func(t *testing.T) {
		// This is why the summary exists: a scan can be quiet and still have
		// had a great deal to say, and the count is the only hint that -verbose would show it.
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})
		report.OnDiagnostic(diagnostic(codescan.SeverityHint, "thinking aloud"))

		assert.NotContains(t, out.String(), "thinking aloud")

		require.NoError(t, report.Summarize())
		assert.Contains(t, out.String(), "hints=1")
	})

	t.Run("should omit a position when there is none", func(t *testing.T) {
		// Covers the diagnostics about the document rather than about a place in it:
		// a route dropped by a tag rule, a definition pruned.
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})
		report.OnDiagnostic(codescan.Diagnostic{
			Severity: codescan.SeverityWarning,
			Code:     "scan.ignored-by-tag",
			Message:  "a route went away",
		})

		assert.Contains(t, out.String(), "code=scan.ignored-by-tag")
		assert.NotContains(t, out.String(), "line=0", `"file= line=0" is not somewhere a reader can go`)
		assert.NotContains(t, out.String(), "file=")
	})

	t.Run("should show hints when verbose", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Verbose: true, Failing: true})
		report.OnDiagnostic(diagnostic(codescan.SeverityHint, "thinking aloud"))

		assert.Contains(t, out.String(), "thinking aloud")
	})

	t.Run("should write nothing when quiet", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Quiet: true, Verbose: true, Failing: true})
		report.OnDiagnostic(diagnostic(codescan.SeverityError, "a real problem"))
		_ = report.Summarize()

		assert.Empty(t, out.String())
	})

	t.Run("should write when threshold is met", func(t *testing.T) {
		t.Parallel()

		for tc := range reporterTestCases() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				report, _ := reporterFor(t, ReporterConfig{FailThreshold: tc.failOn, Failing: tc.failing})
				report.OnDiagnostic(diagnostic(tc.severity, "a finding"))

				err := report.Summarize()
				if !tc.fails {
					require.NoError(t, err)

					return
				}
				require.ErrorIs(t, err, sentinel.ErrDiagnostics)
			})
		}
	})

	t.Run("should trip on a mute diagnostic", func(t *testing.T) {
		// States that policy and volume are separate: -quiet asks for silence, not for a different answer.
		t.Parallel()

		report, _ := reporterFor(t, ReporterConfig{Quiet: true, FailThreshold: codescan.SeverityWarning, Failing: true})
		report.OnDiagnostic(diagnostic(codescan.SeverityWarning, "muted, but still true"))

		require.ErrorIs(t, report.Summarize(), sentinel.ErrDiagnostics)
	})

	t.Run("should say nothing when there is nothing to say", func(t *testing.T) {
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})

		require.NoError(t, report.Summarize())
		assert.Empty(t, out.String(), "a summary of nothing is noise")
	})

	t.Run("should resolve relative paths", func(t *testing.T) {
		t.Parallel()

		report := &Reporter{Root: filepath.FromSlash("/src")}

		assert.Equal(t, filepath.FromSlash("api/models.go"),
			report.relative(filepath.FromSlash("/src/api/models.go")))
		assert.Empty(t, report.relative(""), "a diagnostic about no file in particular stays that way")

		unrelated := &Reporter{}
		assert.Equal(t, filepath.FromSlash("/src/api/models.go"),
			unrelated.relative(filepath.FromSlash("/src/api/models.go")),
			"with no Root to relate it to, the absolute path is still better than nothing")
	})

	t.Run("should keep a path it cannot relate to the root", func(t *testing.T) {
		// A relative path cannot be expressed against an absolute root without knowing where it stands, and
		// whatever the scanner recorded is still more use to a reader than nothing at all.
		t.Parallel()

		report := &Reporter{Root: filepath.FromSlash("/src")}

		assert.Equal(t, filepath.FromSlash("api/models.go"), report.relative(filepath.FromSlash("api/models.go")))
	})

	t.Run("should write a severity it does not know about", func(t *testing.T) {
		// The three are what codescan reports today. A fourth would be worth reading rather than worth dropping,
		// so it is written at the level a reader can least afford to miss it at.
		t.Parallel()

		report, out := reporterFor(t, ReporterConfig{Failing: true})
		report.OnDiagnostic(diagnostic(codescan.Severity(42), "from a severity nobody has declared"))

		assert.Contains(t, out.String(), "from a severity nobody has declared")
	})
}

type reporterTestCase struct {
	name     string
	failOn   codescan.Severity
	failing  bool
	severity codescan.Severity
	fails    bool
}

func reporterTestCases() iter.Seq[reporterTestCase] {
	return slices.Values([]reporterTestCase{
		{"never fails on nothing", codescan.SeverityError, false, codescan.SeverityError, false},
		{"error fails on an error", codescan.SeverityError, true, codescan.SeverityError, true},
		{"error ignores a warning", codescan.SeverityError, true, codescan.SeverityWarning, false},
		{"warning fails on a warning", codescan.SeverityWarning, true, codescan.SeverityWarning, true},
		{"warning fails on an error too", codescan.SeverityWarning, true, codescan.SeverityError, true},
		{"warning ignores a hint", codescan.SeverityWarning, true, codescan.SeverityHint, false},
	})
}

// reporterFor builds a reporter writing into a buffer, as the flags given would.
func reporterFor(t *testing.T, cfg ReporterConfig) (*Reporter, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	cfg.Stderr = &buf
	report := NewReporter(cfg)

	return report, &buf
}

func diagnostic(severity codescan.Severity, message string) codescan.Diagnostic {
	return codescan.Diagnostic{
		Severity: severity,
		Code:     "scan.test",
		Message:  message,
		Pos:      token.Position{Filename: "/src/api/models.go", Line: 7, Column: 3},
	}
}
