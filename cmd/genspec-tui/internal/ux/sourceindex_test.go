// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"go/token"
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func srcPos(file string, line int) token.Position {
	return token.Position{Filename: file, Line: line}
}

func TestSourceIndex_PositionFor(t *testing.T) {
	idx := BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: srcPos("user.go", 10)},
		{Pointer: "/definitions/User/properties/email", Pos: srcPos("user.go", 14)},
		{Pointer: "/paths/~1pets/get", Pos: srcPos("api.go", 3)},
	})

	t.Run("exact hit", func(t *testing.T) {
		p, ok := idx.PositionFor("/definitions/User/properties/email")
		require.True(t, ok)
		assert.Equal(t, "user.go", p.Filename)
		assert.Equal(t, 14, p.Line)
	})

	t.Run("nearest anchored ancestor", func(t *testing.T) {
		// A finer node with no anchor of its own resolves to the closest
		// anchored ancestor — here the property, then the definition.
		p, ok := idx.PositionFor("/definitions/User/properties/email/format")
		require.True(t, ok)
		assert.Equal(t, 14, p.Line, "should resolve up to the property anchor")

		p, ok = idx.PositionFor("/definitions/User/required/0")
		require.True(t, ok)
		assert.Equal(t, 10, p.Line, "should resolve up to the definition anchor")
	})

	t.Run("no anchor at or above", func(t *testing.T) {
		_, ok := idx.PositionFor("/swagger")
		assert.False(t, ok)
		_, ok = idx.PositionFor("")
		assert.False(t, ok)
	})

	t.Run("nil index", func(t *testing.T) {
		var nilIdx *SourceIndex
		_, ok := nilIdx.PositionFor("/definitions/User")
		assert.False(t, ok)
		assert.Equal(t, 0, nilIdx.Len())
	})
}

func TestSourceIndex_PointerAt(t *testing.T) {
	idx := BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User", Pos: srcPos("user.go", 10)},
		{Pointer: "/definitions/User/properties/email", Pos: srcPos("user.go", 14)},
		{Pointer: "/definitions/User/properties/name", Pos: srcPos("user.go", 12)},
		{Pointer: "/paths/~1pets/get", Pos: srcPos("api.go", 3)},
	})

	t.Run("nearest enclosing anchor", func(t *testing.T) {
		// A line inside the email field (anchored at 14) but below it resolves
		// to that field; a line between name (12) and email (14) resolves to name.
		ptr, ok := idx.PointerAt("user.go", 15)
		require.True(t, ok)
		assert.Equal(t, "/definitions/User/properties/email", ptr)

		ptr, ok = idx.PointerAt("user.go", 13)
		require.True(t, ok)
		assert.Equal(t, "/definitions/User/properties/name", ptr)

		ptr, ok = idx.PointerAt("user.go", 10)
		require.True(t, ok)
		assert.Equal(t, "/definitions/User", ptr, "exact line lands on the definition")
	})

	t.Run("line above the first anchor", func(t *testing.T) {
		_, ok := idx.PointerAt("user.go", 1)
		assert.False(t, ok)
	})

	t.Run("unknown file", func(t *testing.T) {
		_, ok := idx.PointerAt("other.go", 5)
		assert.False(t, ok)
	})

	t.Run("per-file isolation", func(t *testing.T) {
		ptr, ok := idx.PointerAt("api.go", 9)
		require.True(t, ok)
		assert.Equal(t, "/paths/~1pets/get", ptr)
	})
}

func TestSourceIndex_FirstAnchor(t *testing.T) {
	idx := BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/User/properties/email", Pos: srcPos("user.go", 14)},
		{Pointer: "/definitions/User", Pos: srcPos("user.go", 10)},
		{Pointer: "/paths/~1pets/get", Pos: srcPos("api.go", 3)},
	})

	ptr, ok := idx.FirstAnchor("user.go")
	require.True(t, ok)
	assert.Equal(t, "/definitions/User", ptr, "earliest line in the file wins")

	_, ok = idx.FirstAnchor("missing.go")
	assert.False(t, ok)

	var nilIdx *SourceIndex
	_, ok = nilIdx.FirstAnchor("user.go")
	assert.False(t, ok)
}

func TestSourceIndex_LastWins(t *testing.T) {
	// A pointer recorded twice (node rewritten during the build) keeps the last
	// position — upsert semantics.
	idx := BuildSourceIndex([]scanner.Provenance{
		{Pointer: "/definitions/X", Pos: srcPos("a.go", 1)},
		{Pointer: "/definitions/X", Pos: srcPos("a.go", 7)},
	})
	p, ok := idx.PositionFor("/definitions/X")
	require.True(t, ok)
	assert.Equal(t, 7, p.Line)
}
