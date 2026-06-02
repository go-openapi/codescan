// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"testing"

	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

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
// Accepts both shapes: a bare $ref schema, and the P7/S7 allOf
// compound where the $ref rides arm[0] (with description and
// optional override siblings on the parent / arm[1]).
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
