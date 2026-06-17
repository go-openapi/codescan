// SPDX-License-Identifier: Apache-2.0

package genericenvelopes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func scanGenericEnvelopes(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/genericenvelopes"},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// writeGolden marshals v and compares it against testdata/<name>, regenerating
// when UPDATE_GOLDEN is set.
func writeGolden(t *testing.T, name string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}

// TestGenericEnvelopes emits and verifies the golden files the "Documenting
// generic responses" guide renders:
//
//   - apiresponse.json   — the generic envelope, with data as an open schema;
//   - statusenvelope.json — the doc-only embed+shadow envelope, with data as a
//     concrete $ref;
//   - paths.json         — the two routes, each pointing at its doc-only
//     envelope.
//
// It asserts the central claim: embedding the generic envelope and shadowing
// Data turns the opaque field into a concrete $ref.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestGenericEnvelopes(t *testing.T) {
	doc := scanGenericEnvelopes(t)
	require.NotNil(t, doc.Definitions)

	apiResp, ok := doc.Definitions["APIResponse"]
	require.True(t, ok, "APIResponse definition missing")
	// Baseline: the generic envelope's data is an open schema (no type, no $ref).
	dataProp := apiResp.Properties["data"]
	assert.Empty(t, dataProp.Type, "generic data has no concrete type")
	assert.Empty(t, dataProp.Ref.String(), "generic data is not a $ref")

	statusEnv, ok := doc.Definitions["StatusEnvelope"]
	require.True(t, ok, "StatusEnvelope definition missing")
	// The doc-only envelope: status/message are promoted from the embed, and the
	// shadowing data is now a concrete $ref to StatusReport.
	assert.Contains(t, statusEnv.Properties, "status")
	assert.Contains(t, statusEnv.Properties, "message")
	statusData := statusEnv.Properties["data"]
	assert.Equal(t, "#/definitions/StatusReport",
		statusData.Ref.String(),
		"shadowed data resolves to the concrete payload")

	require.NotNil(t, doc.Paths)
	require.Contains(t, doc.Paths.Paths, "/status")
	require.Contains(t, doc.Paths.Paths, "/users/{id}")

	writeGolden(t, "apiresponse.json", apiResp)
	writeGolden(t, "statusenvelope.json", statusEnv)
	writeGolden(t, "paths.json", doc.Paths)
}
