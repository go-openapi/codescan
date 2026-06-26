// SPDX-License-Identifier: Apache-2.0

package godoclinks

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

// scanGodocLinks scans the witness tree (the package + its inventory
// subpackage) with CleanGoDoc off or on.
func scanGodocLinks(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/godoclinks/..."},
		ScanModels: true,
		CleanGoDoc: on,
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

// TestGodocLinks_Off is the control: with CleanGoDoc off the godoc doc-link
// syntax is carried into the spec verbatim (byte-identical to the historic
// behaviour).
func TestGodocLinks_Off(t *testing.T) {
	doc := scanGodocLinks(t, false)
	gizmo, ok := doc.Definitions["gizmo"]
	require.True(t, ok, "gizmo definition missing")
	goldenRaw(t, "gizmo_off", gizmo)

	// Brackets and the reference-definition line are preserved when off.
	assert.Contains(t, gizmo.Title, "[Gadget]")
	assert.Contains(t, gizmo.Title, "[Order.CustName]")
	assert.Contains(t, gizmo.Properties["spec"].Description, "[the spec]: https://example.com/spec")
}

// TestGodocLinks_On exercises the full feature: resolvable doc-links are
// recomposed to the referenced schema's EXPOSED name, unresolved links are
// humanized, reference-definition lines are dropped, and non-doc-link brackets
// are left intact.
func TestGodocLinks_On(t *testing.T) {
	doc := scanGodocLinks(t, true)
	gizmo, ok := doc.Definitions["gizmo"]
	require.True(t, ok, "gizmo definition missing")
	goldenRaw(t, "gizmo_on", gizmo)

	// Leading self-name recomposed to the exposed model name (swagger:model
	// gizmo), restored to sentence case: "Widget …" → "Gizmo …".
	assert.Contains(t, gizmo.Title, "Gizmo is the primary")
	assert.NotContains(t, gizmo.Title, "Widget")
	// Doc-link to another model → its exposed name (no override → Go name).
	assert.Contains(t, gizmo.Title, "Gadget")
	assert.NotContains(t, gizmo.Title, "[Gadget]")
	// type.field doc-link → exposed model name + exposed (json) property name.
	assert.Contains(t, gizmo.Title, "Order.customer_name")
	assert.NotContains(t, gizmo.Title, "[Order.CustName]")

	// Description: pointer link resolved; unknown link humanized; cross-package
	// link resolved via imports.
	assert.Contains(t, gizmo.Description, "Gadget pointer")
	assert.Contains(t, gizmo.Description, "unknown sprocket")
	assert.Contains(t, gizmo.Description, "Ledger")
	assert.NotContains(t, gizmo.Description, "[")
	_, ok = doc.Definitions["Ledger"]
	assert.True(t, ok, "cross-package Ledger definition missing")

	// Field doc-link recomposed.
	assert.Contains(t, gizmo.Properties["holder"].Description, "Gadget that owns")
	assert.NotContains(t, gizmo.Properties["holder"].Description, "[Gadget]")

	// Reference-definition line dropped on the spec field; the [Gadget] link
	// there recomposes to its exposed name.
	spec := gizmo.Properties["spec"].Description
	assert.Contains(t, spec, "Spec points at Gadget")
	assert.NotContains(t, spec, "https://example.com/spec")

	// Conservative recognizer: ordinary (non-doc-link) brackets are left intact.
	idx := gizmo.Properties["index"].Description
	assert.Contains(t, idx, "[0]")
	assert.Contains(t, idx, "[see notes]")
	assert.Contains(t, idx, "[id]")
}
