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

// Field-reach witnesses for the schema builder's alias-handling
// contract. The `alias-calibration-embed` fixture pairs two
// envelopes:
//
//   - Envelope               → uses the UNANNOTATED `BaseAlias`
//   - EnvelopeAnnotatedAlias → uses the ANNOTATED   `BaseAliasModeled`
//
// Same Go type behind both aliases (`types.Unalias` collapses
// them to `Base`); the ONLY difference is the `swagger:model`
// annotation. The annotation gates whether the alias is a
// first-class spec entity at use sites:
//
//   - annotated   → preserves `$ref: <alias-name>` + own definition
//   - unannotated → dissolves to `$ref: <unaliased-target>`; no definition
//
// `TransparentAliases=true` overrides the gate (always dissolve at
// use sites). See [§aliases](../builders/schema/README.md#aliases)
// for the rule.

// TestCoverage_AliasField_Default captures the Default-mode
// (Expand) shape:
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

	// Unannotated half — dissolved at use site, no own def.
	assert.NotContains(t, doc.Definitions, "BaseAlias",
		"unannotated alias must not have a standalone definition")
	require.Contains(t, doc.Definitions, "Envelope")
	viaAlias := doc.Definitions["Envelope"].Properties["viaAlias"]
	assert.Equal(t, "#/definitions/Base", viaAlias.Ref.String(),
		"unannotated alias dissolves to its unaliased target at field sites")

	// Annotated half — kept as first-class spec entity.
	require.Contains(t, doc.Definitions, "BaseAliasModeled",
		"annotated alias keeps its standalone definition")
	require.Contains(t, doc.Definitions, "EnvelopeAnnotatedAlias")
	viaModeled := doc.Definitions["EnvelopeAnnotatedAlias"].Properties["viaAliasModeled"]
	assert.Equal(t, "#/definitions/BaseAliasModeled", viaModeled.Ref.String(),
		"annotated alias keeps its $ref identity at field sites")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_default.json")
}

// TestCoverage_AliasField_Ref captures the RefAliases shape. The
// annotation gate fires identically to Default at use sites —
// mode only affects the alias decl's own definition shape (chain
// `$ref` instead of structural copy), not the field-site `$ref`
// target.
func TestCoverage_AliasField_Ref(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-calibration-embed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Unannotated half — dissolved under Ref.
	assert.NotContains(t, doc.Definitions, "BaseAlias",
		"unannotated alias must not have a standalone definition under Ref")
	viaAlias := doc.Definitions["Envelope"].Properties["viaAlias"]
	assert.Equal(t, "#/definitions/Base", viaAlias.Ref.String(),
		"unannotated alias dissolves to its target under Ref as well")

	// Annotated half — kept under Ref.
	require.Contains(t, doc.Definitions, "BaseAliasModeled")
	viaModeled := doc.Definitions["EnvelopeAnnotatedAlias"].Properties["viaAliasModeled"]
	assert.Equal(t, "#/definitions/BaseAliasModeled", viaModeled.Ref.String(),
		"annotated alias keeps its $ref identity at field sites under Ref")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_ref.json")
}

// TestCoverage_AliasField_Transparent captures the dissolve-all
// shape. Both Envelope.viaAlias and
// EnvelopeAnnotatedAlias.viaAliasModeled resolve to {$ref: Base}
// because Transparent supersedes the annotation gate at use sites.
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
	// types resolve to Base at the use site, and the unannotated
	// alias does not appear in definitions.
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
	// `swagger:model` forces decl-level registration regardless of
	// mode. Under Transparent the alias dissolves at USE sites
	// (the viaAliasModeled $ref above) but the decl entry
	// survives — the dangling annotated def trade-off.
	assert.Contains(t, doc.Definitions, "BaseAliasModeled",
		"annotated decl entry survives Transparent (swagger:model forces registration)")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_field_transparent.json")
}
