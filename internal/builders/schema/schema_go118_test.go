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
	prs := &Builder{
		ctx:  sctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))
	schema := models["namedWithType"]

	scantest.AssertProperty(t, &schema, "object", "some_map", "", "SomeMap")

	scantest.CompareOrDumpJSON(t, models, "go118_schema_NamedWithType.json")
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

		prs := &Builder{
			decl: decl,
			ctx:  sctx,
		}
		require.NoError(t, prs.Build(defs))
	}

	for k := range defs {
		for i, b := range names {
			if b == k {
				// remove the entry from the collection
				names = append(names[:i], names[i+1:]...)
			}
		}
	}
	if assert.Empty(t, names) {
		// map types
		assertMapDefinition(t, defs, "SomeObject", "object", "", "")
	}

	scantest.CompareOrDumpJSON(t, defs, "go118_schema_aliased.json")
}

func TestGo118InterfaceField(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)
	decl := getGo118ClassificationModel(sctx, "Interfaced")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  sctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["Interfaced"]
	scantest.AssertProperty(t, &schema, "", "custom_data", "", "CustomData")

	scantest.CompareOrDumpJSON(t, models, "go118_schema_Interfaced.json")
}

func TestGo118_Issue2809(t *testing.T) {
	sctx := scantest.LoadGo118ClassificationPkgsCtx(t)
	decl := getGo118ClassificationModel(sctx, "transportErr")
	require.NotNil(t, decl)
	prs := &Builder{
		ctx:  sctx,
		decl: decl,
	}
	models := make(map[string]oaispec.Schema)
	require.NoError(t, prs.Build(models))

	schema := models["transportErr"]
	scantest.AssertProperty(t, &schema, "", "data", "", "Data")

	scantest.CompareOrDumpJSON(t, models, "go118_schema_transportErr.json")
}
