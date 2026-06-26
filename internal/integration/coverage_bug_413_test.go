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

// TestCoverage_Bug413 locks the fix for go-swagger issue #413 ("error reading embedded struct"):
// embedding an external-package struct that itself carries a custom-typed field
// (maps.NearbySearchRequest with a RankBy field) used to fail the scan with "unable to resolve
// embedded struct for: RankBy".
//
// It now resolves cleanly — the embedded fields are promoted as parameters.
func TestCoverage_Bug413(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/413/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err, "embedding an external struct must not error (the #413 bug)")

	op := doc.Paths.Paths["/places/nearby"]
	require.NotNil(t, op.Post)
	names := make(map[string]string, len(op.Post.Parameters))
	for _, p := range op.Post.Parameters {
		names[p.Name] = p.Type
	}
	assert.Equal(t, "string", names["keyword"])
	assert.Equal(t, "string", names["rankby"], "the custom RankBy type resolves to string")

	scantest.CompareOrDumpJSON(t, doc, "bugs_413_schema.json")
}
