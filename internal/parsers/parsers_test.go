// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestMatchParamIn(t *testing.T) {
	t.Parallel()

	mp := NewMatchParamIn(nil)
	assert.TrueT(t, mp.Matches("In: query"))
	assert.TrueT(t, mp.Matches("in: body"))
	assert.TrueT(t, mp.Matches("in: path"))
	assert.TrueT(t, mp.Matches("in: header"))
	assert.TrueT(t, mp.Matches("in: formData"))
	assert.FalseT(t, mp.Matches("in: cookie")) // not a valid swagger 2.0 location
	assert.FalseT(t, mp.Matches("something else"))

	// Parse is a no-op
	require.NoError(t, mp.Parse(nil))
}

func TestMatchParamRequired(t *testing.T) {
	t.Parallel()

	mp := NewMatchParamRequired(nil)
	assert.TrueT(t, mp.Matches("required: true"))
	assert.TrueT(t, mp.Matches("Required: false"))
	assert.FalseT(t, mp.Matches("something else"))

	// Parse is a no-op
	require.NoError(t, mp.Parse(nil))
}

func TestSetDeprecatedOp(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		op := new(oaispec.Operation)
		sd := NewSetDeprecatedOp(op)
		assert.TrueT(t, sd.Matches("deprecated: true"))
		require.NoError(t, sd.Parse([]string{"deprecated: true"}))
		assert.TrueT(t, op.Deprecated)
	})

	t.Run("false", func(t *testing.T) {
		op := new(oaispec.Operation)
		sd := NewSetDeprecatedOp(op)
		require.NoError(t, sd.Parse([]string{"deprecated: false"}))
		assert.FalseT(t, op.Deprecated)
	})

	t.Run("empty", func(t *testing.T) {
		op := new(oaispec.Operation)
		sd := NewSetDeprecatedOp(op)
		require.NoError(t, sd.Parse(nil))
		require.NoError(t, sd.Parse([]string{}))
		require.NoError(t, sd.Parse([]string{""}))
		assert.FalseT(t, op.Deprecated)
	})

	t.Run("no match", func(t *testing.T) {
		sd := NewSetDeprecatedOp(new(oaispec.Operation))
		assert.FalseT(t, sd.Matches("something else"))
	})
}

func TestConsumesDropEmptyParser(t *testing.T) {
	t.Parallel()

	var got []string
	cp := NewConsumesDropEmptyParser(func(v []string) { got = v })
	assert.TrueT(t, cp.Matches("consumes:"))
	assert.TrueT(t, cp.Matches("Consumes:"))
	assert.FalseT(t, cp.Matches("other"))

	// Q4: body is YAML-list-strict. Input uses `- value` markers.
	require.NoError(t, cp.Parse([]string{"- application/json", "", "- application/xml", "  "}))
	assert.Equal(t, []string{"application/json", "application/xml"}, got)
}

func TestProducesDropEmptyParser(t *testing.T) {
	t.Parallel()

	var got []string
	pp := NewProducesDropEmptyParser(func(v []string) { got = v })
	assert.TrueT(t, pp.Matches("produces:"))
	assert.TrueT(t, pp.Matches("Produces:"))

	require.NoError(t, pp.Parse([]string{"- text/plain", "", "- text/html"}))
	assert.Equal(t, []string{"text/plain", "text/html"}, got)
}

func TestMultilineYAMLListParserNonListDropsValues(t *testing.T) {
	// Q4 strict-list contract: a scalar body emits a warning and
	// produces no values (setter called with nothing? no — setter
	// is NOT called on the non-list path, so `got` stays at its
	// zero value).
	t.Parallel()

	var called bool
	cp := NewConsumesDropEmptyParser(func(v []string) { called = true; _ = v })
	require.NoError(t, cp.Parse([]string{"application/json"})) // bare form, not a list
	assert.FalseT(t, called)
}
