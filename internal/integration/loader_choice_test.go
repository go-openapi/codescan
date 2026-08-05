// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// scanPetstore runs the petstore fixture through whichever loader opts selects.
func scanPetstore(t *testing.T, toolchainFree bool) []byte {
	t.Helper()

	doc, err := codescan.Run(&codescan.Options{
		Packages:            []string{"./petstore/..."},
		WorkDir:             scantest.FixturesDir() + "/goparsing",
		ScanModels:          true,
		ToolchainFreeLoader: toolchainFree,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	out, err := json.Marshal(doc)
	require.NoError(t, err)

	return out
}

// TestLoaderChoice_AgreeOnTheRealFilesystem is the point of ToolchainFreeLoader having its own flag:
// before it, the only way to reach codescan's loader was to hand it a virtual filesystem, so it could
// never be run against an ordinary tree on disk.
//
// The two loaders are meant to be substitutable, so the assertion is equality of the whole document
// rather than a spot check — anything less would pass while a property quietly went missing.
//
// Equality alone cannot tell that the flag was honoured: were it a no-op, both sides would run the
// same loader and agree trivially. TestLoaderChoice_SelectsTheLoader is what witnesses the routing.
func TestLoaderChoice_AgreeOnTheRealFilesystem(t *testing.T) {
	t.Parallel()

	assert.JSONEq(t, string(scanPetstore(t, false)), string(scanPetstore(t, true)),
		"the toolchain-free loader must produce the same spec as go/packages")
}

// TestLoaderChoice_SelectsTheLoader witnesses which loader actually ran, through a behaviour only one
// of them has: StubStdlib is the toolchain-free loader's, and the go/packages path documents that it
// ignores it. So the same options that raise synthesized-import diagnostics under one loader must
// raise none under the other.
func TestLoaderChoice_SelectsTheLoader(t *testing.T) {
	t.Parallel()

	synthesized := func(toolchainFree bool) int {
		count := 0
		_, err := codescan.Run(&codescan.Options{
			Packages:            []string{"./petstore/models/..."},
			WorkDir:             scantest.FixturesDir() + "/goparsing",
			ScanModels:          true,
			ToolchainFreeLoader: toolchainFree,
			StubStdlib:          true,
			OnDiagnostic: func(d codescan.Diagnostic) {
				if d.Code == "scan.synthesized-import" {
					count++
				}
			},
		})
		require.NoError(t, err)

		return count
	}

	assert.Positive(t, synthesized(true),
		"withholding the stdlib must be visible: the toolchain-free loader synthesizes what it cannot read")
	assert.Zero(t, synthesized(false),
		"go/packages has no such mode, so StubStdlib is inert there — which is how we know it was not used")
}

// TestLoaderChoice_DefaultsToGoPackages pins the default, since the whole feature is opt-in: a caller
// that says nothing must keep the historic loader.
func TestLoaderChoice_DefaultsToGoPackages(t *testing.T) {
	t.Parallel()

	var opts codescan.Options

	assert.False(t, opts.ToolchainFreeLoader, "the zero value scans with the go command")
	assert.Nil(t, opts.FS, "and against the real filesystem")
}
