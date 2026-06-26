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

// TestCoverage_Bug2317 locks go-swagger issue #2317 ("x-nullable on a pointer has no effect"): an
// `x-nullable: true` Extensions block on a pointer field is applied — the field becomes
// allOf[$ref] carrying x-nullable.
func TestCoverage_Bug2317(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2317/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	friend := doc.Definitions["Person"].Properties["friend"]
	assert.Equal(t, true, friend.Extensions["x-nullable"], "x-nullable applies on a pointer field (go-swagger#2317)")
	require.Len(t, friend.AllOf, 1)
	assert.Equal(t, "#/definitions/Friend", friend.AllOf[0].Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2317_schema.json")
}
