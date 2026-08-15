// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package fixtures

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan/cmd/internal/cliconf"
	"github.com/go-openapi/testify/v2/require"
)

// Petstore is the corpus's complete example: a meta block, paths, parameters, responses and models.
// A document produced from it exercises every part of this command that is not about flags.
func Petstore(t *testing.T) string {
	t.Helper()

	return filepath.Join(fixtures(t), "goparsing", "petstore")
}

// Unannotated is a fixture with no swagger:meta block, so the document it produces has no info
// section - which is what makes it invalid on its own, and what makes an -input overlay visible.
func Unannotated(t *testing.T) string {
	t.Helper()

	return filepath.Join(fixtures(t), "enhancements", "additional-properties")
}

// ConfigFile writes a configuration file in a fresh directory and reports its path.
func ConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// fixtures locates the repository's corpus from this file, rather than from the working directory,
// which `go test` is free to set to wherever the package lives.
func fixtures(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve this test's own path")

	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "fixtures"))
}
