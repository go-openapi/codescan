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

// TestCoverage_Bug1121 locks the answer to go-swagger issue #1121 ("how to add description to
// tags"): a `Tags:` block in the swagger:meta annotation emits the OAI root-level tags section with
// per-tag description (and externalDocs).
//
// 📖 Need doc: document the swagger:meta Tags: block for tag descriptions.
func TestCoverage_Bug1121(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1121/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.Len(t, doc.Tags, 1)
	assert.Equal(t, "users", doc.Tags[0].Name)
	assert.Equal(t, "Operations about users", doc.Tags[0].Description)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1121_schema.json")
}
