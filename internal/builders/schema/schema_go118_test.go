// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

func getGo118ClassificationModel(sctx *scanner.ScanCtx, nm string) *scanner.EntityDecl {
	decl, ok := sctx.FindDecl(fixturesModule+"/goparsing/go118", nm)
	if !ok {
		return nil
	}
	return decl
}

func TestGo118SwaggerTypeNamed(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)
	decl := getGo118ClassificationModel(sctx, "NamedWithType")
	require.NotNil(t, decl)
	prs := NewBuilder(sctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))
	schema := models[scantest.ResolveTestKey(t, models, "namedWithType")]

	assert.TrueT(t, schema.Type.Contains("object"))
	scantest.AssertProperty(t, &schema, "object", "some_map", "", "SomeMap")
	assert.Equal(t, "NamedWithType", schema.Extensions["x-go-name"])
}

func TestGo118AliasedModels(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)

	names := []string{
		"SomeObject",
	}

	defs := make(map[string]oaispec.Schema)
	for _, nm := range names {
		decl := getGo118ClassificationModel(sctx, nm)
		require.NotNil(t, decl)

		prs := NewBuilder(sctx, decl)
		require.NoError(t, prs.Build(WithDefinitions(defs)))
	}

	for k := range defs {
		for i, b := range names {
			// defs is keyed by the fully-qualified identity; match on the leaf via ResolveTestKey rather
			// than the bare name.
			if scantest.ResolveTestKey(t, defs, b) == k {
				// remove the entry from the collection
				names = append(names[:i], names[i+1:]...)
			}
		}
	}
	if assert.Empty(t, names) {
		// SomeObject refines an untyped map → object with additionalProperties.
		assertMapDefinition(t, defs, scantest.ResolveTestKey(t, defs, "SomeObject"), "object", "", "")
		obj := defs[scantest.ResolveTestKey(t, defs, "SomeObject")]
		assert.EqualT(t, "SomeObject is a type that refines an untyped map", obj.Description)
	}
}

func TestGo118InterfaceField(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)
	decl := getGo118ClassificationModel(sctx, "Interfaced")
	require.NotNil(t, decl)
	prs := NewBuilder(sctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "Interfaced")]
	assert.TrueT(t, schema.Type.Contains("object"))
	// custom_data is an interface field → no type, just the x-go-name.
	scantest.AssertProperty(t, &schema, "", "custom_data", "", "CustomData")
	assert.EqualT(t, "An Interfaced struct contains objects with interface definitions", schema.Description)
}

func TestGo118_Issue2809(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)
	decl := getGo118ClassificationModel(sctx, "transportErr")
	require.NotNil(t, decl)
	prs := NewBuilder(sctx, decl)
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(WithDefinitions(models)))

	schema := models[scantest.ResolveTestKey(t, models, "transportErr")]
	assert.TrueT(t, schema.Type.Contains("object"))
	assert.Equal(t, []string{"message"}, schema.Required)
	// data is an interface field (no type); message is a plain string.
	scantest.AssertProperty(t, &schema, "", "data", "", "Data")
	scantest.AssertProperty(t, &schema, "string", "message", "", "Message")
}
