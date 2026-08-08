// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"io"
	"reflect"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// boolsNotOnCLI are the boolean options deliberately without a flag, with the reason.
var boolsNotOnCLI = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"DescWithRef":         "deprecated in favour of EmitRefSiblings",
	"Debug":               "deprecated no-op; the stderr logger was retired",
	"ToolchainFreeLoader": "reached through -loader, which also has an auto mode a boolean cannot state",
}

// TestFlagsCoverEveryBoolOption is what stops a knob being unreachable from the command line.
//
// A caller cannot use an option that has no flag, and finds out by meeting "flag provided but not
// defined" - after writing something against a surface that was never there. Fail here instead.
//
// Coverage is decided by which field each entry actually writes to, not by deriving a name from the
// field: a mechanical rendering cannot know that JSONify is one word and HTTPServer is two.
func TestFlagsCoverEveryBoolOption(t *testing.T) {
	t.Parallel()

	covered := map[string]string{}
	for _, opt := range boolOptions {
		covered[fieldWrittenBy(t, opt)] = opt.name
	}

	typ := reflect.TypeFor[codescan.Options]()
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() || f.Type.Kind() != reflect.Bool {
			continue
		}
		if _, excused := boolsNotOnCLI[f.Name]; excused {
			continue
		}

		assert.NotEmpty(t, covered[f.Name],
			"codescan.Options.%s is reachable from no flag. Add an entry to boolOptions, or excuse it "+
				"in boolsNotOnCLI with a reason.", f.Name)
	}
}

// fieldWrittenBy reports which field of codescan.Options an entry's setter targets, by writing
// through it and seeing what moved.
func fieldWrittenBy(t *testing.T, opt boolOption) string {
	t.Helper()

	var opts codescan.Options
	*opt.field(&opts) = true

	typ := reflect.TypeFor[codescan.Options]()
	value := reflect.ValueOf(opts)
	for i := range typ.NumField() {
		if typ.Field(i).Type.Kind() == reflect.Bool && value.Field(i).Bool() {
			return typ.Field(i).Name
		}
	}
	require.Fail(t, "flag "+opt.name+" writes to no field of codescan.Options")

	return ""
}

// TestBoolFlagTableIsCurrent catches two flags sharing a name, and two flags writing to the same
// field - either leaves one of them silently doing nothing.
func TestBoolFlagTableIsCurrent(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}
	fields := map[string]string{}

	for _, opt := range boolOptions {
		require.False(t, names[opt.name], "flag %q is declared twice", opt.name)
		names[opt.name] = true

		field := fieldWrittenBy(t, opt)
		require.Empty(t, fields[field],
			"flags %q and %q both write to Options.%s", fields[field], opt.name, field)
		fields[field] = opt.name
	}
}

func TestBoolFlagsRegisterAndApply(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("genspec-wasi", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	parsed := registerBools(fs)

	// scan-models defaults true, so turning it off proves the explicit -name=false form works - the
	// reason booleans are rendered that way rather than as bare switches.
	require.NoError(t, fs.Parse([]string{"-prune-unused-models=true", "-scan-models=false"}))

	var opts codescan.Options
	applyBools(&opts, parsed)

	assert.True(t, opts.PruneUnusedModels)
	assert.False(t, opts.ScanModels)
	assert.False(t, opts.SkipExtensions, "an untouched flag must leave its field alone")
}
