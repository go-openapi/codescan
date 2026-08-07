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

// optionFlags maps each value-typed codescan.Options field to the flag that sets it.
//
// The drift guard checks both directions: every field listed here has a registered flag,
// and every value-typed field is either listed here or explicitly excused below.
var optionFlags = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"WorkDir":          "workdir",
	"Packages":         "packages",
	"BuildTags":        "build-tags",
	"GOOS":             "goos",
	"GOARCH":           "goarch",
	"GOFLAGS":          "goflags",
	"GOWORK":           "gowork",
	"GOEXPERIMENT":     "goexperiment",
	"Include":          "include",
	"Exclude":          "exclude",
	"IncludeTags":      "include-tags",
	"ExcludeTags":      "exclude-tags",
	"NameFromTags":     "name-from-tags",
	"NameConcatBudget": "name-concat-budget",
}

// optionsNotOnCLI are the non-bool fields deliberately without a flag.
var optionsNotOnCLI = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"InputSpec":    "overlay mode: needs a spec loaded from disk, not yet exposed",
	"OnDiagnostic": "wired internally to the diagnostics pane",
	"OnProvenance": "wired internally to the cross-ref linker",
	"FS":           "virtual source filesystem: a programmatic seam, not expressible on a command line",
	"ExportData":   "a filesystem of pre-computed export data: a programmatic seam, like FS",
}

// The CLI exposed three of ten options for a long time, because nothing failed when a new one landed.
//
// This is what fails now.
//
// Booleans are excluded: they are the options overlay's job, and TestOptions_OverlayCoversEveryBoolKnob guards that
// side.
func TestFlags_CoverEveryValueTypedOption(t *testing.T) {
	cli := newTestFlags(t)

	typ := reflect.TypeFor[codescan.Options]()
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() || f.Type.Kind() == reflect.Bool {
			continue
		}

		name, mapped := optionFlags[f.Name]
		if !mapped {
			if _, excused := optionsNotOnCLI[f.Name]; excused {
				continue
			}

			t.Errorf("codescan.Options.%s (%s) has no CLI flag. Add one and list it in "+
				"optionFlags, or excuse it in optionsNotOnCLI with a reason.", f.Name, f.Type)

			continue
		}

		assert.NotNil(t, cli.set.Lookup(name),
			"Options.%s claims flag -%s, which is not registered", f.Name, name)
	}
}

// The option tables must not rot.
//
// An entry naming a field that no longer exists, or one that has since become a bool, is stale.
func TestFlags_TablesAreCurrent(t *testing.T) {
	typ := reflect.TypeFor[codescan.Options]()

	for name := range optionFlags {
		f, ok := typ.FieldByName(name)
		require.True(t, ok, "optionFlags names Options.%s, which no longer exists", name)
		assert.NotEqual(t, reflect.Bool, f.Type.Kind(),
			"Options.%s is a bool and belongs to the overlay, not the CLI", name)
	}

	for name, reason := range optionsNotOnCLI {
		_, ok := typ.FieldByName(name)
		assert.True(t, ok, "optionsNotOnCLI names Options.%s, which no longer exists", name)
		assert.NotEmpty(t, reason, "Options.%s is excused without a reason", name)
		_, alsoMapped := optionFlags[name]
		assert.False(t, alsoMapped, "Options.%s is both excused and mapped to a flag", name)
	}
}

func TestFlags_Defaults(t *testing.T) {
	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse(nil))

	opts := cli.options("/work")

	assert.Equal(t, "/work", opts.WorkDir)
	assert.Equal(t, []string{"./..."}, opts.Packages)
	assert.True(t, opts.ScanModels)
	assert.Empty(t, opts.BuildTags)
	assert.Nil(t, opts.Include)
	assert.Nil(t, opts.ExcludeTags)
	assert.Nil(t, opts.NameFromTags, "unset must stay nil so codescan applies its [\"json\"] default")
	assert.Zero(t, opts.NameConcatBudget, "zero selects codescan's own 0.65 default")
}

func TestFlags_ParseValues(t *testing.T) {
	cli := newTestFlags(t)
	require.NoError(t, cli.set.Parse([]string{
		"-packages", "./api/...,./models/...",
		"-scan-models=false",
		"-build-tags", "integration,dev",
		"-include", "^github.com/me/",
		"-exclude", "vendor,testdata",
		"-include-tags", "public",
		"-exclude-tags", "internal, deprecated",
		"-name-concat-budget", "0.8",
	}))

	opts := cli.options("/work")

	assert.Equal(t, []string{"./api/...", "./models/..."}, opts.Packages)
	assert.False(t, opts.ScanModels)
	assert.Equal(t, "integration,dev", opts.BuildTags, "build tags pass through verbatim")
	assert.Equal(t, []string{"^github.com/me/"}, opts.Include)
	assert.Equal(t, []string{"vendor", "testdata"}, opts.Exclude)
	assert.Equal(t, []string{"public"}, opts.IncludeTags)
	assert.Equal(t, []string{"internal", "deprecated"}, opts.ExcludeTags, "entries are trimmed")
	assert.InDelta(t, 0.8, opts.NameConcatBudget, 1e-9)
}

// NameFromTags is three-way, and flattening it would make -name-from-tags= mean the opposite of what it says.
func TestFlags_NameFromTagsIsThreeWay(t *testing.T) {
	for _, c := range []struct {
		name    string
		args    []string
		want    []string
		wantNil bool
	}{
		{"unset keeps the historic json default", nil, nil, true},
		{"explicit empty means the Go field name", []string{"-name-from-tags="}, []string{}, false},
		{"a list is ordered as given", []string{"-name-from-tags", "form,json"}, []string{"form", "json"}, false},
		{"entries are trimmed", []string{"-name-from-tags", " form , json "}, []string{"form", "json"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			cli := newTestFlags(t)
			require.NoError(t, cli.set.Parse(c.args))

			got := cli.options("/work").NameFromTags

			if c.wantNil {
				assert.Nil(t, got)

				return
			}
			require.NotNil(t, got, "an explicitly passed flag must never yield nil")
			assert.Equal(t, c.want, got)
		})
	}
}

func TestSplitHelpers(t *testing.T) {
	assert.Nil(t, splitList(""))
	assert.Nil(t, splitList("  ,  , "))
	assert.Equal(t, []string{"a", "b"}, splitList(" a , b "))

	assert.Equal(t, []string{"./..."}, splitPatterns(""), "an empty -packages falls back")
	assert.Equal(t, []string{"./..."}, splitPatterns(" , "))
	assert.Equal(t, []string{"./x"}, splitPatterns("./x"))
}

func newTestFlags(t *testing.T) *cliFlags {
	t.Helper()
	fs := flag.NewFlagSet("genspec-tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return registerFlags(fs)
}
