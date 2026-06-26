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

// TestCoverage_Bug1828 locks go-swagger issue #1828 ("route responses tag description"): the inline
// `NNN: description:<text>` form inside a swagger:route Responses block must produce a valid
// response with that description.
//
// The legacy engine emitted an invalid `$ref: '#/responses/'` (empty name); grammar2 parses the
// inline description correctly.
func TestCoverage_Bug1828(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1828/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/cats"].Delete
	require.NotNil(t, op)
	r200, ok := op.Responses.StatusCodeResponses[200]
	require.True(t, ok)
	assert.Equal(t, "Message Success", r200.Description)
	// No spurious $ref: the response is described inline, not referenced.
	assert.Empty(t, r200.Ref.String(), "must not emit an empty $ref (the legacy bug)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1828_schema.json")
}
