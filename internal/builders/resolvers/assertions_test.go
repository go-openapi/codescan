// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// TestIsStdUUID covers the go1.27 stdlib uuid.UUID predicate.
//
// Deliberately synthesizes the *types.TypeName instead of scanning a fixture, so the predicate stays
// covered on every supported toolchain — including the ones where `import "uuid"` does not compile.
// The end-to-end wiring is witnessed by the go1.27-tagged integration test instead.
func TestIsStdUUID(t *testing.T) {
	// stdlib packages have a bare import path; uuid.UUID is [16]byte.
	stdUUID := func() *types.TypeName {
		pkg := types.NewPackage("uuid", "uuid")
		return types.NewTypeName(token.NoPos, pkg, "UUID", types.NewArray(types.Typ[types.Byte], 16))
	}

	named := func(path, name, typeName string) *types.TypeName {
		pkg := types.NewPackage(path, name)
		return types.NewTypeName(token.NoPos, pkg, typeName, types.NewArray(types.Typ[types.Byte], 16))
	}

	t.Run("the go1.27 stdlib uuid.UUID is recognized", func(t *testing.T) {
		assert.TrueT(t, IsStdUUID(stdUUID()))
	})

	t.Run("third-party UUID types are NOT recognized by identity", func(t *testing.T) {
		// These stay on the fuzzy name heuristic in the schema builder; identity must not claim them.
		assert.FalseT(t, IsStdUUID(named("github.com/google/uuid", "uuid", "UUID")))
		assert.FalseT(t, IsStdUUID(named("github.com/gofrs/uuid/v5", "uuid", "UUID")))
		assert.FalseT(t, IsStdUUID(named("github.com/go-openapi/strfmt", "strfmt", "UUID")))
	})

	t.Run("a user package merely named uuid is NOT recognized", func(t *testing.T) {
		// Package *name* is uuid, but the import path carries the module prefix — only the stdlib
		// gets a bare path. This is why the predicate tests Path(), not Name().
		assert.FalseT(t, IsStdUUID(named("example.com/mymod/internal/uuid", "uuid", "UUID")))
	})

	t.Run("another type in the stdlib uuid package is NOT recognized", func(t *testing.T) {
		assert.FalseT(t, IsStdUUID(named("uuid", "uuid", "Version")))
	})

	t.Run("the match is case-sensitive", func(t *testing.T) {
		assert.FalseT(t, IsStdUUID(named("uuid", "uuid", "Uuid")))
	})

	t.Run("a builtin (no package) does not panic", func(t *testing.T) {
		builtin := types.NewTypeName(token.NoPos, nil, "error", nil)
		assert.FalseT(t, IsStdUUID(builtin))
	})
}
