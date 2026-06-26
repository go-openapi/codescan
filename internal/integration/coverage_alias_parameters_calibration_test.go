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

// findBodyParam locates the body parameter named `name` on the operation under (path, verb) in the
// spec — small helper to keep the parameters-builder alias-handling assertions readable.
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

// Calibration coverage for the parameters builder's alias-handling contract.
// The three tests below scan the calibration fixture under all three alias modes (Default,
// RefAliases, TransparentAliases) and pin both inline assertions and goldens.
//
// The fixture deliberately includes:
//
//   - a top-level alias annotated `swagger:parameters` whose RHS is
//     an UNEXPORTED backing struct (must not leak to definitions);
//   - body fields typed as both unannotated and annotated aliases
//     of the canonical Payload model (annotation gate witness);
//   - a non-body field typed as an unannotated alias of a named
//     primitive (SimpleSchema target — always inline).
//
// See [§alias-handling](../builders/parameters/README.md#alias-handling) for the contract.

func TestCoverage_AliasParametersCalibration_Default(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-parameters-calibration/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Top-level `swagger:parameters` alias — neither the alias nor its unexported backing struct
	// surface as model definitions.
	// The /aliased-top operation still gets its parameters built correctly: the fields of the
	// unaliased backing struct become parameters, and `body` resolves to the canonical Payload model.
	assert.NotContains(t, doc.Definitions, "AliasedTopParams",
		"top-level swagger:parameters alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "internalParams",
		"unexported backing struct must not surface as a definition")
	topBody := findBodyParam(t, doc, "/aliased-top", "get", "body")
	assert.Equal(t, "#/definitions/Payload", topBody.Schema.Ref.String(),
		"top-level alias's body param reaches the canonical Payload via the unaliased target's fields")

	// Annotation gates first-class identity at body field sites.
	assert.NotContains(t, doc.Definitions, "PayloadAlias",
		"unannotated body-field alias must not produce a definition")
	assert.NotContains(t, doc.Definitions, "PayloadAlias2",
		"unannotated alias chain must not produce a definition")
	require.Contains(t, doc.Definitions, "PayloadAliasModeled",
		"annotated alias keeps its own definition")

	// Non-body SimpleSchema target alias must not surface.
	assert.NotContains(t, doc.Definitions, "QueryIDAlias",
		"non-body alias must not produce a definition (SimpleSchema target)")

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

	// Behaviour at field sites is mode-agnostic: the annotation gate fires the same way under
	// RefAliases as under Default.
	// The mode only affects the alias decl's OWN definition shape (PayloadAliasModeled's downstream
	// representation), not the field $ref target.
	assert.NotContains(t, doc.Definitions, "AliasedTopParams")
	assert.NotContains(t, doc.Definitions, "internalParams")
	assert.NotContains(t, doc.Definitions, "PayloadAlias")
	assert.NotContains(t, doc.Definitions, "PayloadAlias2")
	assert.NotContains(t, doc.Definitions, "QueryIDAlias")
	assert.NotContains(t, doc.Definitions, "QueryID",
		"the non-body chain target must not surface as a definition under Ref")
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

	// TransparentAliases supersedes the annotation gate at use sites (every body alias dissolves to
	// its unaliased target), but `swagger:model` still forces decl-level registration — annotated
	// aliases keep their definition entry.
	assert.NotContains(t, doc.Definitions, "AliasedTopParams")
	assert.NotContains(t, doc.Definitions, "internalParams")
	assert.NotContains(t, doc.Definitions, "PayloadAlias")
	assert.NotContains(t, doc.Definitions, "QueryIDAlias")
	require.Contains(t, doc.Definitions, "PayloadAliasModeled",
		"annotated alias keeps its decl entry even under Transparent")

	// All body $refs dissolve to Payload — Transparent supersedes annotation.
	plainBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasPlain")
	assert.Equal(t, "#/definitions/Payload", plainBody.Schema.Ref.String())
	annotatedBody := findBodyParam(t, doc, "/direct", "post", "bodyAliasModeled")
	assert.Equal(t, "#/definitions/Payload", annotatedBody.Schema.Ref.String(),
		"Transparent supersedes annotation; body $ref dissolves to Payload")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_parameters_calibration_transparent.json")
}
