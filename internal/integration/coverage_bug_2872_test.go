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

// TestCoverage_Bug2872 verifies the fix for go-swagger issue #2872: an ExternalDocs block in
// swagger:meta now emits the top-level `externalDocs: {description, url}` (it was previously
// dropped — the keyword existed in the grammar but was not wired into the meta builder).
//
// Companion feature (forthcoming-features.md §9): ExternalDocs is also valid on other OAIv2
// objects (schema, operation, tag); supporting those is a separate feature beyond this meta-focused
// fix.
func TestCoverage_Bug2872(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2872/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// FIXED: meta ExternalDocs is emitted at the top level.
	require.NotNil(t, doc.ExternalDocs)
	assert.Equal(t, "foo bar", doc.ExternalDocs.Description)
	assert.Equal(t, "https://example.org", doc.ExternalDocs.URL)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2872_schema.json")
}
