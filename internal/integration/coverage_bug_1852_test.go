// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug1852 locks go-swagger issue #1852 ("Schema error for delete
// operation"): a 204 response with no description comment must still carry the
// OAS2-required `description` key. The legacy engine omitted it entirely, so
// editor.swagger.io rejected the spec ("missingProperty: description"); codescan
// now always emits the key (empty string when no doc comment is present).
func TestCoverage_Bug1852(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1852/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["noContentResponse"]
	require.True(t, ok)

	// The description key must be emitted even though the response type has no
	// doc comment (presence is what OAS2 validation requires).
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(b), `"description"`),
		"the OAS2-required description key must be present (go-swagger#1852)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1852_schema.json")
}
