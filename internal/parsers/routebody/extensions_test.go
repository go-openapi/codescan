// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package routebody

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func TestSetOpExtensions_Matches(t *testing.T) {
	t.Parallel()

	se := NewSetExtensions(nil, false)
	assert.TrueT(t, se.Matches("extensions:"))
	assert.TrueT(t, se.Matches("Extensions:"))
	assert.FalseT(t, se.Matches("something else"))
}

func TestSetOpExtensions_Parse(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		var called bool
		se := NewSetExtensions(func(_ *oaispec.Extensions) { called = true }, false)
		require.NoError(t, se.Parse(nil))
		assert.FalseT(t, called)
		require.NoError(t, se.Parse([]string{}))
		require.NoError(t, se.Parse([]string{""}))
	})

	t.Run("simple key-value", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-custom-value: hello",
		}
		require.NoError(t, se.Parse(lines))
		val, ok := got.GetString("x-custom-value")
		require.TrueT(t, ok)
		assert.EqualT(t, "hello", val)
	})

	t.Run("array extension", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-tags:",
			"  value1",
			"  value2",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)
		val, ok := got["x-tags"]
		require.TrueT(t, ok)
		arr, ok := val.([]string)
		require.TrueT(t, ok)
		assert.Equal(t, []string{"value1", "value2"}, arr)
	})

	t.Run("object extension", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-meta:",
			"  name: obj",
			"  value: field",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)
		val, ok := got["x-meta"]
		require.TrueT(t, ok)
		obj, ok := val.(map[string]any)
		require.TrueT(t, ok)
		assert.EqualT(t, "obj", obj["name"])
		assert.EqualT(t, "field", obj["value"])
	})

	t.Run("multiple extensions", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-first: one",
			"x-second: two",
		}
		require.NoError(t, se.Parse(lines))
		v1, ok := got.GetString("x-first")
		require.TrueT(t, ok)
		assert.EqualT(t, "one", v1)
		v2, ok := got.GetString("x-second")
		require.TrueT(t, ok)
		assert.EqualT(t, "two", v2)
	})

	t.Run("nested object with key-value", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-nested:",
			"  outer:",
			"    inner-key: inner-value",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)
		val, ok := got["x-nested"]
		require.TrueT(t, ok)
		obj, ok := val.(map[string]any)
		require.TrueT(t, ok)
		outer, ok := obj["outer"].(map[string]any)
		require.TrueT(t, ok)
		assert.EqualT(t, "inner-value", outer["inner-key"])
	})

	t.Run("object then back to simple extension", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-obj:",
			"  key1: val1",
			"  key2: val2",
			"x-simple: value",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)

		obj, ok := got["x-obj"]
		require.TrueT(t, ok)
		m, ok := obj.(map[string]any)
		require.TrueT(t, ok)
		assert.EqualT(t, "val1", m["key1"])
		assert.EqualT(t, "val2", m["key2"])

		simple, ok := got.GetString("x-simple")
		require.TrueT(t, ok)
		assert.EqualT(t, "value", simple)
	})

	t.Run("nested object with sub-list", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-deep:",
			"  sub-list:",
			"    item1",
			"    item2",
			"  sub-key: sub-value",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)
		val, ok := got["x-deep"]
		require.TrueT(t, ok)
		obj, ok := val.(map[string]any)
		require.TrueT(t, ok)
		assert.NotNil(t, obj["sub-list"])
		assert.EqualT(t, "sub-value", obj["sub-key"])
	})

	t.Run("array extension followed by object extension", func(t *testing.T) {
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-list:",
			"  alpha",
			"  beta",
			"x-map:",
			"  a: 1",
			"  b: 2",
		}
		require.NoError(t, se.Parse(lines))
		require.NotNil(t, got)

		list, ok := got["x-list"]
		require.TrueT(t, ok)
		arr, ok := list.([]string)
		require.TrueT(t, ok)
		assert.Equal(t, []string{"alpha", "beta"}, arr)

		m, ok := got["x-map"]
		require.TrueT(t, ok)
		obj, ok := m.(map[string]any)
		require.TrueT(t, ok)
		assert.EqualT(t, "1", obj["a"])
	})
}

// TestSetOpExtensions_Parse_Malformed exercises the defensive guards in
// buildExtensionObjects against malformed extension blocks. Each case
// covers a branch that would otherwise be silently skipped with no test
// witness — losing data without warning would be a nasty silent-bug class.
// The contract these tests pin: malformed lines never panic, never
// corrupt prior extensions, and are simply dropped from the output.
func TestSetOpExtensions_Parse_Malformed(t *testing.T) {
	t.Parallel()

	t.Run("line with empty key is skipped", func(t *testing.T) {
		// Covers the `if key == "" { return }` guard in buildExtensionObjects:
		// a line whose colon-prefix produces an empty trimmed key (e.g. leading
		// colon). Must not panic; nothing should land in the output.
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{":just-a-value"}
		require.NoError(t, se.Parse(lines))
		assert.Empty(t, got)
	})

	t.Run("list item at top level with no prior extension is dropped", func(t *testing.T) {
		// Covers `if stack == nil || len(*stack) == 0 { return }` in the
		// `len(kv) <= 1` branch — a bare word (no colon) with no preceding
		// x- extension to attach it to.
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{"orphan-item"}
		require.NoError(t, se.Parse(lines))
		assert.Empty(t, got)
	})

	t.Run("non-x key:value at top level with no prior extension is dropped", func(t *testing.T) {
		// Covers the second `if stack == nil || len(*stack) == 0 { return }`
		// guard further down: a plain key:value line whose key isn't an x-
		// extension and there's no open extension to nest under.
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{"plain-key: plain-value"}
		require.NoError(t, se.Parse(lines))
		assert.Empty(t, got)
	})

	t.Run("valid extension followed by malformed lines keeps valid output", func(t *testing.T) {
		// Composition check: a good x- extension followed by each malformed
		// shape. The good extension must survive; the junk lines must be
		// dropped without disturbing it.
		var got oaispec.Extensions
		se := NewSetExtensions(func(ext *oaispec.Extensions) { got = *ext }, false)

		lines := []string{
			"x-good: keeper",
			":empty-key",
			"orphan-item",
			"plain-key: plain-value",
		}
		require.NoError(t, se.Parse(lines))

		val, ok := got.GetString("x-good")
		require.TrueT(t, ok)
		assert.EqualT(t, "keeper", val)
		_, hasPlain := got["plain-key"]
		assert.FalseT(t, hasPlain, "non-x- key must not leak into output")
	})
}
