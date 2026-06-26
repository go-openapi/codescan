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

// TestCoverage_Bug796 locks the fix for go-swagger issue #796 ("Definitions are generated without
// use of `swagger:model`").
//
// Under `generate spec -m` (ScanModels), the scanner used to emit definitions for types that carry
// no `swagger:model`: a bare interface (Handler) and a `swagger:parameters` struct (pingParams)
// both leaked into `definitions`.
// The parameter struct also leaked its `in: path` line into the property description.
//
// The expected behaviour — locked by the golden — is that only the `swagger:model` type
// (pingResponse) is emitted; the parameter struct is rendered as an operation parameter (with a
// clean description and a real `in` field) and the interface is dropped entirely.
func TestCoverage_Bug796(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/796/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Only the swagger:model type is a definition.
	_, hasModel := doc.Definitions["pingResponse"]
	assert.True(t, hasModel, "the swagger:model type must be emitted")
	assert.NotContains(t, doc.Definitions, "Handler", "an un-annotated interface must not become a definition")
	assert.NotContains(t, doc.Definitions, "pingParams", "a swagger:parameters struct must not become a definition")
	assert.Len(t, doc.Definitions, 1, "no extra definitions beyond the swagger:model")

	// The parameter struct renders as an operation parameter with a clean description (no `in:`
	// leakage) and a real `in` field.
	pi, ok := doc.Paths.Paths["/ping/{who}"]
	require.True(t, ok)
	require.NotNil(t, pi.Get)
	require.Len(t, pi.Get.Parameters, 1)
	who := pi.Get.Parameters[0]
	assert.Equal(t, "who", who.Name)
	assert.Equal(t, "path", who.In)
	assert.Equal(t, "Represents who is pinging", who.Description)
	assert.NotContains(t, who.Description, "in:", "the in: line must not leak into the description")

	scantest.CompareOrDumpJSON(t, doc, "bugs_796_schema.json")
}
