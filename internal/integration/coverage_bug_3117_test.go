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

// TestCoverage_Bug3117 locks go-swagger issue #3117 ("spec generating type
// property along with schema references"): a body parameter whose schema is a
// $ref must NOT also carry a sibling `type: object` (which is invalid Swagger
// 2.0). The body param schema is a clean $ref.
func TestCoverage_Bug3117(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3117/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/it"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	sch := op.Parameters[0].Schema
	require.NotNil(t, sch)
	assert.Equal(t, "#/definitions/Payload", sch.Ref.String())
	assert.Empty(t, sch.Type, "a $ref body schema must not carry a sibling type (go-swagger#3117)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3117_schema.json")
}
