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

// TestCoverage_AliasResponseShapes_Default — Q4 / Q5 exploration
// goldens. After Q4's fix to responses.buildFieldAlias (parity with
// parameters), default mode dissolves an alias chain on a body
// field down to the canonical decl ($ref → Envelope, no
// EnvelopeAlias{,2} pollution in definitions). Headers typed as an
// alias of a named primitive now also surface correctly as
// SimpleSchema primitives ({string, ""}); pre-Q4 they emitted
// empty headers because the response builder's buildFieldAlias
// fell through to makeRef even on header targets.
func TestCoverage_AliasResponseShapes_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-response-shapes/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Header aliased to a named string: surfaces as a primitive.
	hdr := doc.Responses["headerAliasedBasicResponse"].Headers["X-Session"]
	assert.Equal(t, "string", hdr.Type, "alias-of-named-string header should be {string, ''} post-Q4")

	// Body aliased through a chain: dissolves to canonical Envelope
	// under default mode (no per-alias definition pollution).
	bodyRef := doc.Responses["bodyAliasResponse"].Schema.Ref.String()
	assert.Equal(t, "#/definitions/Envelope", bodyRef, "body alias-chain dissolves to canonical decl")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias", "EnvelopeAlias should not pollute definitions under default mode")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias2", "EnvelopeAlias2 should not pollute definitions under default mode")
	assert.NotContains(t, doc.Definitions, "SessionIDAlias", "SessionIDAlias should not pollute definitions under default mode")

	// R8 bidirectional companion — the annotated alias preserves
	// its identity at the body response site; the unannotated
	// alias above dissolves to canonical.
	modeledRef := doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String()
	assert.Equal(t, "#/definitions/EnvelopeAliasModeled", modeledRef,
		"R8: annotated body response alias keeps the alias name in the $ref")
	assert.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"R8: annotated alias has its own definition")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_response_shapes_default.json")
}

// TestCoverage_AliasResponseShapes_Ref — under RefAliases the body
// $ref chains through EnvelopeAlias2 → EnvelopeAlias → Envelope;
// headers STILL expand inline (no $ref) because $ref is not legal
// on a SimpleSchema target. The Q4 gate's
// "in != body OR !RefAliases" branch covers both legs.
func TestCoverage_AliasResponseShapes_Ref(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-response-shapes/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	hdr := doc.Responses["headerAliasedBasicResponse"].Headers["X-Session"]
	assert.Equal(t, "string", hdr.Type, "header still expands inline under RefAliases")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_response_shapes_ref.json")
}

// TestCoverage_AliasResponseShapes_Transparent — TransparentAliases
// dissolves every layer up front; the body $ref points at
// Envelope, headers carry their primitive type, and the
// alias-of-struct case on a header surfaces empty (the struct
// can't reduce to a SimpleSchema primitive — same as Q2's
// post-fix behaviour).
func TestCoverage_AliasResponseShapes_Transparent(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:           []string{"./enhancements/alias-response-shapes/..."},
		WorkDir:            scantest.FixturesDir(),
		ScanModels:         true,
		TransparentAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	hdr := doc.Responses["headerAliasedBasicResponse"].Headers["X-Session"]
	assert.Equal(t, "string", hdr.Type, "header expands inline under Transparent")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_response_shapes_transparent.json")
}
