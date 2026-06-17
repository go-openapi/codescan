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

// TestCoverage_Bug1913 locks go-swagger issue #1913 ("sub-types not generated
// from go discriminated type"): an interface model with `discriminator: true`
// emits a base definition carrying `discriminator`, and the struct subtypes
// that embed it via `swagger:allOf` emit `allOf: [{$ref base}, {own props}]`.
//
// (The reporter's residual concern — subtypes require -m, which over-generates
// — is tracked as forthcoming-features §12 (prune under -m) and §15
// (auto-discover discriminator subtypes of a referenced base).)
func TestCoverage_Bug1913(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1913/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	base := doc.Definitions["TeslaCar"]
	assert.Equal(t, "model", base.Discriminator, "the base carries the discriminator")

	for _, name := range []string{"modelS", "modelX"} {
		sub, ok := doc.Definitions[name]
		require.True(t, ok, "subtype %s must be emitted", name)
		require.Len(t, sub.AllOf, 2, "subtype %s is allOf[base, own]", name)
		assert.Equal(t, "#/definitions/TeslaCar", sub.AllOf[0].Ref.String())
	}

	scantest.CompareOrDumpJSON(t, doc, "bugs_1913_schema.json")
}
