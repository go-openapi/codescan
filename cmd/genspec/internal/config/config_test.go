// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	"flag"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec/internal/diagnostics"
	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/swag/conv"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	for tc := range formatTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveFormat(conv.Pointer(tc.format), conv.Pointer(tc.output))
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("should reject unsupported format", func(t *testing.T) {
		t.Parallel()

		_, err := resolveFormat(conv.Pointer("toml"), conv.Pointer("-"))

		require.ErrorIs(t, err, sentinel.ErrUsage)
		assert.Contains(t, err.Error(), "toml")
	})

	t.Run("should not break on nil input", func(t *testing.T) {
		t.Parallel()

		_, err := resolveFormat(nil, nil)

		require.ErrorIs(t, err, sentinel.ErrUsage)
	})
}

func TestResolveColor(t *testing.T) {
	// not parallel as tests tamper with the environment

	t.Run("should resolve colors", func(t *testing.T) {
		t.Run("with ColorAlways", func(t *testing.T) {
			always, err := resolveColor(conv.Pointer(ColorAlways), &bytes.Buffer{})
			require.NoError(t, err)
			assert.True(t, always, "a buffer is not a terminal, and -color=always does not ask")
		})

		t.Run("with ColorNever", func(t *testing.T) {
			never, err := resolveColor(conv.Pointer(ColorNever), os.Stderr)
			require.NoError(t, err)
			assert.False(t, never)
		})

		t.Run("with invalid value", func(t *testing.T) {
			_, err := resolveColor(conv.Pointer("sometimes"), &bytes.Buffer{})
			require.ErrorIs(t, err, sentinel.ErrUsage)
		})

		t.Run("without a terminal", func(t *testing.T) {
			// Covers the case every pipeline meets: a redirected stream gets no escape codes without anybody having to say so.
			auto, err := resolveColor(conv.Pointer(ColorAuto), &bytes.Buffer{})
			require.NoError(t, err)
			assert.False(t, auto)

			file, err := os.CreateTemp(t.TempDir(), "out")
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			auto, err = resolveColor(conv.Pointer(ColorAuto), file)
			require.NoError(t, err)
			assert.False(t, auto, "a regular file is not a terminal either")
		})

		t.Run("with env var NO_COLOR", func(t *testing.T) {
			t.Setenv("NO_COLOR", "1")
			// Not on a terminal to begin with, so this states the environment is consulted at all rather than
			// that it wins - which is the most this can check without one.
			auto, err := resolveColor(conv.Pointer(ColorAuto), &bytes.Buffer{})
			require.NoError(t, err)
			assert.False(t, auto)
		})

		t.Run("with env var TERM", func(t *testing.T) {
			t.Setenv("TERM", "dum")
			// Not on a terminal to begin with, so this states the environment is consulted at all rather than
			// that it wins - which is the most this can check without one.
			auto, err := resolveColor(conv.Pointer(ColorAuto), &bytes.Buffer{})
			require.NoError(t, err)
			assert.False(t, auto)
		})

		t.Run("should not break on nil input", func(t *testing.T) {
			auto, err := resolveColor(nil, nil)
			require.Error(t, err)
			assert.False(t, auto)
		})
	})
}

func TestResolveFailOn(t *testing.T) {
	t.Parallel()

	_, failing, err := resolveFailOn(conv.Pointer(FailNever))
	require.NoError(t, err)
	assert.False(t, failing, "the default lets a scan report whatever it likes and still succeed")

	severity, failing, err := resolveFailOn(conv.Pointer(FailError))
	require.NoError(t, err)
	assert.True(t, failing)
	assert.Equal(t, codescan.SeverityError, severity)

	severity, failing, err = resolveFailOn(conv.Pointer(FailWarning))
	require.NoError(t, err)
	assert.True(t, failing)
	assert.Equal(t, codescan.SeverityWarning, severity)

	_, _, err = resolveFailOn(conv.Pointer("hint"))
	require.ErrorIs(t, err, sentinel.ErrUsage, "hints are not a threshold: a scan that fails on them fails always")

	_, _, err = resolveFailOn(nil)
	require.ErrorIs(t, err, sentinel.ErrUsage, "nil is converted to the zero value")
}

