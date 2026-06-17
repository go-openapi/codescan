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

// TestCoverage_Bug2286 locks go-swagger issue #2286 ("model accepted as
// response"): naming a swagger:model directly as a route response (`200: model`)
// is accepted and yields a valid response with a $ref to that model — the lenient
// shorthand for `200: body:model`.
func TestCoverage_Bug2286(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2286/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	r200 := doc.Paths.Paths["/logout"].Get.Responses.StatusCodeResponses[200]
	require.NotNil(t, r200.Schema)
	assert.Equal(t, "#/definitions/logoutResponse", r200.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2286_schema.json")
}
