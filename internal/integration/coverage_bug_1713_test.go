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

// TestCoverage_Bug1713 locks the answer to go-swagger issue #1713 ("how to label
// the example Response & Request"): response `examples:` keyed by mime type are
// supported in the swagger:operation YAML body.
//
// (The struct-based swagger:response example-by-mime form is the separate
// forthcoming feature §10 / #2871.)
func TestCoverage_Bug1713(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1713/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/test"].Get
	require.NotNil(t, op)
	r200 := op.Responses.StatusCodeResponses[200]
	ex, ok := r200.Examples["application/json"]
	require.True(t, ok, "examples keyed by mime type are emitted")
	exMap, ok := ex.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "blah", exMap["test"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1713_schema.json")
}
