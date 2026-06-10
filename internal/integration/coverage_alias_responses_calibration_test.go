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

// Cycle-5 W3 alias workshop — responses analogue of cycle-4.
//
// The three tests below scan the cycle-5 calibration fixture under
// the three alias modes and dump golden files capturing the
// pre-R8 state. The diff between these and the post-patch goldens
// will be the audit trail for the responses-builder fix.
//
// The fixture deliberately includes:
//
//   - a top-level alias annotated `swagger:response` whose RHS is
//     an UNEXPORTED backing struct (Q12 witness — the unexported
//     struct must not leak into `definitions`);
//   - body fields typed as both unannotated and annotated aliases of
//     the canonical Envelope model (R8 clause-2 witness — annotation
//     gates whether the alias surfaces as a first-class spec entity
//     at body field sites);
//   - a non-body (header) field typed as an unannotated alias of a
//     named primitive (SimpleSchema target — R8 clause-3 witness).
//
// At this point (pre-patch), the goldens are expected to surface
// leaks symmetrical to the cycle-4 pre-patch parameters captures
// (AliasedTopResponse, internalResponse, EnvelopeAlias, etc. in
// definitions under Default / Ref). Default mode may additionally
// crash on top-level response aliases — the existing
// alias-response/ fixture documents that gap.

func TestCoverage_AliasResponsesCalibration_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-responses-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// R8 clause 1 — neither the swagger:response alias nor its
	// unexported backing struct surface as model definitions. The
	// aliasedTopResponse still gets a valid body schema (the
	// canonical Envelope), confirming the previous "anonymous
	// types are currently not supported" crash and the empty-schema
	// drop are both gone.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse",
		"R8 clause 1: top-level swagger:response alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "internalResponse",
		"Q12 fix: unexported backing struct must not surface as a definition")

	// R8 clause 2 — annotation gates first-class identity at body field sites.
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias",
		"R8 clause 2: unannotated body alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias2",
		"R8 clause 2: unannotated alias chain must not produce a definition")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"R8 clause 2: annotated alias keeps its own definition")

	// R8 clause 3 — non-body SimpleSchema target alias must not surface.
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias",
		"R8 clause 3: non-body alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "HeaderID",
		"chain target of a non-body alias must not surface either")

	// Response body $refs pin the contract.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["aliasedTopResponse"].Schema.Ref.String(),
		"R8 clause 1: top-level alias reaches the canonical Envelope")
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasPlainResponse"].Schema.Ref.String(),
		"R8 clause 2: unannotated body alias dissolves to Envelope")
	assert.Equal(t, "#/definitions/EnvelopeAliasModeled",
		doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String(),
		"R8 clause 2: annotated body alias preserves the alias name")
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasChainResponse"].Schema.Ref.String(),
		"R8 clause 2: unannotated alias chain dissolves fully to Envelope")

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

	// R8 behaviour at use sites is mode-agnostic: the annotation gate
	// fires the same way under RefAliases as under Default. The mode
	// only affects the alias decl's OWN definition shape, not the
	// field $ref target.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse")
	assert.NotContains(t, doc.Definitions, "internalResponse")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias2")
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias")
	assert.NotContains(t, doc.Definitions, "HeaderID",
		"HeaderID chain target must not surface either under Ref")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled")

	// Pre-R8, aliasedTopResponse had an EMPTY schema attached because
	// the old buildAlias RefAliases branch created a throwaway typable
	// inside MakeRef that never connected to the response object. R8
	// routes through buildFromType(Rhs), which dispatches to
	// buildNamedType → buildFromStruct and walks the backing struct's
	// Body field — restoring the canonical Envelope $ref.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["aliasedTopResponse"].Schema.Ref.String(),
		"R8 fixes the pre-existing empty-schema bug on Ref-mode top-level response aliases")
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

	// TransparentAliases supersedes the annotation gate at use sites
	// (every body alias dissolves to its unaliased target), but R2
	// holds at the decl level — annotated aliases keep their
	// definition entry.
	assert.NotContains(t, doc.Definitions, "AliasedTopResponse")
	assert.NotContains(t, doc.Definitions, "internalResponse")
	assert.NotContains(t, doc.Definitions, "EnvelopeAlias")
	assert.NotContains(t, doc.Definitions, "HeaderIDAlias")
	require.Contains(t, doc.Definitions, "EnvelopeAliasModeled",
		"R2 holds even under Transparent: annotated alias keeps its decl entry")

	// All response bodies → $ref: Envelope; Transparent supersedes annotation.
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasPlainResponse"].Schema.Ref.String())
	assert.Equal(t, "#/definitions/Envelope",
		doc.Responses["bodyAliasModeledResponse"].Schema.Ref.String(),
		"Transparent supersedes annotation; body $ref dissolves to Envelope")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_responses_calibration_transparent.json")
}
