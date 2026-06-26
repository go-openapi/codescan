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

// TestQuirk_GofmtMetaYAML verifies the fix for quirk F7 (doc-site-quirks.md): a swagger:meta YAML
// block in its gofmt-canonical shape — a column-0 key, the blank "//" line gofmt inserts below
// it, then tab-indented children — now scans cleanly.
//
// Previously the tab indentation after the blank line reached the YAML sub-parser and aborted the
// scan with "found character that cannot start any token".
//
// Root: RemoveIndent keyed the dedent strip width off the literal first body line. gofmt's inserted
// blank line made that width zero, so the children's tabs survived.
// The fix keys the strip width off the first non-blank line, so the gofmt-canonical form dedents
// (and retabs) identically to the uniformly-indented form.
//
// This closes the residual behind #2959-class meta failures: write the natural column-0 form, run
// gofmt, still scans.
func TestQuirk_GofmtMetaYAML(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./quirks/gofmt-meta/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err, "the gofmt-canonical meta block must parse")
	require.NotNil(t, doc)

	// The securityDefinitions block parsed despite the gofmt blank line.
	require.Contains(t, doc.SecurityDefinitions, "oauth2")
	oauth2 := doc.SecurityDefinitions["oauth2"]
	require.NotNil(t, oauth2)
	assert.Equal(t, "oauth2", oauth2.Type)
	assert.Equal(t, "header", oauth2.In)
	assert.Equal(t, "application", oauth2.Flow)

	scantest.CompareOrDumpJSON(t, doc, "quirk_gofmt_meta.json")
}
