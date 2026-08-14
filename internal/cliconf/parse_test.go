// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package cliconf

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	values, err := Parse([]byte(`
scan:
  workdir: ./api
  exclude-tags: [internal, debug]
emit:
  scan-models: true
  name-concat-budget: 0.8
`))

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"scan.workdir":            "./api",
		"scan.exclude-tags":       []any{"internal", "debug"},
		"emit.scan-models":        true,
		"emit.name-concat-budget": 0.8,
	}, values)
}

// TestParseReadsJSON: JSON is a subset of YAML, so a file written that way needs no parser of its
// own - which is why .codescan.json is in the search list.
func TestParseReadsJSON(t *testing.T) {
	t.Parallel()

	values, err := Parse([]byte(`{"scan": {"workdir": "./api"}, "emit": {"scan-models": true}}`))

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"scan.workdir": "./api", "emit.scan-models": true}, values)
}

func TestParseAnEmptyFileSaysNothing(t *testing.T) {
	t.Parallel()

	for _, content := range []string{"", "\n", "# nothing but a comment\n"} {
		values, err := Parse([]byte(content))

		require.NoErrorf(t, err, "%q", content)
		assert.Emptyf(t, values, "%q", content)
	}
}

func TestParseRefusesWhatIsNotAConfiguration(t *testing.T) {
	t.Parallel()

	_, err := Parse([]byte("scan: [this is a list, not a section\n"))

	require.ErrorIs(t, err, ErrBadConfig)
}

// TestFlattenLeavesListsAlone: a list is a value - it is how a repeated flag is written - and
// descending into one would turn the entries into keys named after their positions.
func TestFlattenLeavesListsAlone(t *testing.T) {
	t.Parallel()

	flat := Flatten(map[string]any{
		"scan": map[string]any{"exclude-tags": []any{"a", "b"}},
	})

	assert.Equal(t, map[string]any{"scan.exclude-tags": []any{"a", "b"}}, flat)
}

func TestFlattenGoesAsDeepAsItIsGiven(t *testing.T) {
	t.Parallel()

	flat := Flatten(map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}})

	assert.Equal(t, map[string]any{"a.b.c": 1}, flat,
		"a key deeper than a section is carried through, for Apply to refuse by name")
}

func TestSplit(t *testing.T) {
	t.Parallel()

	section, name := Split("scan.workdir")
	assert.Equal(t, "scan", section)
	assert.Equal(t, "workdir", name)

	section, name = Split("workdir")
	assert.Empty(t, section, "a key in no section")
	assert.Equal(t, "workdir", name)

	section, name = Split("a.b.c")
	assert.Equal(t, "a", section)
	assert.Equal(t, "b.c", name, "only the first segment is the section")
}

// TestYAMLMarshalRoundTrips covers the half of koanf's parser interface this package does not use
// itself, but must supply.
func TestYAMLMarshalRoundTrips(t *testing.T) {
	t.Parallel()

	data, err := YAML{}.Marshal(map[string]any{"scan": map[string]any{"workdir": "./api"}})
	require.NoError(t, err)

	values, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"scan.workdir": "./api"}, values)
}
