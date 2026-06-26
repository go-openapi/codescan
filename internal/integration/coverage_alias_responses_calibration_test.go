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

// Calibration coverage for the responses builder's alias-handling contract.
// The three tests below scan the calibration fixture under all three alias modes (Default,
// RefAliases, TransparentAliases) and pin both inline assertions and goldens.
//
// The fixture deliberately includes:
//
//   - a top-level alias annotated `swagger:response` whose RHS is
//     an UNEXPORTED backing struct (must not leak to definitions);
//   - body fields typed as both unannotated and annotated aliases
//     of the canonical Envelope model (annotation gate witness);
//   - a non-body (header) field typed as an unannotated alias of a
//     named primitive (SimpleSchema target — always inline).
//
// See [§alias-handling](../builders/responses/README.md#alias-handling) for the contract.

func TestCoverage_AliasResponsesCalibration_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-responses-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Top-level `swagger:response` alias — neither the alias nor its unexported backing struct
	// surface as model definitions.
	//
	// The aliasedTopResponse still gets a valid body schema (the canonical Envelope) — earlier
	// states crashed in Default with "anonymous types are currently not supported" or attached an
	// empty schema; both gone.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse",
		"top-level swagger:response alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "internalResponse",
		"unexported backing struct must not surface as a definition")

	// Annotation gates first-class identity at body field sites.
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias",
		"unannotated body alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias2",
		"unannotated alias chain must not produce a definition")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"annotated alias keeps its own definition")

	// Non-body SimpleSchema target alias must not surface.
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias",
		"non-body alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "HeaderID",
		"chain target of a non-body alias must not surface either")

	// Response body $refs pin the contract.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["aliasedTopResponse"].Schema.Ref.String(),
		"top-level alias reaches the canonical Envelope")
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasPlainResponse"].Schema.Ref.String(),
		"unannotated body alias dissolves to Envelope")
	assert.Equal(t, "#/definitions/EnvelopeAliasModeled",
		doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String(),
		"annotated body alias preserves the alias name")
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasChainResponse"].Schema.Ref.String(),
		"unannotated alias chain dissolves fully to Envelope")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_responses_calibration_default.json")
}

func TestCoverage_AliasResponsesCalibration_Ref(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-responses-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Behaviour at use sites is mode-agnostic: the annotation gate fires the same way under RefAliases
	// as under Default.
	// The mode only affects the alias decl's OWN definition shape, not the field $ref target.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse")
	assert.NotContains(t, doc.Definitions, "internalResponse")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias2")
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias")
	assert.NotContains(t, doc.Definitions, "HeaderID",
		"HeaderID chain target must not surface either under Ref")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled")

	// Top-level response carries the canonical Envelope $ref.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["aliasedTopResponse"].Schema.Ref.String(),
		"top-level alias reaches the canonical Envelope under Ref")
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasPlainResponse"].Schema.Ref.String())
	assert.Equal(t, "#/definitions/EnvelopeAliasModeled",
		doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String())
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasChainResponse"].Schema.Ref.String(),
		"chain must dissolve fully under Ref (no one-step ref)")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_responses_calibration_ref.json")
}

func TestCoverage_AliasResponsesCalibration_Transparent(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:           []string{"./enhancements/alias-responses-calibration/..."},
		WorkDir:            scantest.FixturesDir(),
		ScanModels:         true,
		TransparentAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// TransparentAliases supersedes the annotation gate at use sites (every body alias dissolves to
	// its unaliased target), but `swagger:model` still forces decl-level registration — annotated
	// aliases keep their definition entry.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse")
	assert.NotContains(t, doc.Definitions, "internalResponse")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias")
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"annotated alias keeps its decl entry even under Transparent")

	// All response bodies → $ref: Envelope; Transparent supersedes annotation.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasPlainResponse"].Schema.Ref.String())
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String(),
		"Transparent supersedes annotation; body $ref dissolves to Envelope")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_responses_calibration_transparent.json")
}
