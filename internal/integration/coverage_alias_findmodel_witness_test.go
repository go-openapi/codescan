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

// TestCoverage_AliasFindModelWitness — locks the post-migration
// behaviour of GetModel + AppendPostDecl on alias targets that are
// not annotated swagger:model.
//
// PlainTarget (unannotated) must reach spec.definitions via the
// orchestrator's discovery loop after the alias's RHS lookup
// queues it; under the pre-migration FindModel, the same decl
// reached definitions via the implicit ExtraModels side effect.
// Both paths produce identical output; the golden is the witness.
func TestCoverage_AliasFindModelWitness(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/alias-findmodel-witness/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		RefAliases: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Definitions, "PlainTarget",
		"unannotated alias target must reach definitions via the orchestrator's discovery path")

	// R6 / R7 / R8 bidirectional witness — unannotated alias
	// dissolves to its target at every use site; annotated alias
	// preserves the alias name. AliasOfPlain (unannotated) and
	// AliasOfPlainModeled (annotated) sit on the same fixture so
	// both halves are visible side by side in this golden.
	assert.NotContains(t, doc.Definitions, "AliasOfPlain",
		"R6: unannotated alias must not produce a definition")
	require.Contains(t, doc.Definitions, "AliasOfPlainModeled",
		"R6: annotated alias keeps its own definition")

	// Body-response $refs pin the gate on the responses side (R8).
	assert.Equal(t, "#/definitions/PlainTarget",
		doc.Responses["witnessResponse"].Schema.Ref.String(),
		"R8: unannotated body response alias dissolves to the underlying target")
	assert.Equal(t, "#/definitions/AliasOfPlainModeled",
		doc.Responses["witnessModeledResponse"].Schema.Ref.String(),
		"R8: annotated body response alias preserves the alias name")

	// The parameters-side bidirectional contract is observable in
	// the golden but not in doc.Parameters here: this fixture has no
	// swagger:route, so the swagger:parameters declarations don't
	// bind to an operation and never reach spec.Parameters.
	// witnessRequest / witnessModeledRequest live only in the
	// scanner's internal parameter-set map; they show up in the
	// golden via the WitnessParams / WitnessParamsModeled struct
	// surfaces (or their absence under R7's clause-1 leak fix).
	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_findmodel_witness.json")
}
