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

	scantest.CompareOrDumpJSON(t, doc, "enhancements_alias_findmodel_witness.json")
}
