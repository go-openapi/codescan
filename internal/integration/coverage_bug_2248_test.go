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

// TestCoverage_Bug2248 locks go-swagger issue #2248 ("time.Duration in response's
// header"): a response header typed time.Duration resolves to {type:integer,
// format:int64} — it carries a type, so the spec is valid (the reporter saw
// "headers.<name>.type in body is required").
func TestCoverage_Bug2248(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2248/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	resp := doc.Responses["indexedNodes"]
	h, ok := resp.Headers["lastContact"]
	require.True(t, ok)
	assert.Equal(t, "integer", h.Type, "the time.Duration header has a resolved type (go-swagger#2248)")
	assert.Equal(t, "int64", h.Format)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2248_schema.json")
}
