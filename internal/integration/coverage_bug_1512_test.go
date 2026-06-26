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

// TestCoverage_Bug1512 documents the resolution to go-swagger issue #1512 ("invalid integer formats
// (uint64)"): codescan emits the Go-specific uint formats (uint64 / uint32) by design — a vendor
// convention that round-trips back to Go.
//
// Works as designed.
// For OAI-conformant / precision-safe output, a field is overridden with `swagger:strfmt int64`,
// which yields the string-encoded {type: string, format: int64} representation.
//
// 📖 Need doc: the uint* formats are intentional; use swagger:strfmt int64 to emit a conformant
// string-encoded int64.
func TestCoverage_Bug1512(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1512/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	props := doc.Definitions["Sizes"].Properties

	big := props["big"]
	require.Len(t, big.Type, 1)
	assert.Equal(t, "integer", big.Type[0])
	assert.Equal(t, "uint64", big.Format, "default vendor format for uint64")
	assert.Equal(t, "uint32", props["small"].Format)
	assert.Equal(t, "int64", props["signed"].Format)

	// The documented override: swagger:strfmt int64 -> string-encoded int64.
	ov := props["overridden"]
	require.Len(t, ov.Type, 1)
	assert.Equal(t, "string", ov.Type[0], "swagger:strfmt overrides to a string-encoded representation")
	assert.Equal(t, "int64", ov.Format)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1512_schema.json")
}
