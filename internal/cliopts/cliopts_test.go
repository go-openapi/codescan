// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliopts

import (
	"flag"
	"io"
	"reflect"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// notOnTheCommandLine are the value-typed options deliberately without a flag, with the reason.
//
// Everything else must be reachable. An entry here is a decision, and reading it should be enough to
// judge whether it still holds.
var notOnTheCommandLine = map[string]string{ //nolint:gochecknoglobals // table for the drift guard
	"Packages":    "positional arguments, so that `genspec ./api/...` reads like every other Go command",
	"DescWithRef": "deprecated in favour of EmitRefSiblings",
	"Debug":       "deprecated no-op; the stderr logger was retired",
}

// coveredByLoader is the option the -loader flag discharges.
//
// It is a boolean the tables cannot carry, because the choice has three answers and the useful
// default is the third one. TestLoaderWritesTheToolchainFreeOption is what proves this excuse.
const coveredByLoader = "ToolchainFreeLoader"

// TestFlagsCoverEveryValueTypedOption is what stops a knob being unreachable from the command line.
//
// A caller cannot use an option that has no flag, and finds out by meeting "flag provided but not
// defined" - after writing something against a surface that was never there. Fail here instead, on
// the pull request that adds the option.
func TestFlagsCoverEveryValueTypedOption(t *testing.T) {
	t.Parallel()

	covered := coveredFields(t)
	covered[coveredByLoader] = loaderFlag

	typ := reflect.TypeFor[Options]()
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() || !isValueTyped(f.Type) {
			continue
		}
		if _, excused := notOnTheCommandLine[f.Name]; excused {
			continue
		}

		assert.NotEmptyf(t, covered[f.Name],
			"codescan.Options.%s is reachable from no flag. Add an entry to the tables in cliopts.go, "+
				"or excuse it in notOnTheCommandLine with a reason.", f.Name)
	}
}

// TestFlagTablesAreCurrent catches two flags sharing a name, and two flags writing to the same field
// - either leaves one of them silently doing nothing.
func TestFlagTablesAreCurrent(t *testing.T) {
	t.Parallel()

	names := map[string]bool{loaderFlag: true}
	fields := map[string]string{}

	for field, name := range coveredFields(t) {
		require.Falsef(t, names[name], "flag %q is declared twice", name)
		names[name] = true

		require.Emptyf(t, fields[field], "flags %q and %q both write to Options.%s", fields[field], name, field)
		fields[field] = name
	}
}

// TestExcusedOptionsStillExist keeps notOnTheCommandLine honest.
//
// An excuse for a field that has since been renamed or removed reads as a decision that was made,
// and silently stops guarding anything.
func TestExcusedOptionsStillExist(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[Options]()
	for name := range notOnTheCommandLine {
		_, ok := typ.FieldByName(name)
		assert.Truef(t, ok, "notOnTheCommandLine excuses %s, which codescan.Options no longer has", name)
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()

	opts := parse(t)

	assert.Equal(t, ".", opts.WorkDir)
	assert.True(t, opts.ScanModels, "a command asked for a specification should emit the models it finds")
	assert.False(t, opts.PruneUnusedModels)
	assert.Empty(t, opts.BuildTags)
	assert.Zero(t, opts.NameConcatBudget, "0 is what the library reads as its own default of 0.65")
	assert.Nil(t, opts.Include)
	assert.Nil(t, opts.NameFromTags)
	assert.Equal(t, !canExec(), opts.ToolchainFreeLoader, "-loader defaults to auto")
}

func TestParseEveryKind(t *testing.T) {
	t.Parallel()

	opts := parse(t,
		"-scan-models=false",
		"-prune-unused-models",
		"-workdir=/src",
		"-goexperiment=jsonv2",
		"-name-concat-budget=0.8",
		"-include=a/...,  b/... ",
		"-exclude-tags=internal",
		"-loader=own",
	)

	assert.False(t, opts.ScanModels, "the explicit -name=false form is why booleans are not bare switches")
	assert.True(t, opts.PruneUnusedModels)
	assert.Equal(t, "/src", opts.WorkDir)
	assert.Equal(t, "jsonv2", opts.GOEXPERIMENT)
	assert.InDelta(t, 0.8, opts.NameConcatBudget, 1e-9)
	assert.Equal(t, []string{"a/...", "b/..."}, opts.Include, "entries are trimmed, empties dropped")
	assert.Equal(t, []string{"internal"}, opts.ExcludeTags)
	assert.True(t, opts.ToolchainFreeLoader)
	assert.False(t, opts.SkipExtensions, "an untouched flag must leave its field alone")
}

// TestNameFromTagsIsThreeWay pins the one option where an empty value is not the same as no value.
func TestNameFromTagsIsThreeWay(t *testing.T) {
	t.Parallel()

	t.Run("unset means the library's default", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, parse(t).NameFromTags)
	})

	t.Run("empty means the Go field name", func(t *testing.T) {
		t.Parallel()

		got := parse(t, "-name-from-tags=").NameFromTags
		require.NotNil(t, got, "an explicit empty list must not collapse onto nil, which means json")
		assert.Empty(t, got)
	})

	t.Run("a list is the list", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []string{"form", "json"}, parse(t, "-name-from-tags=form,json").NameFromTags)
	})
}

