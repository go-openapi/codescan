// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"strings"
	"testing"

	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// ResolveTestKey returns the fully-qualified definitions key whose leaf (the segment after the last
// '/') equals short, so a sub-builder unit test can keep indexing the definitions map by the short
// Go name:
//
//	schema := models[scantest.ResolveTestKey(t, models, "NoModel")]
//
// and can build the expected fully-qualified $ref the same way:
//
//	scantest.AssertRef(t, &schema, "pet", "Pet",
//	    "#/definitions/"+scantest.ResolveTestKey(t, models, "pet"))
//
// The schema builder keys the definitions map by the fully-qualified identity ("<pkgpath>/<name>",
// see scanner.EntityDecl.DefKey); the spec orchestrator's reduce stage later shortens unique leaves
// back to the bare name, but sub-builder unit tests run WITHOUT that stage.
//
// When no definition carries the leaf, short is returned unchanged so absence checks (`_, ok :=
// models[ResolveTestKey(...)]; assert.False(ok)`) still observe a miss.
// It fails only on a genuinely ambiguous match, which no current unit fixture produces.
// §12.1.
func ResolveTestKey(t *testing.T, defs map[string]oaispec.Schema, short string) string {
	t.Helper()

	var (
		key  string
		hits int
	)
	for k := range defs {
		seg := k
		if i := strings.LastIndex(k, "/"); i >= 0 {
			seg = k[i+1:]
		}
		if seg == short {
			key = k
			hits++
		}
	}
	if hits > 1 {
		require.FailNowf(t, "ambiguous test ref",
			"%d definitions share leaf %q: %v", hits, short, keysOf(defs))
	}
	if hits == 0 {
		return short
	}
	return key
}

func keysOf(defs map[string]oaispec.Schema) []string {
	keys := make([]string, 0, len(defs))
	for k := range defs {
		keys = append(keys, k)
	}
	return keys
}

func AssertProperty(t *testing.T, schema *oaispec.Schema, typeName, jsonName, format, goName string) {
	t.Helper()

	if typeName == "" {
		assert.Empty(t, schema.Properties[jsonName].Type)
	} else if assert.NotEmpty(t, schema.Properties[jsonName].Type) {
		assert.EqualT(t, typeName, schema.Properties[jsonName].Type[0])
	}

	if goName == "" {
		assert.Nil(t, schema.Properties[jsonName].Extensions["x-go-name"])
	} else {
		assert.Equal(t, goName, schema.Properties[jsonName].Extensions["x-go-name"])
	}

	assert.EqualT(t, format, schema.Properties[jsonName].Format)
}

func AssertArrayProperty(t *testing.T, schema *oaispec.Schema, typeName, jsonName, format, goName string) {
	t.Helper()

	prop := schema.Properties[jsonName]
	assert.NotEmpty(t, prop.Type)
	assert.TrueT(t, prop.Type.Contains("array"))
	require.NotNil(t, prop.Items)

	if typeName != "" {
		require.NotNil(t, prop.Items.Schema)
		require.NotEmpty(t, prop.Items.Schema.Type)
		assert.EqualT(t, typeName, prop.Items.Schema.Type[0])
	}

	assert.Equal(t, goName, prop.Extensions["x-go-name"])
	assert.EqualT(t, format, prop.Items.Schema.Format)
}

func AssertArrayRef(t *testing.T, schema *oaispec.Schema, jsonName, goName, fragment string) {
	t.Helper()

	AssertArrayProperty(t, schema, "", jsonName, "", goName)
	prop := schema.Properties[jsonName]
	items := prop.Items
	require.NotNil(t, items)

	psch := items.Schema
	assert.EqualT(t, fragment, psch.Ref.String())
}

// AssertRef checks that the named property is a $ref to fragment.
//
// Accepts both shapes: a bare $ref schema, and the P7/S7 allOf compound where the $ref rides arm[0]
// (with description and optional override siblings on the parent / arm[1]).
func AssertRef(t *testing.T, schema *oaispec.Schema, jsonName, _, fragment string) {
	t.Helper()

	psch := schema.Properties[jsonName]
	assert.Empty(t, psch.Type)
	if psch.Ref.String() != "" {
		assert.EqualT(t, fragment, psch.Ref.String())
		return
	}
	require.NotEmpty(t, psch.AllOf, "property %q has neither $ref nor allOf", jsonName)
	assert.EqualT(t, fragment, psch.AllOf[0].Ref.String())
}
