// SPDX-License-Identifier: Apache-2.0

package nameconflicts_test

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

// scanTree scans the whole nameconflicts package tree so the reduce stage sees
// every colliding identity at once — deconfliction is a pure function of the
// reachable model set, not of discovery order.
func scanTree(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/nameconflicts/..."},
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// goldenJSON marshals one definition and compares it to (or, under
// UPDATE_GOLDEN, rewrites) testdata/<feature>.json.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func goldenJSON(t *testing.T, doc *spec.Swagger, feature, defName string) {
	t.Helper()
	schema, ok := doc.Definitions[defName]
	require.Truef(t, ok, "definition %q not found", defName)
	got, err := json.MarshalIndent(schema, "", "  ")
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

// TestDeconfliction emits the golden fragments the how-to renders and asserts
// the central claims: cross-package collisions are qualified by segment,
// same-package duplicates fall back to the Go name, and a cross-package
// type-name keyword resolves a unique leaf to a $ref.
func TestDeconfliction(t *testing.T) {
	doc := scanTree(t)

	goldenJSON(t, doc, "dashboard", "Dashboard")
	goldenJSON(t, doc, "billingaccount", "BillingAccount")
	goldenJSON(t, doc, "bag", "Bag")

	defs := doc.Definitions

	// The two same-named Accounts never merge onto a bare "Account": each is
	// qualified with its package segment.
	assert.NotContains(t, defs, "Account")
	require.Contains(t, defs, "BillingAccount")
	require.Contains(t, defs, "IdentityAccount")

	// Distinct fields prove they did not union into one definition.
	require.Contains(t, defs["BillingAccount"].Properties, "balance")
	require.Contains(t, defs["IdentityAccount"].Properties, "email")

	// Dashboard's refs point at the resolved names, never a bare "Account".
	billingRef := defs["Dashboard"].Properties["billing"]
	assert.Equal(t, "#/definitions/BillingAccount", billingRef.Ref.String())
	identityRef := defs["Dashboard"].Properties["identity"]
	assert.Equal(t, "#/definitions/IdentityAccount", identityRef.Ref.String())

	// Same-package duplicate: one keeps "Entry"; the loser reverts to its Go
	// name "Reversal". Both exist, distinct — never a silent merge.
	require.Contains(t, defs, "Entry")
	require.Contains(t, defs, "Reversal")
	assert.NotContains(t, defs, "Posting")
	primaryRef := defs["Dashboard"].Properties["primary"]
	assert.Equal(t, "#/definitions/Entry", primaryRef.Ref.String())
	secondaryRef := defs["Dashboard"].Properties["secondary"]
	assert.Equal(t, "#/definitions/Reversal", secondaryRef.Ref.String())

	// Cross-package type-name keyword: the unique leaf "Widget" resolves to a
	// $ref on Bag's additionalProperties.
	bag := defs["Bag"]
	require.NotNil(t, bag.AdditionalProperties)
	require.NotNil(t, bag.AdditionalProperties.Schema)
	assert.Equal(t, "#/definitions/Widget", bag.AdditionalProperties.Schema.Ref.String())
}
