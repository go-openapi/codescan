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

// TestCoverage_Bug2296 locks go-swagger issue #2296 ("panic, embedded meet
// anonymous"): embedding a struct that itself has an anonymous-struct field no
// longer panics during schema build; the promoted anonymous-struct property is
// emitted as a nested object.
func TestCoverage_Bug2296(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2296/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "embedded + anonymous struct must not panic (go-swagger#2296)")

	c := doc.Definitions["C"]
	assert.Contains(t, c.Properties, "name")
	require.Contains(t, c.Properties, "inner", "promoted anonymous-struct field is present")
	assert.Equal(t, []string{"object"}, []string(c.Properties["inner"].Type))

	scantest.CompareOrDumpJSON(t, doc, "bugs_2296_schema.json")
}
