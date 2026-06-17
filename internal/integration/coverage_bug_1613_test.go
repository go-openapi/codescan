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

// TestCoverage_Bug1613 locks the fix for go-swagger issue #1613 ("generate spec
// for response with string content type"): a swagger:response with a plain string
// in:body field used to emit a response with no schema (description only). It now
// emits schema {type: string} (the primitive-body work, #2942).
func TestCoverage_Bug1613(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1613/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["serviceStatusResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema, "a plain string body must still produce a schema")
	require.Len(t, resp.Schema.Type, 1)
	assert.Equal(t, "string", resp.Schema.Type[0])

	scantest.CompareOrDumpJSON(t, doc, "bugs_1613_schema.json")
}
