// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"flag"
	"io"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// a flag set standing in for a command's: two sections, one of each kind of value.
type flags struct {
	set     *flag.FlagSet
	workdir *string
	models  *bool
	budget  *float64
	tags    *string
	output  *string
}

func newFlags(t *testing.T, argv ...string) *flags {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	f := &flags{
		set:     fs,
		workdir: fs.String("workdir", ".", ""),
		models:  fs.Bool("scan-models", false, ""),
		budget:  fs.Float64("name-concat-budget", 0, ""),
		tags:    fs.String("exclude-tags", "", ""),
		output:  fs.String("output", "-", ""),
	}
	require.NoError(t, fs.Parse(argv))

	return f
}

func schema() Schema {
	return Schema{
		"workdir":            "scan",
		"exclude-tags":       "scan",
		"scan-models":        "emit",
		"name-concat-budget": "emit",
		"output":             "write",
	}
}

func TestApplyWritesWhatTheFileSays(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	result, err := Apply(f.set, map[string]any{
		"scan.workdir":            "./api",
		"scan.exclude-tags":       []any{"internal", "debug"},
		"emit.scan-models":        true,
		"emit.name-concat-budget": 0.8,
	}, schema())

	require.NoError(t, err)
	assert.Equal(t, "./api", *f.workdir)
	assert.True(t, *f.models)
	assert.InDelta(t, 0.8, *f.budget, 1e-9)
	assert.Equal(t, "internal,debug", *f.tags, "a sequence is the comma-separated form a list flag takes")
	assert.Equal(t, []string{"name-concat-budget", "scan-models", "exclude-tags", "workdir"}, result.Set,
		"reported in the order the keys were read, which is the file's own sorted order")
	assert.Empty(t, result.Ignored)
}

// TestApplyLosesToTheCommandLine is the whole precedence rule, including the case that makes it
// subtle: a flag typed with the value it already had is still typed.
func TestApplyLosesToTheCommandLine(t *testing.T) {
	t.Parallel()

	f := newFlags(t, "-workdir=/typed", "-scan-models=false")

	result, err := Apply(f.set, map[string]any{
		"scan.workdir":     "./from-the-file",
		"emit.scan-models": true,
	}, schema())

	require.NoError(t, err)
	assert.Equal(t, "/typed", *f.workdir)
	assert.False(t, *f.models, "-scan-models=false is a decision, not an absence")
	assert.Empty(t, result.Set)
}

// TestApplyDoesNotMistakeItsOwnWritesForTypedFlags is why what was typed is read once, up front:
// flag.Set records into the same place Visit reads, so asking as we go would make the first key
// applied look like a command-line argument to the second.
func TestApplyDoesNotMistakeItsOwnWritesForTypedFlags(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	result, err := Apply(f.set, map[string]any{
		"scan.workdir":     "./api",
		"emit.scan-models": true,
		"write.output":     "spec.json",
	}, schema())

	require.NoError(t, err)
	assert.Len(t, result.Set, 3, "every key applied, not just the first")
	assert.Equal(t, "spec.json", *f.output)
}

// TestApplySkipsSectionsThisCommandDoesNotKnow is what lets one file serve several commands.
func TestApplySkipsSectionsThisCommandDoesNotKnow(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	result, err := Apply(f.set, map[string]any{
		"scan.workdir": "./api",
		"tui.theme":    "dark",
		"tui.mouse":    true,
	}, schema())

	require.NoError(t, err)
	assert.Equal(t, "./api", *f.workdir)
	assert.Equal(t, []string{"tui.mouse", "tui.theme"}, result.Ignored,
		"reported rather than dropped, so a misspelled section is findable")
}

func TestApplyRefusesAnUnknownKeyInAKnownSection(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	_, err := Apply(f.set, map[string]any{"scan.workdirs": "./api"}, schema())

	require.ErrorIs(t, err, ErrUnknownKey)
	assert.Contains(t, err.Error(), "scan.workdirs")
}

// TestApplySaysWhereAMisplacedKeyBelongs covers the mistake a sectioned file invites: the right
// key, in the wrong section.
func TestApplySaysWhereAMisplacedKeyBelongs(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	_, err := Apply(f.set, map[string]any{"scan.scan-models": true}, schema())

	require.ErrorIs(t, err, ErrUnknownKey)
	assert.Contains(t, err.Error(), `is addressed in section "emit"`)
}

