// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug1499 locks the fix for go-swagger issue #1499 ("not possible
// to override parameter schema type").
//
// A `swagger:type` override on a swagger:parameters field whose Go type is a
// struct must make the parameter that simple type (here `string`). The schema
// (model) builder already honours this (#2419/#2184), but the PARAMETERS builder
// used to ignore it — the param came out typeless, which is invalid Swagger 2.0.
//
// The `[]`-prefixed form collapses a Go struct slice to a simple
// array-of-scalar query parameter, mirroring the schema builder's array layers.
func TestCoverage_Bug1499(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1499/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	params := doc.Paths.Paths["/foo"].Get.Parameters
	byName := make(map[string]spec.Parameter, len(params))
	for _, p := range params {
		byName[p.Name] = p
	}

	// Scalar override: the struct-typed field becomes a plain string query param.
	bar := byName["bar"]
	assert.Equal(t, "query", bar.In)
	assert.Equal(t, "string", bar.Type, "swagger:type must override the param field type (go-swagger#1499)")
	assert.Empty(t, bar.Ref.String(), "the override is inline, never a $ref")

	// Array override: `[]string` collapses []Bar to array-of-string.
	bars := byName["bars"]
	assert.Equal(t, "query", bars.In)
	assert.Equal(t, "array", bars.Type, "[]-prefixed swagger:type yields an array param (go-swagger#1499)")
	require.NotNil(t, bars.Items, "array param must carry an items schema")
	assert.Equal(t, "string", bars.Items.Type, "the array element type comes from the swagger:type base")
}