// TestFlagsReachTheLibrary states the point of sharing the table with the other commands: what the
// library takes is registered here too, and lands where it says.
func TestFlagsReachTheLibrary(t *testing.T) {
	t.Parallel()

	cfg := parseInto(t,
		"-workdir=/src", "-scan-models=false", "-exclude-tags=internal", "-emit-x-go-type",
	)

	var opts codescan.Options
	require.NoError(t, cfg.scan.Apply(&opts))

	assert.Equal(t, "/src", opts.WorkDir)
	assert.False(t, opts.ScanModels)
	assert.Equal(t, []string{"internal"}, opts.ExcludeTags)
	assert.True(t, opts.EmitXGoType)
}

func TestOptionsRefusesAnUnknownLoader(t *testing.T) {
	t.Parallel()

	t.Run("should refuse an unknown loaded", func(t *testing.T) {
		t.Parallel()

		cfg := parseInto(t, "-loader=cargo")

		_, err := cfg.options(nil, &diagnostics.Reporter{})

		require.Error(t, err)
		require.ErrorIsf(t, err, sentinel.ErrUsage, "a flag value that does not exist is a usage error")
	})

	t.Run("should default package path to the current working directory", func(t *testing.T) {
		// Covers the argument that is not a flag.
		t.Parallel()

		cfg := parseInto(t)

		opts, err := cfg.options(nil, &diagnostics.Reporter{})
		require.NoError(t, err)
		assert.Equal(t, []string{"./..."}, opts.Packages)

		opts, err = cfg.options([]string{"./api/..."}, &diagnostics.Reporter{})
		require.NoError(t, err)
		assert.Equal(t, []string{"./api/..."}, opts.Packages)
	})
}

// PrepareScan settles the whole command line at once, so a value none of the resolvers can make sense of has to come
// back from here rather than reach a scan and be discovered halfway through one.
func TestPrepareScan(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name string
		argv []string
	}{
		{"should refuse a format it does not know", []string{"-format", "toml"}},
		{"should refuse a colour policy it does not know", []string{"-color", "sometimes"}},
		{"should refuse a threshold it does not know", []string{"-fail-on", "hint"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			cfg := parseInto(t, append([]string{"-no-config"}, c.argv...)...)

			_, _, err := cfg.PrepareScan()

			require.ErrorIs(t, err, sentinel.ErrUsage)
		})
	}

	t.Run("should refuse an -input it cannot read", func(t *testing.T) {
		// The one refusal that comes from the options rather than from a flag's own value.
		t.Parallel()

		cfg := parseInto(t, "-no-config", "-input", filepath.Join(t.TempDir(), "absent.json"))

		_, _, err := cfg.PrepareScan()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot read -input")
	})

	t.Run("should settle a command line with nothing wrong with it", func(t *testing.T) {
		t.Parallel()

		cfg := parseInto(t, "-no-config", "-workdir", ".", "./api/...")

		opts, report, err := cfg.PrepareScan()
		require.NoError(t, err)

		here, err := filepath.Abs(".")
		require.NoError(t, err)

		require.NotNil(t, report)
		assert.Equal(t, []string{"./api/..."}, opts.Packages)
		assert.Equal(t, ".", opts.WorkDir, "the scan runs from where the command did, so it is left as it was given")
		assert.Equal(t, here, report.Root,
			"resolved only for the reporter, which relates the scanner's absolute positions back to it")
		assert.NotNil(t, opts.OnDiagnostic, "and everything the scan observes has somewhere to go")
	})
}

/* Format. */
type formatTestCase struct {
	name   string
	format string
	output string
	want   string
}

func formatTestCases() iter.Seq[formatTestCase] {
	return slices.Values([]formatTestCase{
		{"json is json", FormatJSON, "-", FormatJSON},
		{"yaml is yaml", FormatYAML, "-", FormatYAML},
		{"auto to standard output is json", FormatAuto, "-", FormatJSON},
		{"auto reads a .yaml extension", FormatAuto, "spec.yaml", FormatYAML},
		{"auto reads a .yml extension", FormatAuto, "spec.yml", FormatYAML},
		{"auto reads it whatever its case", FormatAuto, "SPEC.YAML", FormatYAML},
		{"auto falls back to json", FormatAuto, "spec.json", FormatJSON},
		{"auto on a file with no extension is json", FormatAuto, "spec", FormatJSON},
		{"an explicit format wins over the name", FormatJSON, "spec.yaml", FormatJSON},
	})
}

// parseInto registers the command line and parses argv into it.
func parseInto(t *testing.T, argv ...string) *Config {
	t.Helper()

	fs := flag.NewFlagSet("genspec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := NewWithFlags(fs, os.Stdout, os.Stderr)
	require.NoError(t, fs.Parse(argv))

	return cfg
}
