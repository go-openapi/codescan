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

// TestCoverage_Bug3069 demonstrates the resolution to go-swagger issue #3069 ("change the
// representation of one parameter"): a custom-marshalled struct cannot be inferred by codescan, but
// a `swagger:type` override states its representation explicitly.
//
// ComplexType -> string makes a []ComplexType field render as an array of strings, matching the
// actual wire format. (📖 Need doc: document this override for custom-marshalled types.)
func TestCoverage_Bug3069(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3069/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	ct := doc.Definitions["MyResponse"].Properties["complexType"]
	assert.Equal(t, "array", ct.Type[0])
	require.NotNil(t, ct.Items)
	require.NotNil(t, ct.Items.Schema)
	assert.Equal(t, "string", ct.Items.Schema.Type[0],
		"the swagger:type override must drive the item representation")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3069_schema.json")
}
