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
// section, so it is invalid on its own, and an -input overlay shows up against it.
func Unannotated(t *testing.T) string {
	t.Helper()

	return filepath.Join(fixtures(t), "enhancements", "additional-properties")
}

// configFileMode is the permission a configuration file is written with: the command reads it and
// nobody else needs to.
const configFileMode = 0o600

// ConfigFile writes a configuration file in a fresh directory and reports its path.
func ConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), cliconf.Names[0])
	require.NoError(t, os.WriteFile(path, []byte(content), configFileMode))

	return path
}

// fixtures locates the repository's corpus from this file, rather than from the working directory,
// which `go test` is free to set to wherever the package lives.
func fixtures(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve this test's own path")

	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "testdata"))
}
