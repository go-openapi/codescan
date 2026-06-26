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

// TestCoverage_Bug2062 locks go-swagger issue #2062 (the security half): a `security:` requirement
// in the swagger:operation YAML body is emitted on the operation. (SecurityDefinitions is a global
// swagger:meta concept in OpenAPI 2.0, not a per-operation field — that part of the ask is N/A by
// spec.)
func TestCoverage_Bug2062(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2062/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/user"].Put
	require.NotNil(t, op)
	require.Len(t, op.Security, 1, "operation-level security requirement is emitted (go-swagger#2062)")
	_, ok := op.Security[0]["api_key"]
	assert.True(t, ok)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2062_schema.json")
}
