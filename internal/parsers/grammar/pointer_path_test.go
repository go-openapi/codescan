// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package grammar

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func kw(t *testing.T, name string) Keyword {
	t.Helper()
	k, ok := Lookup(name)
	require.True(t, ok, "keyword %q should be in the table", name)
	return k
}

func TestPointerPath(t *testing.T) {
	t.Run("validation segment equals the keyword name", func(t *testing.T) {
		segs, ok := PointerPath(kw(t, KwMaximum), CtxSchema)
		require.True(t, ok)
		assert.Equal(t, []string{"maximum"}, segs)
	})

	t.Run("unique renames to uniqueItems", func(t *testing.T) {
		segs, ok := PointerPath(kw(t, KwUnique), CtxSchema)
		require.True(t, ok)
		assert.Equal(t, []string{"uniqueItems"}, segs)
	})

	t.Run("meta Info.* nests under /info, root fields do not", func(t *testing.T) {
		segs, ok := PointerPath(kw(t, KwVersion), CtxMeta)
		require.True(t, ok)
		assert.Equal(t, []string{"info", "version"}, segs)

		segs, ok = PointerPath(kw(t, KwHost), CtxMeta)
		require.True(t, ok)
		assert.Equal(t, []string{"host"}, segs)
	})

	t.Run("shared keyword resolves in every legal context", func(t *testing.T) {
		for _, ctx := range []KeywordContext{CtxMeta, CtxRoute, CtxOperation} {
			segs, ok := PointerPath(kw(t, KwConsumes), ctx)
			require.True(t, ok, "consumes should anchor in %s", ctx)
			assert.Equal(t, []string{"consumes"}, segs)
		}
	})

	t.Run("deprecated anchors on operations/routes but NOT schema", func(t *testing.T) {
		// Lexically legal on a schema (so the lexer accepts it), but renders no schema node in OAS2 —
		// must not be anchored there.
		require.Contains(t, kw(t, KwDeprecated).Contexts, CtxSchema)
		_, ok := PointerPath(kw(t, KwDeprecated), CtxSchema)
		assert.False(t, ok, "deprecated must not anchor in a schema")

		segs, ok := PointerPath(kw(t, KwDeprecated), CtxRoute)
		require.True(t, ok)
		assert.Equal(t, []string{"deprecated"}, segs)
	})

	t.Run("non-node keywords are absent", func(t *testing.T) {
		for _, name := range []string{KwRequired, KwIn, KwCollectionFormat, KwSecurity} {
			_, ok := PointerPath(kw(t, name), CtxSchema)
			assert.False(t, ok, "%q should not anchor (no addressable node)", name)
		}
	})

	t.Run("context gate: a keyword not legal in ctx does not anchor", func(t *testing.T) {
		// version is meta-only; it must not anchor in a schema even though it has a pointer path.
		_, ok := PointerPath(kw(t, KwVersion), CtxSchema)
		assert.False(t, ok)
	})
}
