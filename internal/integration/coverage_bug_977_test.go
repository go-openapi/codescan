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

// TestCoverage_Bug977 locks go-swagger issue #977 ("aliased string type as map
// key"): a map keyed by a named string type (type Role string) renders as
// {type:object, additionalProperties:<value>} — its underlying string kind makes
// it a valid object-key.
func TestCoverage_Bug977(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/977/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true})
	require.NoError(t, err)
	perms := doc.Definitions["Acl"].Properties["perms"]
	assert.Equal(t, []string{"object"}, []string(perms.Type))
	require.NotNil(t, perms.AdditionalProperties)
	require.NotNil(t, perms.AdditionalProperties.Schema)
	assert.Equal(t, []string{"array"}, []string(perms.AdditionalProperties.Schema.Type), "aliased-string map key is honored (go-swagger#977)")
	scantest.CompareOrDumpJSON(t, doc, "bugs_977_schema.json")
}
