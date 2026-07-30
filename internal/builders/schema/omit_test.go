// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// TestParseOmitTargets locks the argument grammar: the lexer hands over the whole remainder
// verbatim (the list may carry spaces after commas), so the splitting contract lives here.
func TestParseOmitTargets(t *testing.T) {
	t.Run("a single target", func(t *testing.T) {
		assert.Equal(t, []string{"ID"}, parseOmitTargets("ID"))
	})

	t.Run("a comma-separated list, with or without spaces", func(t *testing.T) {
		assert.Equal(t, []string{"ID", "Created"}, parseOmitTargets("ID,Created"))
		assert.Equal(t, []string{"ID", "Created"}, parseOmitTargets("ID, Created"))
		assert.Equal(t, []string{"ID", "Created"}, parseOmitTargets("  ID ,  Created  "))
	})

	t.Run("qualified paths are kept intact", func(t *testing.T) {
		assert.Equal(t, []string{"Base.ID", "Nested.Inner.Deep"},
			parseOmitTargets("Base.ID, Nested.Inner.Deep"))
	})

	t.Run("empty and blank arguments yield nothing", func(t *testing.T) {
		assert.Empty(t, parseOmitTargets(""))
		assert.Empty(t, parseOmitTargets("   "))
		assert.Empty(t, parseOmitTargets(",,"))
	})
}

// TestDeclOmitsFor locks how declaration-level targets are dispatched to one embed: a dotted path
// applies only to the embed its head names (with the head consumed), while a bare name is offered to
// every embed and resolved against the promoted set there.
func TestDeclOmitsFor(t *testing.T) {
	base := fakeEmbed(t, "Base")
	other := fakeEmbed(t, "Other")

	t.Run("a dotted path applies only to the embed it names", func(t *testing.T) {
		assert.Equal(t, []string{"ID"}, declOmitsFor(base, []string{"Base.ID"}))
		assert.Empty(t, declOmitsFor(other, []string{"Base.ID"}))
	})

	t.Run("the head is consumed, so deeper paths keep their tail", func(t *testing.T) {
		assert.Equal(t, []string{"Inner.Deep"}, declOmitsFor(base, []string{"Base.Inner.Deep"}))
	})

	t.Run("a bare name is offered to every embed", func(t *testing.T) {
		assert.Equal(t, []string{"Created"}, declOmitsFor(base, []string{"Created"}))
		assert.Equal(t, []string{"Created"}, declOmitsFor(other, []string{"Created"}))
	})

	t.Run("no declaration-level targets is not a match", func(t *testing.T) {
		assert.Empty(t, declOmitsFor(base, nil))
	})
}

// fakeEmbed builds a minimal embedded field object named name, for the dispatch tests above (which
// exercise name matching only, never type resolution).
func fakeEmbed(t *testing.T, name string) *types.Var {
	t.Helper()
	pkg := types.NewPackage("example.com/fake", "fake")
	obj := types.NewTypeName(token.NoPos, pkg, name, nil)
	named := types.NewNamed(obj, types.NewStruct(nil, nil), nil)

	return types.NewField(token.NoPos, pkg, name, named, true)
}
