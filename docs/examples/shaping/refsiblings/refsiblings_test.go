// SPDX-License-Identifier: Apache-2.0

package refsiblings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/require"
)

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func scan(t *testing.T, opts codescan.Options) *spec.Swagger {
	t.Helper()
	opts.WorkDir = examplesRoot(t)
	opts.Packages = []string{"./shaping/refsiblings"}
	opts.ScanModels = true
	doc, err := codescan.Run(&opts)
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenHome marshals the Person.home property and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenHome(t *testing.T, doc *spec.Swagger, feature string) {
	t.Helper()
	person, ok := doc.Definitions["Person"]
	require.True(t, ok, "Person definition missing")
	home, ok := person.Properties["home"]
	require.True(t, ok, "home property missing")
	got, err := json.MarshalIndent(home, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", feature+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	require.JSONEq(t, string(want), string(got))
}

// TestRefSiblings emits the three fragments rendered by the how-to: the default
// allOf wrap, the EmitRefSiblings direct-sibling shape, and the bare $ref left
// by SkipAllOfCompounding.
func TestRefSiblings(t *testing.T) {
	// Default: extension present → single-arm allOf wrap; description and x-*
	// ride the outer compound.
	goldenHome(t, scan(t, codescan.Options{}), "default")

	// EmitRefSiblings: description and x-* ride directly beside the $ref.
	goldenHome(t, scan(t, codescan.Options{EmitRefSiblings: true}), "siblings")

	// SkipAllOfCompounding: no compound is produced, so the description and
	// extension are dropped and a bare $ref remains.
	goldenHome(t, scan(t, codescan.Options{SkipAllOfCompounding: true}), "skip")
}
