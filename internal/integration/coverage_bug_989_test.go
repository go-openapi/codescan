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

// TestCoverage_Bug989 locks the resolution to go-swagger issue #989 ("add description to
// response"): the inline description text is no longer misinterpreted as a model $ref, and an
// inline response description is fully captured via the swagger:operation YAML body (403 ->
// "Unauthorized").
//
// 📖 Need doc: in the legacy swagger:route Responses block a status code maps to a response name;
// an inline description value there is NOT captured (the 403 gets an empty description).
// To add a description, use swagger:operation or a dedicated swagger:response object.
func TestCoverage_Bug989(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/989/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	// swagger:operation captures the inline response description.
	adm := doc.Paths.Paths["/admins"].Get
	require.NotNil(t, adm)
	assert.Equal(t, "Unauthorized", adm.Responses.StatusCodeResponses[403].Description)

	// swagger:route maps codes to response names; 403 has no model and the inline description value is
	// dropped (documented limitation).
	usr := doc.Paths.Paths["/users"].Get
	require.NotNil(t, usr)
	r200 := usr.Responses.StatusCodeResponses[200]
	assert.Equal(t, "#/responses/listResponse", r200.Ref.String(),
		"code maps to a response name, not a broken ref to the description text")
	assert.Empty(t, usr.Responses.StatusCodeResponses[403].Description,
		"the swagger:route block drops the inline description value")

	scantest.CompareOrDumpJSON(t, doc, "bugs_989_schema.json")
}
