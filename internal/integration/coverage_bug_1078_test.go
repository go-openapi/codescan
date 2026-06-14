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

// TestCoverage_Bug1078 locks the fix for go-swagger issue #1078 ("fails to
// generate JSON schema for time fields"): a time.Time field is documented as
// {type: string, format: date-time}.
func TestCoverage_Bug1078(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1078/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	createdAt := doc.Definitions["Event"].Properties["createdAt"]
	require.Len(t, createdAt.Type, 1)
	assert.Equal(t, "string", createdAt.Type[0])
	assert.Equal(t, "date-time", createdAt.Format)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1078_schema.json")
}
