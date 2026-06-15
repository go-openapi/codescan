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

// TestCoverage_Bug2013 locks go-swagger issue #2013 ("panic: index out of range"
// in SetEnum during buildEmbedded): an enum on a promoted (embedded) field is
// parsed without panicking and the enum values are carried onto the composed
// model.
func TestCoverage_Bug2013(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2013/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	thing := doc.Definitions["Thing"]
	kind := thing.Properties["kind"]
	assert.Equal(t, []any{"a", "b", "c"}, kind.Enum, "embedded field enum is promoted")
	status := thing.Properties["status"]
	assert.Equal(t, []any{"open", "closed"}, status.Enum)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2013_schema.json")
}
