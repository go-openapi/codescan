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

// Q-E witnesses — the field-reach axis of the W3 alias workshop.
//
// The cycle-3 fixture `alias-calibration-embed` pairs two
// envelopes:
//
//   - Envelope               → uses the UNANNOTATED `BaseAlias`
//   - EnvelopeAnnotatedAlias → uses the ANNOTATED   `BaseAliasModeled`
//
// Same Go type behind both aliases (`types.Unalias` collapses them
// to `Base`); the ONLY difference is the `swagger:model` annotation.
// Per R6 (the rule the workshop converged on), that annotation
// gates whether the alias is a first-class spec entity:
//
//   - annotated   → preserves `$ref: <alias-name>` at use sites + own definition
//   - unannotated → dissolves to `$ref: <unaliased-target>` at use sites; no definition
//
// `TransparentAliases=true` is the canonical "always dissolve"
// mode; under R6 Default and Ref converge on Transparent's shape
// for unannotated aliases, while preserving the alias identity
// for annotated ones.
//
// Three tests below capture goldens per mode so the diff between
// pre-R6 and post-R6 is the audit trail. The assertions are
// minimal at this point — the goldens carry the full shape.

// TestCoverage_AliasField_Default captures the Default-mode
// (Expand) shape for the field-reach Q-E witnesses. Under R6:
//
//   - Envelope.viaAlias (UNANNOTATED BaseAlias) → $ref: Base
//     (alias dissolves at the use site); BaseAlias has no own def
//   - EnvelopeAnnotatedAlias.viaAliasModeled (ANNOTATED) →
//     $ref: BaseAliasModeled (alias keeps identity); has own def
func TestCoverage_AliasField_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Unannotated half — R6 negative: dissolved at use site, no own def.
	assert.NotContains(t, doc.Definitions, "BaseAlias",
		"R6: unannotated alias must not have a standalone definition")
	require.Contains(t, doc.Definitions, "Envelope")
	viaAlias := doc.Definitions["Envelope"].Properties["viaAlias"]
	assert.Equal(t, "#/definitions/Base", viaAlias.Ref.String(),
		"R6: unannotated alias dissolves to its unaliased target at field sites")

	// Annotated half — R6 positive: kept as first-class spec entity.
	require.Contains(t, doc.Definitions, "BaseAliasModeled",
		"annotated alias keeps its standalone definition")
	require.Contains(t, doc.Definitions, "EnvelopeAnnotatedAlias")
	viaModeled := doc.Definitions["EnvelopeAnnotatedAlias"].Properties["viaAliasModeled"]
	assert.Equal(t, "#/definitions/BaseAliasModeled", viaModeled.Ref.String(),
		"annotated alias keeps its $ref identity at field sites")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_default.json")
}

// TestCoverage_AliasField_Ref captures the RefAliases shape. R6
// applies identically to Default — annotation gates first-class
// status; the only mode-level effect is the chain shape of the
// annotated decl itself (separately covered by the Q-C tests).
func TestCoverage_AliasField_Ref(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Unannotated half — R6 negative under Ref.
	assert.NotContains(t, doc.Definitions, "BaseAlias",
		"R6: unannotated alias must not have a standalone definition under Ref")
	viaAlias := doc.Definitions["Envelope"].Properties["viaAlias"]
	assert.Equal(t, "#/definitions/Base", viaAlias.Ref.String(),
		"R6: unannotated alias dissolves to its target under Ref as well")

	// Annotated half — R6 positive under Ref.
	require.Contains(t, doc.Definitions, "BaseAliasModeled")
	viaModeled := doc.Definitions["EnvelopeAnnotatedAlias"].Properties["viaAliasModeled"]
	assert.Equal(t, "#/definitions/BaseAliasModeled", viaModeled.Ref.String(),
		"annotated alias keeps its $ref identity at field sites under Ref")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_ref.json")
}

// TestCoverage_AliasField_Transparent captures the dissolve-all
// shape. Both Envelope.viaAlias AND
// EnvelopeAnnotatedAlias.viaAliasModeled resolve to {$ref: Base}
// because Transparent supersedes annotation by design.
func TestCoverage_AliasField_Transparent(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:           []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:            scantest.FixturesDir(),
		ScanModels:         true,
		TransparentAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Transparent dissolves regardless of annotation: both field
	// types resolve to Base at the use site, and the aliases do
	// not appear in definitions.
	require.Contains(t, doc.Definitions, "Envelope")
	viaAlias := doc.Definitions["Envelope"].Properties["viaAlias"]
	assert.Equal(t, "#/definitions/Base", viaAlias.Ref.String(),
		"Transparent dissolves the unannotated alias to Base")

	require.Contains(t, doc.Definitions, "EnvelopeAnnotatedAlias")
	viaModeled := doc.Definitions["EnvelopeAnnotatedAlias"].Properties["viaAliasModeled"]
	assert.Equal(t, "#/definitions/Base", viaModeled.Ref.String(),
		"Transparent supersedes annotation; annotated alias also dissolves to Base")

	assert.NotContains(t, doc.Definitions, "BaseAlias",
		"Transparent dissolves the unannotated alias decl")
	// R2 still applies to annotated decls under Transparent: the
	// `swagger:model` annotation forces decl-level registration
	// regardless of mode. Transparent dissolves the alias at USE
	// sites (the viaAliasModeled $ref above), leaving
	// BaseAliasModeled as a "dangling annotated def" — a known
	// trade-off accepted in cycle 1 (R2-implication).
	assert.Contains(t, doc.Definitions, "BaseAliasModeled",
		"R2 holds even under Transparent: annotated alias keeps its decl entry")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_transparent.json")
}
