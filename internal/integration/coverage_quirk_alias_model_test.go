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

// TestQuirk_AliasModelNoHang is the F9 regression lock (doc-site-quirks.md). A
// swagger:model-annotated Go type alias (type Price = Money, both
// swagger:model) once sent the scanner into an infinite loop when the alias
// was built — under the default (expand) and RefAliases modes;
// TransparentAliases dissolved the alias before the looping path. The hang is
// gone on the current base; this test keeps it gone across all three alias
// modes. A regression would resurface as a test timeout (CI runs with
// -timeout), not an assertion failure — that is the point of the lock.
func TestQuirk_AliasModelNoHang(t *testing.T) {
	cases := []struct {
		name       string
		opts       *codescan.Options
		priceIsRef bool // RefAliases chains a $ref; expand/transparent inline
	}{
		{"default", &codescan.Options{Packages: []string{"./quirks/alias-model/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true}, false},
		{"refAliases", &codescan.Options{Packages: []string{"./quirks/alias-model/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true, RefAliases: true}, true},
		{"transparentAliases", &codescan.Options{Packages: []string{"./quirks/alias-model/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true, TransparentAliases: true}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := codescan.Run(tc.opts)
			require.NoError(t, err, "the model-annotated alias must scan, not hang (F9)")
			require.NotNil(t, doc)

			require.Contains(t, doc.Definitions, "Money")
			require.Contains(t, doc.Definitions, "Price")
			require.Contains(t, doc.Definitions, "Invoice")

			price := doc.Definitions["Price"]
			if tc.priceIsRef {
				assert.Equal(t, "#/definitions/Money", price.Ref.String(),
					"RefAliases chains the alias to its target")
			} else {
				assert.Equal(t, []string{"object"}, []string(price.Type),
					"expand/transparent inline the alias structurally")
				assert.Contains(t, price.Properties, "cents")
				assert.Contains(t, price.Properties, "currency")
			}
		})
	}
}
