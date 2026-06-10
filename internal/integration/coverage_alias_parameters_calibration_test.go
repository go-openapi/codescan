// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// findBodyParam locates the body parameter named `name` on the
// operation under (path, verb) in the spec — small helper to keep
// the cycle-4 R7 assertions readable.
func findBodyParam(t *testing.T, doc *oaispec.Swagger, path, verb, name string) oaispec.Parameter {
	t.Helper()
	op := doc.Paths.Paths[path].PathItemProps
	var item *oaispec.Operation
	switch verb {
	case "get":
		item = op.Get
	case "post":
		item = op.Post
	default:
		t.Fatalf("unsupported verb %q", verb)
	}
	require.NotNil(t, item, "operation %s %s must exist", verb, path)
	for _, p := range item.Parameters {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("parameter %q not found on %s %s", name, verb, path)
	return oaispec.Parameter{}
}

// Cycle-4 W3 alias workshop — parameters analogue of cycle-3.
//
// The three tests below scan the cycle-4 calibration fixture under
// the three alias modes (Default, RefAliases, TransparentAliases)
// and dump golden files capturing the pre-R7 state. The diff
// between these and the post-patch goldens will be the audit trail
// for the parameters-builder fix.
//
// The fixture deliberately includes:
//
//   - a top-level alias annotated `swagger:parameters` whose RHS is
//     an UNEXPORTED backing struct (Q12 witness — the unexported
//     struct must not leak into `definitions`);
//   - body fields typed as both unannotated and annotated aliases of
//     the canonical Payload model (R7-clause-2 witness — annotation
//     gates whether the alias surfaces as a first-class spec entity
//     at body field sites);
//   - a non-body field typed as an unannotated alias of a named
//     primitive (SimpleSchema target — R7-clause-3 witness).
//
// See `.claude/plans/workshops/alias-parameters.md` for the R7
// rule candidate and `.claude/plans/workshops/alias-ledger.md`
// cycle 4 for the running judgment.
//
// At this point (pre-patch), the goldens are expected to surface
// at least:
//
//   - `internalParams` and `AliasedTopParams` as `definitions`
//     entries (wrong under R7 clause 1);
//   - `PayloadAlias` / `PayloadAlias2` / `QueryIDAlias` as
//     `definitions` entries (wrong under R7 clause 2/3);
//   - `paths` populated by the two `swagger:route` handlers, with
//     whatever parameter shape the current alias dispatch produces.

func TestCoverage_AliasParametersCalibration_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-parameters-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// R7 clause 1 — neither the swagger:parameters alias nor its
	// unexported backing struct surface as model definitions. The
	// /aliased-top operation still gets its parameters built correctly:
	// the fields of the unaliased backing struct become parameters,
	// and `body` resolves to the canonical Payload model.
	assert.NotContains(t, doc.Definitions, "AliasedTopParams",
		"R7 clause 1: top-level swagger:parameters alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "internalParams",
		"Q12 fix: unexported backing struct must not surface as a definition")
	topBody := findBodyParam(t, doc, "/aliased-top", "get", "body")
	assert.Equal(t, "#/definitions/Payload", topBody.Schema.Ref.String(),
		"R7 clause 1: top-level alias's body param reaches the canonical Payload via the unaliased target's fields")

	// R7 clause 2 — annotation gates first-class identity at body field sites.
	assert.NotContains(t, doc.Definitions, "PayloadAlias",
		"R7 clause 2: unannotated body-field alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "PayloadAlias2",
		"R7 clause 2: unannotated alias chain must not produce a definition")
	require.Contains(t, doc.Definitions, "PayloadAliasModeled",
		"R7 clause 2: annotated alias keeps its own definition")

	// R7 clause 3 — non-body SimpleSchema target alias must not surface.
	assert.NotContains(t, doc.Definitions, "QueryIDAlias",
		"R7 clause 3: non-body alias must not produce a definition (SimpleSchema target)")

	// Body-field $ref targets pin the annotation gate.
	plainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasPlain")
	assert.Equal(t, "#/definitions/Payload", plainBody.Schema.Ref.String(),
		"unannotated body alias dissolves to the unaliased target")

	annotatedBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasModeled")
	assert.Equal(t, "#/definitions/PayloadAliasModeled", annotatedBody.Schema.Ref.String(),
		"annotated body alias preserves the alias name in the field $ref")

	chainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasChain")
	assert.Equal(t, "#/definitions/Payload", chainBody.Schema.Ref.String(),
		"unannotated alias chain dissolves fully to the unaliased target")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_parameters_calibration_default.json")
}

func TestCoverage_AliasParametersCalibration_Ref(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-parameters-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// R7 behaviour at field sites is mode-agnostic: the annotation
	// gate fires the same way under RefAliases as under Default. The
	// mode only affects the alias decl's OWN definition shape
	// (PayloadAliasModeled's downstream representation), not the
	// field $ref target.
	assert.NotContains(t, doc.Definitions, "AliasedTopParams")
	assert.NotContains(t, doc.Definitions, "internalParams")
	assert.NotContains(t, doc.Definitions, "PayloadAlias")
	assert.NotContains(t, doc.Definitions, "PayloadAlias2")
	assert.NotContains(t, doc.Definitions, "QueryIDAlias")
	assert.NotContains(t, doc.Definitions, "QueryID",
		"QueryID leak fix: the non-body chain target must not surface as a definition under Ref")
	require.Contains(t, doc.Definitions, "PayloadAliasModeled")

	plainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasPlain")
	assert.Equal(t, "#/definitions/Payload", plainBody.Schema.Ref.String())
	annotatedBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasModeled")
	assert.Equal(t, "#/definitions/PayloadAliasModeled", annotatedBody.Schema.Ref.String())
	chainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasChain")
	assert.Equal(t, "#/definitions/Payload", chainBody.Schema.Ref.String(),
		"chain must dissolve fully even under RefAliases (no one-step ref)")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_parameters_calibration_ref.json")
}

func TestCoverage_AliasParametersCalibration_Transparent(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:           []string{"./enhancements/alias-parameters-calibration/..."},
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
	assert.NotContains(t, doc.Definitions, "AliasedTopParams")
	assert.NotContains(t, doc.Definitions, "internalParams")
	assert.NotContains(t, doc.Definitions, "PayloadAlias")
	assert.NotContains(t, doc.Definitions, "QueryIDAlias")
	require.Contains(t, doc.Definitions, "PayloadAliasModeled",
		"R2 holds even under Transparent: annotated alias keeps its decl entry")

	// All body $refs dissolve to Payload — Transparent supersedes annotation.
	plainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasPlain")
	assert.Equal(t, "#/definitions/Payload", plainBody.Schema.Ref.String())
	annotatedBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasModeled")
	assert.Equal(t, "#/definitions/Payload", annotatedBody.Schema.Ref.String(),
		"Transparent supersedes annotation; body $ref dissolves to Payload")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_parameters_calibration_transparent.json")
}
