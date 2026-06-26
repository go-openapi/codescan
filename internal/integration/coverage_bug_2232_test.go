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

// TestCoverage_Bug2232 locks go-swagger issue #2232 ("tags with spaces"): a multi-word tag is
// expressed via the swagger:operation YAML body `tags:` list. (The swagger:route line cannot carry
// spaced tags — its tags are space-delimited tokens — so the YAML form is the supported route
// to it.)
func TestCoverage_Bug2232(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2232/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	assert.Equal(t, []string{"Thing Management"}, op.Tags)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2232_schema.json")
}
