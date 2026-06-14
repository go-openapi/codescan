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

// TestCoverage_Bug1609 locks the resolution to go-swagger issue #1609 ("vendor
// extension x-example"): x-* vendor extensions are emitted on both parameters and
// response headers when declared via an Extensions: block (mirroring
// swagger:meta's InfoExtensions / Extensions). Both round-trip into the marshaled
// spec.
//
// 📖 Need doc: a bare `// x-example: …` line is swallowed as the description; the
// supported form is an Extensions: block on the parameter / header.
func TestCoverage_Bug1609(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1609/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	// Parameter Extensions:.
	p := doc.Paths.Paths["/users/{user_id}"].Delete.Parameters[0]
	assert.Equal(t, "USERID", p.Extensions["x-example"],
		"parameter Extensions: emits x-example")

	// Response-header Extensions:.
	hdr := doc.Responses["statusResponse"].Headers["X-Rate-Limit"]
	assert.Equal(t, "requests-per-minute", hdr.Extensions["x-units"],
		"response-header Extensions: emits x-units")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1609_schema.json")
}
