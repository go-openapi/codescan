// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

// reporterFor builds a reporter writing into a buffer, as the flags given would.
func reporterFor(t *testing.T, argv ...string) (*reporter, *bytes.Buffer) {
	t.Helper()

	fs := parseInto(t, argv...)
	var buf bytes.Buffer

	report, err := newReporter(fs, &buf)
	require.NoError(t, err)

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

func TestReporterWritesWhatItIsGiven(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-color=never")
	report.onDiagnostic(diagnostic(codescan.SeverityWarning, "something to say"))

	assert.Contains(t, out.String(), "something to say")
	assert.Contains(t, out.String(), "WARN")
	assert.Contains(t, out.String(), "code=scan.test")
	assert.Contains(t, out.String(), "line=7")
}

// TestReporterMutesHintsButCountsThem is why the summary exists: a scan can be quiet and still have
// had a great deal to say, and the count is the only hint that -verbose would show it.
func TestReporterMutesHintsButCountsThem(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-color=never")
	report.onDiagnostic(diagnostic(codescan.SeverityHint, "thinking aloud"))

	assert.NotContains(t, out.String(), "thinking aloud")

	require.NoError(t, report.summarize())
	assert.Contains(t, out.String(), "hints=1")
}

// TestReporterOmitsAPositionThereIsNoneOf covers the diagnostics about the document rather than
// about a place in it: a route dropped by a tag rule, a definition pruned.
func TestReporterOmitsAPositionThereIsNoneOf(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-color=never")
	report.onDiagnostic(codescan.Diagnostic{
		Severity: codescan.SeverityWarning,
		Code:     "scan.ignored-by-tag",
		Message:  "a route went away",
	})

	assert.Contains(t, out.String(), "code=scan.ignored-by-tag")
	assert.NotContains(t, out.String(), "line=0", `"file= line=0" is not somewhere a reader can go`)
	assert.NotContains(t, out.String(), "file=")
}

func TestReporterVerboseShowsHints(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-color=never", "-verbose")
	report.onDiagnostic(diagnostic(codescan.SeverityHint, "thinking aloud"))

	assert.Contains(t, out.String(), "thinking aloud")
}

func TestReporterQuietWritesNothingAtAll(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-quiet", "-verbose")
	report.onDiagnostic(diagnostic(codescan.SeverityError, "a real problem"))
	_ = report.summarize()

	assert.Empty(t, out.String())
}

func TestReporterThreshold(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		failOn   string
		severity codescan.Severity
		fails    bool
	}{
		{"never fails on nothing", "never", codescan.SeverityError, false},
		{"error fails on an error", "error", codescan.SeverityError, true},
		{"error ignores a warning", "error", codescan.SeverityWarning, false},
		{"warning fails on a warning", "warning", codescan.SeverityWarning, true},
		{"warning fails on an error too", "warning", codescan.SeverityError, true},
		{"warning ignores a hint", "warning", codescan.SeverityHint, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report, _ := reporterFor(t, "-color=never", "-fail-on="+tc.failOn)
			report.onDiagnostic(diagnostic(tc.severity, "a finding"))

			err := report.summarize()
			if !tc.fails {
				require.NoError(t, err)

				return
			}
			require.ErrorIs(t, err, errDiagnostics)
		})
	}
}

// TestReporterTripsOnAMutedDiagnostic states that policy and volume are separate: -quiet asks for
// silence, not for a different answer.
func TestReporterTripsOnAMutedDiagnostic(t *testing.T) {
	t.Parallel()

	report, _ := reporterFor(t, "-quiet", "-fail-on=warning")
	report.onDiagnostic(diagnostic(codescan.SeverityWarning, "muted, but still true"))

	require.ErrorIs(t, report.summarize(), errDiagnostics)
}

func TestReporterSaysNothingWhenThereIsNothingToSay(t *testing.T) {
	t.Parallel()

	report, out := reporterFor(t, "-color=never")

	require.NoError(t, report.summarize())
	assert.Empty(t, out.String(), "a summary of nothing is noise")
}

func TestReporterRelative(t *testing.T) {
	t.Parallel()

	report := &reporter{root: filepath.FromSlash("/src")}

	assert.Equal(t, filepath.FromSlash("api/models.go"),
		report.relative(filepath.FromSlash("/src/api/models.go")))
	assert.Empty(t, report.relative(""), "a diagnostic about no file in particular stays that way")

	unrelated := &reporter{}
	assert.Equal(t, filepath.FromSlash("/src/api/models.go"),
		unrelated.relative(filepath.FromSlash("/src/api/models.go")),
		"with no root to relate it to, the absolute path is still better than nothing")
}

func TestResolveColor(t *testing.T) {
	t.Parallel()

	always, err := resolveColor(colorAlways, &bytes.Buffer{})
	require.NoError(t, err)
	assert.True(t, always, "a buffer is not a terminal, and -color=always does not ask")

	never, err := resolveColor(colorNever, os.Stderr)
	require.NoError(t, err)
	assert.False(t, never)

	_, err = resolveColor("sometimes", &bytes.Buffer{})
	require.ErrorIs(t, err, errUsage)
}

// TestResolveColorAutoOnSomethingThatIsNotATerminal covers the case every pipeline meets: a
// redirected stream gets no escape codes without anybody having to say so.
func TestResolveColorAutoOnSomethingThatIsNotATerminal(t *testing.T) {
	t.Parallel()

	auto, err := resolveColor(colorAuto, &bytes.Buffer{})
	require.NoError(t, err)
	assert.False(t, auto)

	file, err := os.CreateTemp(t.TempDir(), "out")
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	auto, err = resolveColor(colorAuto, file)
	require.NoError(t, err)
	assert.False(t, auto, "a regular file is not a terminal either")
}

func TestResolveColorAutoRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	// Not on a terminal to begin with, so this states the environment is consulted at all rather than
	// that it wins - which is the most this can check without one.
	auto, err := resolveColor(colorAuto, &bytes.Buffer{})
	require.NoError(t, err)
	assert.False(t, auto)
}
