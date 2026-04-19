// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scantest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// CompareOrDumpJSON marshals got to stable JSON and either writes it to
// <repo>/fixtures/integration/golden/<name> (when UPDATE_GOLDEN=1) or
// asserts that it JSON-equals the stored golden.
//
// This is the regression-testing harness used to detect any behavior change
// in the go-openapi/spec objects produced by the scanner, compared against
// a captured baseline.
//
// Golden files are named by content (fixture bundle + object kind + entity),
// not by test name, so they survive test reshuffling.
func CompareOrDumpJSON(t *testing.T, got any, name string) {
	t.Helper()

	data, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)

	path := filepath.Join(goldenDir(), name)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		const (
			dirPerm  = 0o700
			filePerm = 0o600
		)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), dirPerm))
		require.NoError(t, os.WriteFile(path, data, filePerm))
		t.Logf("wrote golden %s", name)
		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err, "missing golden %s — run with UPDATE_GOLDEN=1 to create", name)
	assert.JSONEqT(t, string(want), string(data))
}

// goldenDir returns the absolute path to the repo-level golden directory.
func goldenDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("scantest: unable to resolve caller for golden path")
	}
	// thisFile is <repo>/internal/scantest/golden.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "fixtures", "integration", "golden"))
}
