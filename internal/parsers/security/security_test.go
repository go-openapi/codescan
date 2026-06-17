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
		assert.Nil(t, Parse("   \n  "))
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

	// --- canonical YAML: sequence of requirement objects -------------------

	t.Run("sequence item: bare name has empty scopes", func(t *testing.T) {
		got := Parse("- api_key:")
		require.Len(t, got, 1)
		require.Contains(t, got[0], "api_key")
		assert.Empty(t, got[0]["api_key"])
	})

	t.Run("sequence item: empty flow scopes", func(t *testing.T) {
		got := Parse("- auth0: []")
		require.Len(t, got, 1)
		require.Contains(t, got[0], "auth0")
		assert.Empty(t, got[0]["auth0"])
	})

	t.Run("sequence item: flow-style scope list", func(t *testing.T) {
		got := Parse("- oauth: [read, write]")
		require.Len(t, got, 1)
		assert.Equal(t, []string{"read", "write"}, got[0]["oauth"])
	})

	t.Run("sequence item: block-style scope list", func(t *testing.T) {
		// the other idiomatic YAML scope form — was silently broken before the
		// YAML-decode rewrite.
		got := Parse("- oauth:\n    - read\n    - write")
		require.Len(t, got, 1)
		assert.Equal(t, []string{"read", "write"}, got[0]["oauth"])
	})

	t.Run("multiple sequence items are separate (OR) requirements", func(t *testing.T) {
		got := Parse("- api_key: []\n- oauth: [read]")
		require.Len(t, got, 2)
		assert.Empty(t, got[0]["api_key"])
		assert.Equal(t, []string{"read"}, got[1]["oauth"])
	})

	// go-swagger#2294: two keys under ONE sequence item mean AND — a single
	// requirement object carrying both schemes.
	t.Run("AND form: two keys in one sequence item", func(t *testing.T) {
		got := Parse("- x_client_id: []\n  access_token: []")
		require.Len(t, got, 1, "AND is a single requirement, not two")
		require.Contains(t, got[0], "x_client_id")
		require.Contains(t, got[0], "access_token")
		assert.Empty(t, got[0]["x_client_id"])
		assert.Empty(t, got[0]["access_token"])
	})

	t.Run("AND form coexists with a following OR item", func(t *testing.T) {
		// (a AND b) OR c
		got := Parse("- a: []\n  b: []\n- c: [read]")
		require.Len(t, got, 2)
		assert.Contains(t, got[0], "a")
		assert.Contains(t, got[0], "b")
		assert.Equal(t, []string{"read"}, got[1]["c"])
	})

	// --- bare-name shorthand (go-swagger convenience) ----------------------

	t.Run("bare-name sequence shorthand", func(t *testing.T) {
		got := Parse("- petstore_auth\n- api_key")
		require.Len(t, got, 2)
		require.Contains(t, got[0], "petstore_auth")
		require.Contains(t, got[1], "api_key")
		assert.Empty(t, got[0]["petstore_auth"])
		assert.Empty(t, got[1]["api_key"])
	})

	t.Run("bare-name flow shorthand", func(t *testing.T) {
		got := Parse("[petstore_auth, api_key]")
		require.Len(t, got, 2)
		require.Contains(t, got[0], "petstore_auth")
		require.Contains(t, got[1], "api_key")
	})

	// --- legacy top-level mapping (preserved, OR) --------------------------

	t.Run("legacy flat mapping: one requirement per key (OR)", func(t *testing.T) {
		got := Parse("api_key:\noauth: read, write")
		require.Len(t, got, 2, "bare top-level mapping keys are OR-combined (legacy)")
		assert.Empty(t, got[0]["api_key"])
		assert.Equal(t, []string{"read", "write"}, got[1]["oauth"], "scalar scope value is comma-split")
	})

	t.Run("legacy scope with embedded colon survives", func(t *testing.T) {
		got := Parse("oauth: orders:read, https://example.com/scope")
		require.Len(t, got, 1)
		assert.Equal(t, []string{"orders:read", "https://example.com/scope"}, got[0]["oauth"])
	})

	t.Run("malformed YAML yields nil (no panic)", func(t *testing.T) {
		assert.Nil(t, Parse("\t- : : ::\n  bad: ["))
	})
}
