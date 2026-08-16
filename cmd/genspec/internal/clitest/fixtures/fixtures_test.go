// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// These locate the corpus by counting directories up from this file, so a package that moves takes the count with it -
// and every test that scans a fixture would then fail at once, saying nothing about where the tree went.
//
// Which is what this is for: it fails here instead, next to the path that needs adjusting.
func TestFixtures(t *testing.T) {
	t.Parallel()

	t.Run("should point at the petstore", func(t *testing.T) {
		t.Parallel()

		dir := Petstore(t)

		assert.DirExists(t, dir)
		assert.FileExists(t, filepath.Join(dir, "doc.go"), "the meta block the petstore is scanned for")
	})

	t.Run("should point at a tree with nothing to say about itself", func(t *testing.T) {
		// Its being unannotated is what makes it useful: no info section, so the document it produces is invalid
		// on its own and an -input overlay shows.
		t.Parallel()

		assert.DirExists(t, Unannotated(t))
	})

	t.Run("should write a configuration file where the search will find it", func(t *testing.T) {
		t.Parallel()

		path := ConfigFile(t, "document:\n  compact: true\n")

		assert.Equal(t, cliconf.Names[0], filepath.Base(path), "under the name the command looks for")

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "document:\n  compact: true\n", string(content))
	})
}