func TestApplyRefusesAKeyOutsideAnySection(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	_, err := Apply(f.set, map[string]any{"workdir": "./api"}, schema())

	require.ErrorIs(t, err, ErrUnknownKey)
	assert.Contains(t, err.Error(), "in no section")
	assert.Contains(t, err.Error(), "emit, scan, write", "and says which sections there are")
}

func TestApplyRefusesAValueTheFlagWillNotTake(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	_, err := Apply(f.set, map[string]any{"emit.scan-models": "yes please"}, schema())

	require.ErrorIs(t, err, ErrBadValue)
	assert.Contains(t, err.Error(), "emit.scan-models")
}

func TestApplyRefusesAValueOfNoUsableKind(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	_, err := Apply(f.set, map[string]any{"scan.workdir": map[string]any{"nested": "deeper"}}, schema())

	require.ErrorIs(t, err, ErrBadValue)
}

// TestApplyTakesAnEmptyValueAsAStatement covers the flags that tell an empty list from an absent
// one: writing the key with nothing after it has to reach them as an explicit empty.
func TestApplyTakesAnEmptyValueAsAStatement(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	result, err := Apply(f.set, map[string]any{"scan.exclude-tags": nil}, schema())

	require.NoError(t, err)
	assert.Empty(t, *f.tags)
	assert.Equal(t, []string{"exclude-tags"}, result.Set, "the flag was set, not skipped")
}

// TestApplyRefusesASchemaThatOutrunsTheFlagSet catches the caller's mistake rather than the file's.
func TestApplyRefusesASchemaThatOutrunsTheFlagSet(t *testing.T) {
	t.Parallel()

	f := newFlags(t)
	broken := schema()
	broken["never-registered"] = "scan"

	_, err := Apply(f.set, map[string]any{"scan.never-registered": "x"}, broken)

	require.ErrorIs(t, err, ErrBadConfig)
	assert.Contains(t, err.Error(), "registered on no flag")
}

func TestApplyOnAnEmptyFile(t *testing.T) {
	t.Parallel()

	f := newFlags(t)

	result, err := Apply(f.set, map[string]any{}, schema())

	require.NoError(t, err)
	assert.Empty(t, result.Set)
	assert.Equal(t, ".", *f.workdir)
}

func TestSchemaMerge(t *testing.T) {
	t.Parallel()

	merged, err := schema().Merge(Schema{"color": "report"})
	require.NoError(t, err)
	assert.Equal(t, "report", merged["color"])
	assert.Equal(t, "scan", merged["workdir"], "and the original survives")
	assert.NotContains(t, schema(), "color", "which is left alone")
}

func TestSchemaMergeRefusesAFlagInTwoSections(t *testing.T) {
	t.Parallel()

	_, err := schema().Merge(Schema{"workdir": "write"})

	require.ErrorIs(t, err, ErrBadConfig)
	assert.Contains(t, err.Error(), "workdir")
}

func TestSchemaSections(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"emit", "scan", "write"}, schema().Sections())
}

func TestStringify(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value any
		want  string
	}{
		{"a string is itself", "yaml", "yaml"},
		{"a bool", true, "true"},
		{"an int", 7, "7"},
		{"a 64-bit int", int64(7), "7"},
		{"a float keeps no spurious decimals", 0.65, "0.65"},
		{"nothing is the empty string", nil, ""},
		{"a list is comma-separated", []any{"a", "b"}, "a,b"},
		{"a list of anything", []any{1, true}, "1,true"},
		{"an empty list", []any{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := stringify(tc.value)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStringifyRefusesWhatAFlagCannotTake(t *testing.T) {
	t.Parallel()

	_, err := stringify(struct{}{})
	require.Error(t, err)

	_, err = stringify([]any{struct{}{}})
	require.Error(t, err, "and inside a list too")
}

func TestRegisterDeclaresTheFlag(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	config := Register(fs)

	require.NoError(t, fs.Parse([]string{"-config", "somewhere.yaml"}))
	assert.Equal(t, "somewhere.yaml", *config)
}
