// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug1635 asserts the resolution of go-swagger issue #1635 ("spec
// response generation without inner struct").
//
// A swagger:response (or swagger:parameters) that ANONYMOUSLY embeds a named
// struct marked in:body must treat the embed AS the body — exactly like a named
// `Body Foo` field — instead of promoting the embedded struct's fields. For
// responses the buggy form put the promoted fields under `headers`; for
// parameters it emitted one body parameter per promoted field (invalid OAS2,
// which allows at most one body parameter per operation).
func TestCoverage_Bug1635(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1635/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	const bodyRef = "#/definitions/PositionResponseBody"

	// --- responses ---
	// Guard rail: the named Body field form already yields a $ref body schema.
	named := doc.Responses["namedBodyResponse"].Schema
	require.NotNil(t, named)
	assert.Equal(t, bodyRef, named.Ref.String())

	// The #1635 defect: the anonymous-embed body must become a body schema, not
	// response headers.
	embed := doc.Responses["positionResponse"]
	assert.Empty(t, embed.Headers, "the embedded body fields must not become response headers (go-swagger#1635)")
	require.NotNil(t, embed.Schema, "the anonymous-embed body must produce a schema (go-swagger#1635)")
	assert.Equal(t, bodyRef, embed.Schema.Ref.String(),
		"the anonymous-embed body should $ref the embedded struct, like the named form")

	// --- parameters (same rule) ---
	// Guard rail: the named Body field form yields a single body parameter.
	namedParams := doc.Paths.Paths["/position-named"].Post.Parameters
	require.Len(t, namedParams, 1)
	require.NotNil(t, namedParams[0].Schema)
	assert.Equal(t, "body", namedParams[0].In)
	assert.Equal(t, bodyRef, namedParams[0].Schema.Ref.String())

	// The #1635 defect for parameters: the anonymous-embed body must become a
	// SINGLE body parameter $ref'ing the embedded struct, not one body param per
	// promoted field.
	embedParams := doc.Paths.Paths["/position"].Post.Parameters
	require.Len(t, embedParams, 1, "an in: body embed must produce exactly one body parameter (go-swagger#1635)")
	assert.Equal(t, "body", embedParams[0].In)
	require.NotNil(t, embedParams[0].Schema, "the anonymous-embed body parameter must carry a schema (go-swagger#1635)")
	assert.Equal(t, bodyRef, embedParams[0].Schema.Ref.String(),
		"the anonymous-embed body parameter should $ref the embedded struct, like the named form")
}
