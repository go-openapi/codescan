// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"io"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

// parseInto registers the command line and parses argv into it.
func parseInto(t *testing.T, argv ...string) *config {
	t.Helper()

	fs := flag.NewFlagSet("genspec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := registerFlags(fs)
	require.NoError(t, fs.Parse(argv))

	return cfg
}

func TestResolveFormat(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		format string
		output string
		want   string
	}{
		{"json is json", formatJSON, "-", formatJSON},
		{"yaml is yaml", formatYAML, "-", formatYAML},
		{"auto to standard output is json", formatAuto, "-", formatJSON},
		{"auto reads a .yaml extension", formatAuto, "spec.yaml", formatYAML},
		{"auto reads a .yml extension", formatAuto, "spec.yml", formatYAML},
		{"auto reads it whatever its case", formatAuto, "SPEC.YAML", formatYAML},
		{"auto falls back to json", formatAuto, "spec.json", formatJSON},
		{"auto on a file with no extension is json", formatAuto, "spec", formatJSON},
		{"an explicit format wins over the name", formatJSON, "spec.yaml", formatJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveFormat(tc.format, tc.output)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveFormatRefusesAnythingElse(t *testing.T) {
	t.Parallel()

	_, err := resolveFormat("toml", "-")

	require.ErrorIs(t, err, errUsage)
	assert.Contains(t, err.Error(), "toml")
}

func TestResolveFailOn(t *testing.T) {
	t.Parallel()

	_, failing, err := resolveFailOn(failNever)
	require.NoError(t, err)
	assert.False(t, failing, "the default lets a scan report whatever it likes and still succeed")

	severity, failing, err := resolveFailOn(failError)
	require.NoError(t, err)
	assert.True(t, failing)
	assert.Equal(t, codescan.SeverityError, severity)

	severity, failing, err = resolveFailOn(failWarning)
	require.NoError(t, err)
	assert.True(t, failing)
	assert.Equal(t, codescan.SeverityWarning, severity)

	_, _, err = resolveFailOn("hint")
	require.ErrorIs(t, err, errUsage, "hints are not a threshold: a scan that fails on them fails always")
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

	cfg := parseInto(t, "-loader=cargo")

	_, err := cfg.options(nil, &reporter{})

	require.Error(t, err)
	assert.Equal(t, exitUsage, exitStatus(err), "a flag value that does not exist is a usage error")
}

// TestOptionsDefaultsToEverythingUnderTheWorkingDirectory covers the argument that is not a flag.
func TestOptionsDefaultsToEverythingUnderTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	cfg := parseInto(t)

	opts, err := cfg.options(nil, &reporter{})
	require.NoError(t, err)
	assert.Equal(t, []string{"./..."}, opts.Packages)

	opts, err = cfg.options([]string{"./api/..."}, &reporter{})
	require.NoError(t, err)
	assert.Equal(t, []string{"./api/..."}, opts.Packages)
}
