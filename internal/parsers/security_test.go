// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package parsers

import (
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestSetSchemes(t *testing.T) {
	t.Parallel()

	t.Run("single scheme", func(t *testing.T) {
		var got []string
		ss := NewSetSchemes(func(v []string) { got = v })
		assert.TrueT(t, ss.Matches("schemes: http"))
		require.NoError(t, ss.Parse([]string{"schemes: http"}))
		assert.Equal(t, []string{"http"}, got)
	})

	t.Run("multiple schemes", func(t *testing.T) {
		var got []string
		ss := NewSetSchemes(func(v []string) { got = v })
		require.NoError(t, ss.Parse([]string{"schemes: http, https"}))
		assert.Equal(t, []string{"http", "https"}, got)
	})

	t.Run("wss", func(t *testing.T) {
		var got []string
		ss := NewSetSchemes(func(v []string) { got = v })
		require.NoError(t, ss.Parse([]string{"Schemes: ws, wss"}))
		assert.Equal(t, []string{"ws", "wss"}, got)
	})

	t.Run("empty", func(t *testing.T) {
		var got []string
		ss := NewSetSchemes(func(v []string) { got = v })
		require.NoError(t, ss.Parse(nil))
		require.NoError(t, ss.Parse([]string{}))
		require.NoError(t, ss.Parse([]string{""}))
		assert.Nil(t, got)
	})

	t.Run("no match", func(t *testing.T) {
		ss := NewSetSchemes(nil)
		assert.FalseT(t, ss.Matches("something else"))
	})
}

func TestSetSecurity(t *testing.T) {
	t.Parallel()

	t.Run("with scopes", func(t *testing.T) {
		var got []map[string][]string
		ss := NewSetSecurityScheme(func(v []map[string][]string) { got = v })
		assert.TrueT(t, ss.Matches("security:"))
		require.NoError(t, ss.Parse([]string{
			"api_key:",
			"oauth2: read:pets, write:pets",
		}))
		require.Len(t, got, 2)
		assert.Equal(t, map[string][]string{"api_key": {}}, got[0])
		assert.Equal(t, map[string][]string{"oauth2": {"read:pets", "write:pets"}}, got[1])
	})

	t.Run("empty", func(t *testing.T) {
		var got []map[string][]string
		ss := NewSetSecurityScheme(func(v []map[string][]string) { got = v })
		require.NoError(t, ss.Parse(nil))
		require.NoError(t, ss.Parse([]string{}))
		require.NoError(t, ss.Parse([]string{""}))
		assert.Nil(t, got)
	})

	t.Run("no colon in line", func(t *testing.T) {
		var got []map[string][]string
		ss := NewSetSecurityScheme(func(v []map[string][]string) { got = v })
		require.NoError(t, ss.Parse([]string{"no-colon-here"}))
		assert.Nil(t, got) // line without colon is skipped
	})
}
