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

// TestCoverage_Bug1459 locks go-swagger issue #1459 ("example values on map keys
// instead of AdditionalProp"): a map field accepts an `example` object with
// meaningful keys, which the spec carries on the additionalProperties schema.
// The misleading additionalProp1/2/3 keys are only the Swagger UI's fallback
// when no example is provided.
func TestCoverage_Bug1459(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1459/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)

	settings := doc.Definitions["Cfg"].Properties["settings"]
	require.NotNil(t, settings.AdditionalProperties)
	ex, ok := settings.Example.(map[string]any)
	require.True(t, ok, "the map field carries a custom example object (go-swagger#1459)")
	assert.Equal(t, "30s", ex["timeout"])
	assert.Equal(t, "3", ex["retries"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1459_schema.json")
}
