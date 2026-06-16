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

// TestCoverage_Bug622 locks go-swagger issue #622 ("response example objects"): an
// `example:` on a response body field is emitted on the response schema. (The
// reporter's `exampleValue:` was non-standard; `example:` is the keyword.)
func TestCoverage_Bug622(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/622/..."}, WorkDir: scantest.FixturesDir()})
	require.NoError(t, err)
	name := doc.Responses["personResponse"].Schema.Properties["name"]
	assert.Equal(t, "Bob", name.Example, "example on a response body field is emitted (go-swagger#622)")
	scantest.CompareOrDumpJSON(t, doc, "bugs_622_schema.json")
}
