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

// TestCoverage_Bug2407 locks go-swagger issue #2407 ("how add example to yml"): an `example: [1, 2,
// 3]` on an array-typed body response is emitted on the response schema (previously absent).
func TestCoverage_Bug2407(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2407/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	sch := doc.Responses["encryptSuccessResponse"].Schema
	require.NotNil(t, sch)
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, sch.Example,
		"the array example is emitted on the response schema (go-swagger#2407)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2407_schema.json")
}
