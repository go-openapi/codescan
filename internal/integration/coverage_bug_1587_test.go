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

// TestCoverage_Bug1587 locks the fix for go-swagger issue #1587 ("import detection fails if package
// path does not match package name"): a field referencing a type from a package whose import-path
// tail (josev2) differs from its package name (jose) used to fail with "no import found for jose".
// go/packages resolves it by the real package name, so the $ref and its definition are emitted.
func TestCoverage_Bug1587(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1587/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "import resolution must not fail on path != package name")

	client, ok := doc.Definitions["oAuth2Client"]
	require.True(t, ok)
	jwks := client.Properties["jwks"]
	assert.Equal(t, "#/definitions/JSONWebKeySet", jwks.Ref.String())
	assert.Contains(t, doc.Definitions, "JSONWebKeySet", "the referenced type is emitted")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1587_schema.json")
}
