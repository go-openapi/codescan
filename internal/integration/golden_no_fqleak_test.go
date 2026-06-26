// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// fqDefKey matches a fully-qualified definition KEY (an object-valued member whose name is a Go
// import path), e.g. `"github.com/acme/foo/Bar": {`.
//
// The x-go-package extension carries an import path too, but as a STRING value (`"x-go-package":
// "github.com/..."`), which this pattern does not match.
var fqDefKey = regexp.MustCompile(`"[a-z0-9.-]+\.[a-z]{2,}/[^"]*":\s*\{`)

// TestGoldens_NoFullyQualifiedLeak locks the invariant that every committed golden is the REDUCED,
// full-pipeline output: no definition key or $ref may carry a fully-qualified `<pkgpath>/<name>`
// identity.
//
// The build keys definitions by that fq identity, and the reduce stage shortens them; a leak here
// means either a builder-unit snapshot crept back in (golden dumps belong to full-pipeline
// integration tests) or the reduce rewrite pass missed a site.
func TestGoldens_NoFullyQualifiedLeak(t *testing.T) {
	dir := filepath.Join(scantest.FixturesDir(), "integration", "golden")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			require.NoError(t, err)
			content := string(raw)

			assert.NotContains(t, content, "#/definitions/github.com",
				"fully-qualified $ref leaked — golden is not reduced full-pipeline output")
			assert.NotRegexp(t, fqDefKey, content,
				"fully-qualified definition key leaked — golden is not reduced full-pipeline output")
		})
	}
}
