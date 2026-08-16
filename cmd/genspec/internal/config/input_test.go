// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/codescan/cmd/genspec/internal/sentinel"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// -input is where everything a scanner cannot know comes from, so every way of not getting it says which of them
// happened: the document is the caller's, and so is the mistake.
func TestLoadInputSpec(t *testing.T) {
	t.Parallel()

	t.Run("should load a document to merge into", func(t *testing.T) {
		t.Parallel()

		doc, err := loadInputSpec(inputFile(t, `{"swagger":"2.0","host":"example.com"}`))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, "example.com", doc.Host)
	})

	t.Run("should refuse one that is not there", func(t *testing.T) {
		t.Parallel()

		_, err := loadInputSpec(filepath.Join(t.TempDir(), "absent.json"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot read -input")
	})

	t.Run("should refuse a directory", func(t *testing.T) {
		// Which is a usage error rather than a failed load: nothing was wrong with a document, because there was
		// no document.
		t.Parallel()

		_, err := loadInputSpec(t.TempDir())

		require.ErrorIs(t, err, sentinel.ErrUsage)
		assert.Contains(t, err.Error(), "not a specification")
	})

	t.Run("should refuse one it cannot read as a specification", func(t *testing.T) {
		t.Parallel()

		_, err := loadInputSpec(inputFile(t, "this is not a specification"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot load -input")
	})
}

func inputFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "base.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}
