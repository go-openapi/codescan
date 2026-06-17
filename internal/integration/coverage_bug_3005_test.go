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

// TestCoverage_Bug3005 locks the resolution of go-swagger issue #3005
// ("additionalProperties are lost when generating spec from code"): a model
// with named properties AND free-form values. The json:"-" map the reporter
// used is muted; the type-level `swagger:additionalProperties number` marker
// supplies the additionalProperties value schema. field1 stays a named
// property alongside it.
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

	// The #3005 fix: the marker complements the object with additionalProperties.
	require.NotNil(t, sch.AdditionalProperties)
	require.NotNil(t, sch.AdditionalProperties.Schema)
	assert.True(t, sch.AdditionalProperties.Schema.Type.Contains("number"),
		"swagger:additionalProperties number sets the value schema (go-swagger#3005)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3005_schema.json")
}
