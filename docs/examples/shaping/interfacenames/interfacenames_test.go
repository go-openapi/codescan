// SPDX-License-Identifier: Apache-2.0

package interfacenames

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

// scanInterfaceNames scans the witness with SkipJSONifyInterfaceMethods off or on.
func scanInterfaceNames(t *testing.T, skip bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:                     examplesRoot(t),
		Packages:                    []string{"./shaping/interfacenames/..."},
		ScanModels:                  true,
		SkipJSONifyInterfaceMethods: skip,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenRaw marshals an arbitrary value into testdata/<feature>.json,
// honouring UPDATE_GOLDEN.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenRaw(t *testing.T, feature string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	golden := filepath.Join("testdata", feature+".json")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(golden, got, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}

// TestInterfaceNames_Off is the control: with the flag off interface-method
// names auto-jsonify (ID → id), matching the historic behaviour.
func TestInterfaceNames_Off(t *testing.T) {
	doc := scanInterfaceNames(t, false)
	acct, ok := doc.Definitions["Account"]
	require.True(t, ok, "Account definition missing")
	goldenRaw(t, "account_off", acct)

	props := acct.Properties
	for _, name := range []string{"id", "createdAt"} {
		_, has := props[name]
		assert.True(t, has, "default path must auto-jsonify to %q when off", name)
	}
	for _, name := range []string{"ID", "CreatedAt"} {
		_, has := props[name]
		assert.False(t, has, "verbatim Go name %q must not appear when off", name)
	}
	_, has := props["explicit_name"]
	assert.True(t, has, "swagger:name override must stay verbatim")
}

// TestInterfaceNames_On exercises SkipJSONifyInterfaceMethods: default-path
// methods are emitted verbatim while the swagger:name override is unchanged.
func TestInterfaceNames_On(t *testing.T) {
	doc := scanInterfaceNames(t, true)
	acct, ok := doc.Definitions["Account"]
	require.True(t, ok, "Account definition missing")
	goldenRaw(t, "account_on", acct)

	props := acct.Properties
	for _, name := range []string{"ID", "CreatedAt"} {
		_, has := props[name]
		assert.True(t, has, "opt-out must emit the Go name %q verbatim", name)
	}
	for _, name := range []string{"id", "createdAt"} {
		_, has := props[name]
		assert.False(t, has, "mangled spelling %q must not appear when opted out", name)
	}
	_, has := props["explicit_name"]
	assert.True(t, has, "swagger:name override must stay verbatim")
	_, remangled := props["explicitName"]
	assert.False(t, remangled, "the override must not be re-mangled")
}
