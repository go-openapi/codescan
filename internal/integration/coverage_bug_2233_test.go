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

// TestCoverage_Bug2233 locks go-swagger issue #2233 ("unable to find responses defined in other
// package using swagger:operation"): a swagger:operation YAML body that `$ref`s a swagger:response
// defined in ANOTHER package resolves — the response and its body model are emitted (no "$refs
// must reference a valid location").
func TestCoverage_Bug2233(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2233/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	r200 := doc.Paths.Paths["/user"].Get.Responses.StatusCodeResponses[200]
	assert.Equal(t, "#/responses/userResponse", r200.Ref.String())
	resp, ok := doc.Responses["userResponse"]
	require.True(t, ok, "the cross-package response is collected")
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "#/definitions/User", resp.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2233_schema.json")
}
