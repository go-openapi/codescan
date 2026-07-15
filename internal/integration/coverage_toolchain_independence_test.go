// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/require"
)

// jobCEnvMarker is set by the bespoke `toolchain-independence` CI job (job C).
//
// It signals that this process is a binary built against one Go toolchain and run against another,
// with GOROOT deliberately pointed away from the build-time install.
//
// See .github/workflows/toolchain-independence.yml.
const jobCEnvMarker = "CODESCAN_TC_JOBC"

// TestToolchainIndependence_FullScan is the wide, full-workflow guard (job C) for the
// TextMarshaler toolchain-independence fix.
//
// It only does real work when jobCEnvMarker=="1"; under a normal `go test ./...` run it skips,
// because the pre-condition it needs (build-time GOROOT absent at runtime) cannot be arranged
// in-process. When the marker is set it enforces two things in order:
//
//  1. Self-check: the GOROOT baked into this binary at build time (runtime.GOROOT()) does NOT
//     exist at runtime. This is what makes a green result meaningful — under the old go/importer
//     implementation, a scan in exactly this environment produced {type: object} for every
//     TextMarshaler type. If the baked path still exists (e.g. the CI runner pre-cached the build
//     toolchain at the same path), the divergence never happened and the test would be a false
//     green, so we fail loudly rather than skip.
//
//  2. Symptom: a full codescan.Run over the text-marshal fixture — which drives
//     packages.Load -> schema.go -> resolvers.IsTextMarshaler end to end — still renders the
//     TextMarshaler-bearing fields as strings.
func TestToolchainIndependence_FullScan(t *testing.T) {
	if os.Getenv(jobCEnvMarker) != "1" {
		t.Skipf("%s!=1: skipping the toolchain-divergence guard (only meaningful under CI job C)", jobCEnvMarker)
	}

	t.Run("self-check: build-time GOROOT is absent at runtime", func(t *testing.T) {
		// runtime.GOROOT() is deprecated exactly because it returns the (stale) build-time root when GOROOT is unset.
		// This is "not meaningful if the binary is copied to another machine".
		// That stale value is precisely the signal this guard needs: it is the path the old go/importer consulted,
		// and its absence at runtime is what reproduces the bug's environment.
		baked := runtime.GOROOT() //nolint:staticcheck // intentionally reading the stale build-time GOROOT to prove build!=run divergence
		t.Logf("baked runtime.GOROOT()=%q; runtime GOROOT env=%q", baked, os.Getenv("GOROOT"))
		require.NotEmpty(t, baked, "binary has no baked GOROOT — cannot assert divergence")

		_, err := os.Stat(baked)
		require.TrueT(t, os.IsNotExist(err),
			"build-time GOROOT %q still exists at runtime: the build!=run divergence did not happen, "+
				"so this job cannot distinguish the fix from the bug (masking). Ensure job C removes the "+
				"build toolchain and unsets GOROOT before running.", baked)
	})

	t.Run("symptom: TextMarshaler fields still render as strings", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./enhancements/text-marshal/..."},
			WorkDir:    scantest.FixturesDir(),
			ScanModels: true,
		})
		require.NoError(t, err, "full scan must still succeed (packages.Load uses the runtime toolchain)")
		require.NotNil(t, doc)

		device, ok := doc.Definitions["Device"]
		require.TrueT(t, ok, "Device definition should be discovered")

		// All three fields are TextMarshaler-implementing types; each must be a string, not object.
		for _, field := range []string{"id", "mac", "data"} {
			prop, ok := device.Properties[field]
			require.TrueT(t, ok, "Device.%s should be present", field)
			require.TrueT(t, prop.Type.Contains("string"),
				"Device.%s must be type:string (TextMarshaler), got %v — the toolchain-independent "+
					"IsTextMarshaler regressed", field, prop.Type)
		}
	})
}
