// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package codescan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

// Public-API smoke suite.
// Fixture-heavy tests live in internal/integration.

func TestApplication_DeprecatedDebugOption(t *testing.T) {
	// Options.Debug is a deprecated no-op (the legacy debug logger was retired in favour of
	// diagnostics).
	// Verify Run still accepts it without error and produces a spec.
	_, err := Run(&Options{
		Packages:   []string{"./goparsing/petstore/..."},
		WorkDir:    "fixtures",
		ScanModels: true,
		Debug:      true,
	})

	require.NoError(t, err)
}

func TestRun_InvalidWorkDir(t *testing.T) {
	// Exercises the Run() error path when package loading fails.
	_, err := Run(&Options{
		Packages: []string{"./..."},
		WorkDir:  "/nonexistent/directory",
	})

	require.Error(t, err)
}

func TestSetEnumDoesNotPanic(t *testing.T) {
	// Regression: ensure Run() does not panic on minimal source with an enum.
	dir := t.TempDir()

	src := `
	package failure

	// swagger:model Order
	type Order struct {
		State State ` + "`json:\"state\"`" + `
	}

	// State represents the state of an order.
	// enum: ["created","processed"]
	type State string
	`
	err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(src), 0o600)
	require.NoError(t, err)

	goMod := `
	module failure
	go 1.23`
	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600)
	require.NoError(t, err)

	_, err = Run(&Options{
		WorkDir:    dir,
		ScanModels: true,
	})

	require.NoError(t, err)
}
