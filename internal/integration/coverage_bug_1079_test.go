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

// TestCoverage_Bug1079 documents the resolution to go-swagger issue #1079 ("don't generate schemas
// for models without swagger:model"): under -m, codescan emits the annotated model plus the types
// it transitively references — but NOT unreferenced non-annotated types.
//
// So no "junk schemas" appear; a referenced type must still be emitted to keep the $ref resolvable.
//
// 📖 Need doc: clarify -m semantics (annotated models + referenced types, not every struct in the
// package).
func TestCoverage_Bug1079(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1079/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	assert.Contains(t, doc.Definitions, "Annotated", "the swagger:model is emitted")
	assert.Contains(t, doc.Definitions, "Referenced", "a referenced type must be emitted for the $ref")
	assert.NotContains(t, doc.Definitions, "Unreferenced",
		"an unreferenced non-annotated type is omitted (no junk schema)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1079_schema.json")
}
