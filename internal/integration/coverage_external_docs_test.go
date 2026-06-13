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

// TestCoverage_ExternalDocs covers two swagger:meta ExternalDocs edge cases
// (each package may carry only one swagger:meta, so they scan separately):
//   - the `external docs` alias spelling with a url-only block;
//   - an empty block, which must NOT emit a useless `externalDocs: {}`.
func TestCoverage_ExternalDocs(t *testing.T) {
	t.Run("aliased url-only", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages: []string{"./enhancements/external-docs/aliased/..."},
			WorkDir:  scantest.FixturesDir(),
		})
		require.NoError(t, err)
		require.NotNil(t, doc.ExternalDocs)
		assert.Equal(t, "https://aliased.example.org", doc.ExternalDocs.URL)
		assert.Empty(t, doc.ExternalDocs.Description)
	})

	t.Run("empty block skipped", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages: []string{"./enhancements/external-docs/empty/..."},
			WorkDir:  scantest.FixturesDir(),
		})
		require.NoError(t, err)
		assert.Nil(t, doc.ExternalDocs, "empty ExternalDocs block must not emit externalDocs: {}")
	})
}
