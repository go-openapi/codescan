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

// TestCoverage_Bug2311 locks go-swagger issue #2311 (the behaviour behind the doc gap):
// `swagger:ignore` on a struct FIELD drops that property from the model (the docs only mentioned
// types/whole declarations).
func TestCoverage_Bug2311(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2311/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["Account"].Properties
	assert.Contains(t, props, "name")
	assert.NotContains(t, props, "secret", "swagger:ignore drops the field (go-swagger#2311)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2311_schema.json")
}
