// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package security

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestParse(t *testing.T) {
	t.Run("empty body yields nil", func(t *testing.T) {
		assert.Nil(t, Parse(""))
	})

	t.Run("flat form: name with scopes", func(t *testing.T) {
		got := Parse("oauth: read, write")
		require.Len(t, got, 1)
		assert.Equal(t, []string{"read", "write"}, got[0]["oauth"])
	})

	// go-swagger#2479: an explicit empty sequence is the OAS 2.0 opt-out idiom.
	// It must be distinguishable from an absent block (nil) so the spec marshals
	// `"security": []` rather than omitting the key.
	t.Run("explicit empty list opts out (non-nil empty)", func(t *testing.T) {
		got := Parse("[]")
		require.NotNil(t, got, "`[]` must yield a non-nil empty list, not nil")
		assert.Empty(t, got)
	})

	t.Run("explicit empty list tolerates surrounding space", func(t *testing.T) {
		got := Parse("  [] ")
		require.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("flat form: bare name has empty scopes", func(t *testing.T) {
		got := Parse("api_key:")
		require.Len(t, got, 1)
		assert.Empty(t, got[0]["api_key"])
	})

	// go-swagger#2403: the YAML sequence marker must not be glued onto the key.
	t.Run("sequence form: bare name", func(t *testing.T) {
		got := Parse("- api_key:")
		require.Len(t, got, 1)
		_, badKey := got[0]["- api_key"]
		assert.False(t, badKey, "sequence marker must be stripped")
		require.Contains(t, got[0], "api_key")
		assert.Empty(t, got[0]["api_key"])
	})

	t.Run("sequence form: empty inline list", func(t *testing.T) {
		got := Parse("- auth0: []")
		require.Len(t, got, 1)
		require.Contains(t, got[0], "auth0")
		assert.Empty(t, got[0]["auth0"], "`[]` must be empty scopes, not a literal `[]` scope")
	})

	t.Run("sequence form: populated inline list", func(t *testing.T) {
		got := Parse("- oauth: [read, write]")
		require.Len(t, got, 1)
		assert.Equal(t, []string{"read", "write"}, got[0]["oauth"])
	})

	t.Run("multiple requirements across lines", func(t *testing.T) {
		got := Parse("- api_key: []\n- oauth: [read]")
		require.Len(t, got, 2)
		assert.Empty(t, got[0]["api_key"])
		assert.Equal(t, []string{"read"}, got[1]["oauth"])
	})

	t.Run("scheme name beginning with dash is preserved", func(t *testing.T) {
		got := Parse("-weird:")
		require.Len(t, got, 1)
		require.Contains(t, got[0], "-weird", "only `- ` (dash + space) is a sequence marker")
	})
}
