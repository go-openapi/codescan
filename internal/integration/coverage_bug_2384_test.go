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

// TestCoverage_Bug2384 locks go-swagger issue #2384 ("pattern containing '\n' is interpreted"):
// codescan reads a `pattern:` value VERBATIM — the backslash escapes `\t` / `\n` are preserved as
// the two characters they are, not turned into literal tab/newline. (The reporter's broken
// multi-line output came from go-swagger's spec->code generator, not from codescan's code->spec
// scan.)
func TestCoverage_Bug2384(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2384/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	name := doc.Definitions["Role"].Properties["name"]
	assert.Equal(t, `^[^ \t\n](.*[^ \t\n])*$`, name.Pattern,
		"the pattern's backslash escapes are preserved verbatim (go-swagger#2384)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2384_schema.json")
}
