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

// TestCoverage_Bug3005 documents the CURRENT (buggy) behaviour for go-swagger
// issue #3005 ("additionalProperties are lost when generating spec from
// code"): a map field tagged json:"-", intended to carry the model's
// additionalProperties, is dropped — only field1 is emitted, with no
// additionalProperties on the object schema.
//
// FIX TARGET (fix/go-swagger-3005) — DESIGN DECISION PENDING: provide a way
// to mark a map field as the schema's additionalProperties (the reporter
// proposes a dedicated annotation, e.g. `// Additional properties`). Decide
// the annotation in the fix-needed analysis pass, then assert
// sch.AdditionalProperties.Schema.Type == "number" and regenerate the golden.
func TestCoverage_Bug3005(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3005/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	sch, ok := doc.Definitions["TestAdditionalProperties"]
	require.True(t, ok)
	assert.Contains(t, sch.Properties, "field1")
	// CURRENT (wrong): the json:"-" map is dropped; no additionalProperties.
	assert.Nil(t, sch.AdditionalProperties,
		"TODO #3005: a marked map field should become the schema additionalProperties")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3005_schema.json")
}
