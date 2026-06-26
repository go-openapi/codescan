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

// TestCoverage_Bug2701 covers go-swagger issue #2701: an `in: path` (and `required: true`)
// annotation on an EMBEDDED struct field used to be ignored — the promoted fields
// (vendorID/urcapID/version) defaulted to `in: query`, invalid for the route's path templates.
//
// FIXED (fix/go-swagger-2701): the embed's in:/required: annotation propagates to the parameters it
// promotes, recursively.
// The test also guards the exportedness contract: only exported fields are promoted (including ones
// reached through an unexported embed), and unexported fields never surface.
func TestCoverage_Bug2701(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2701/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pi, ok := doc.Paths.Paths["/urcap/{vendorID}/{urcapID}/{version}/{trace}"]
	require.True(t, ok)
	require.NotNil(t, pi.Delete)
	require.NotEmpty(t, pi.Delete.Parameters)

	names := make([]string, 0, len(pi.Delete.Parameters))
	for _, p := range pi.Delete.Parameters {
		// FIXED: the embed's in:path propagates to every promoted field, including "trace" reached
		// through the unexported auditInfo embed.
		assert.Equal(t, "path", p.In,
			"in:path on the embedded struct must make %q a path param (go-swagger#2701)", p.Name)
		assert.True(t, p.Required,
			"path params must be required: %q", p.Name)
		names = append(names, p.Name)
	}

	// Exported fields promote (recursively); unexported "secret" and "token" must never surface —
	// the product documents only the public API surface.
	assert.ElementsMatch(t, []string{"vendorID", "urcapID", "version", "trace"}, names)
	assert.NotContains(t, names, "secret")
	assert.NotContains(t, names, "token")
}