// TestFiltersAreNotThreeWay states the other half of the contract: everywhere else, empty and absent
// are the same thing, because nil is already what "no filter" means.
func TestFiltersAreNotThreeWay(t *testing.T) {
	t.Parallel()

	assert.Nil(t, parse(t, "-include=").Include)
	assert.Nil(t, parse(t, "-include=  ,  ").Include)
}

func TestLoaderWritesTheToolchainFreeOption(t *testing.T) {
	t.Parallel()

	assert.True(t, parse(t, "-loader=own").ToolchainFreeLoader)
	assert.False(t, parse(t, "-loader=go").ToolchainFreeLoader)
	assert.Equal(t, !canExec(), parse(t, "-loader=auto").ToolchainFreeLoader)
}

func TestLoaderRefusesAnythingElse(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values := Register(fs)
	require.NoError(t, fs.Parse([]string{"-loader=cargo"}))

	err := values.Apply(&Options{})

	require.ErrorIs(t, err, ErrBadFlag)
	assert.Contains(t, err.Error(), "cargo")
}

func TestPatterns(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{DefaultPatterns}, Patterns(nil))
	assert.Equal(t, []string{"./api/..."}, Patterns([]string{"./api/..."}))
}

func TestSplitList(t *testing.T) {
	t.Parallel()

	assert.Nil(t, SplitList(""))
	assert.Nil(t, SplitList(" , "))
	assert.Equal(t, []string{"a", "b"}, SplitList(" a , b "))
}

// parse registers the surface, parses argv and returns the options it produced.
func parse(t *testing.T, argv ...string) Options {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values := Register(fs)
	require.NoError(t, fs.Parse(argv))

	var opts Options
	require.NoError(t, values.Apply(&opts))

	return opts
}

// coveredFields reports which field of the options each flag writes to, keyed by field name.
//
// It asks by writing a sentinel through the setter and seeing what moved, rather than deriving a
// field name from a flag name: a mechanical rendering cannot know that JSONify is one word where
// HTTPServer is two, and a rename would leave the derivation agreeing with itself.
func coveredFields(t *testing.T) map[string]string {
	t.Helper()

	covered := make(map[string]string)
	for _, opt := range boolOptions {
		covered[fieldWrittenBy(t, opt.name, func(o *Options) { *opt.field(o) = true })] = opt.name
	}
	for _, opt := range stringOptions {
		covered[fieldWrittenBy(t, opt.name, func(o *Options) { *opt.field(o) = sentinel })] = opt.name
	}
	for _, opt := range floatOptions {
		covered[fieldWrittenBy(t, opt.name, func(o *Options) { *opt.field(o) = sentinelFloat })] = opt.name
	}
	for _, opt := range listOptions {
		covered[fieldWrittenBy(t, opt.name, func(o *Options) { *opt.field(o) = []string{sentinel} })] = opt.name
	}

	return covered
}

const (
	sentinel      = "cliopts sentinel"
	sentinelFloat = 42.5
)

// fieldWrittenBy runs write against a zero Options and reports the single field that changed.
func fieldWrittenBy(t *testing.T, name string, write func(*Options)) string {
	t.Helper()

	var opts Options
	write(&opts)

	var moved string
	typ := reflect.TypeFor[Options]()
	value := reflect.ValueOf(opts)
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() || !isValueTyped(f.Type) || value.Field(i).IsZero() {
			continue
		}
		require.Emptyf(t, moved, "flag %q writes to both Options.%s and Options.%s", name, moved, f.Name)
		moved = f.Name
	}
	require.NotEmptyf(t, moved, "flag %q writes to no field of codescan.Options", name)

	return moved
}

// isValueTyped reports whether a field is of a kind the tables can carry.
//
// The rest - a filesystem to read, a specification to merge, a callback - is not something a command
// line can state, and is the command's business rather than this package's.
func isValueTyped(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Bool, reflect.String, reflect.Float64:
		return true
	case reflect.Slice:
		return typ.Elem().Kind() == reflect.String
	default:
		return false
	}
}
