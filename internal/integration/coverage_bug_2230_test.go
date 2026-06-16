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

// TestCoverage_Bug2230 locks go-swagger issue #2230 ("example in json.RawMessage
// field"): a json.RawMessage field renders as an open (typeless) schema — i.e.
// "any JSON", which matches RawMessage's meaning — while a typed sibling field
// carries its example. To constrain or example the RawMessage, annotate it
// (swagger:type / swagger:strfmt); the raw bytes are not forced to an int array.
func TestCoverage_Bug2230(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2230/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["SearchObject"].Properties
	assert.Equal(t, 200, props["limit"].Example, "typed field keeps its example")
	search := props["search"]
	assert.Empty(t, search.Type, "json.RawMessage is an open (any-JSON) schema (go-swagger#2230)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2230_schema.json")
}
