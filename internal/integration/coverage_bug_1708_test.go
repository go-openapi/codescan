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

// TestCoverage_Bug1708 locks the fix for go-swagger issue #1708 ("response annotation missing
// property type", follow-up of #619): a body response whose fields are pointers to other models no
// longer errors with "missing property type" — the fields resolve to $refs in an object schema.
func TestCoverage_Bug1708(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1708/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["eventsResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	require.Len(t, resp.Schema.Type, 1)
	assert.Equal(t, "object", resp.Schema.Type[0])
	meta := resp.Schema.Properties["meta"]
	data := resp.Schema.Properties["data"]
	assert.Equal(t, "#/definitions/ListResponseMeta", meta.Ref.String())
	assert.Equal(t, "#/definitions/ListResponseData", data.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_1708_schema.json")
}
