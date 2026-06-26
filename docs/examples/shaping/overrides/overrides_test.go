// SPDX-License-Identifier: Apache-2.0

package overrides

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

// Stable, public diagnostic-code strings (codescan.Code is a string alias).
const (
	codeEmptyOverride  codescan.Code = "scan.empty-override"
	codeContextInvalid codescan.Code = "parse.context-invalid"
)

func examplesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// scanOverrides scans the witness package. emitRefSiblings selects whether a
// $ref'd field keeps its override siblings or drops to a bare $ref; diags
// collects the scan diagnostics.
func scanOverrides(t *testing.T, emitRefSiblings bool, diags *[]codescan.Diagnostic) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:         examplesRoot(t),
		Packages:        []string{"./shaping/overrides"},
		ScanModels:      true,
		EmitRefSiblings: emitRefSiblings,
		OnDiagnostic: func(d codescan.Diagnostic) {
			if diags != nil {
				*diags = append(*diags, d)
			}
		},
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

func hasCode(diags []codescan.Diagnostic, code codescan.Code) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// TestOverrides emits and verifies the golden fragments the how-to pairs with
// each source region, and pins the behaviours the prose claims.
func TestOverrides(t *testing.T) {
	var diags []codescan.Diagnostic
	doc := scanOverrides(t, false, &diags)

	widget, ok := doc.Definitions["Widget"]
	require.True(t, ok, "Widget definition missing")
	goldenRaw(t, "widget", widget)

	// Model: both title and description overridden — the Go-facing godoc is gone.
	assert.Equal(t, "A Public Widget", widget.Title)
	assert.Equal(t, "A widget exposed via the public API.", widget.Description)

	// Field: description override; a property's title comes ONLY from an override.
	assert.Equal(t, "The unique widget identifier.", widget.Properties["id"].Description)
	assert.Equal(t, "", widget.Properties["id"].Title)
	assert.Equal(t, "Display Label", widget.Properties["label"].Title)

	// Regression: no override → godoc description retained.
	assert.NotEqual(t, "", widget.Properties["plain"].Description)

	// Co-location: the description override applies AND maximum survives.
	capacity := widget.Properties["capacity"]
	assert.Equal(t, "The maximum capacity, in liters.", capacity.Description)
	require.NotNil(t, capacity.Maximum)
	assert.Equal(t, float64(1000), *capacity.Maximum)

	// Multi-line (Option B): body lines fold into one description, \n-joined.
	assert.Equal(t,
		"Free-form notes about the widget.\nThey may span several lines, all folded into one description.",
		widget.Properties["notes"].Description)

	// Empty override: bare swagger:description suppresses the godoc and warns.
	assert.Equal(t, "", widget.Properties["suppressed"].Description)
	assert.True(t, hasCode(diags, codeEmptyOverride),
		"expected scan.empty-override for the bare swagger:description")

	// $ref field, default flags: title/description drop to a bare {$ref}.
	gadget := widget.Properties["gadget"]
	goldenRaw(t, "gadget_bare", gadget)
	assert.NotEqual(t, "", gadget.Ref.String())
	assert.Equal(t, "", gadget.Title)
	assert.Equal(t, "", gadget.Description)

	// $ref field, EmitRefSiblings: the overrides are preserved beside the $ref.
	sib := scanOverrides(t, true, nil).Definitions["Widget"].Properties["gadget"]
	goldenRaw(t, "gadget_siblings", sib)
	assert.Equal(t, "Gadget Ref", sib.Title)
	assert.Equal(t, "The attached gadget, described for API consumers.", sib.Description)
	assert.NotEqual(t, "", sib.Ref.String())
}

// TestOverrides_Responses pins the response / header description override and
// the swagger:title context error (OAS2 responses and headers have no title).
func TestOverrides_Responses(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./shaping/overrides"},
		OnDiagnostic: func(d codescan.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["errorResponse"]
	require.True(t, ok, "errorResponse missing")
	goldenRaw(t, "errorresponse", resp)

	// Response + header descriptions overridden.
	assert.Equal(t, "The error payload returned to API consumers.", resp.Description)
	hdr, ok := resp.Headers["X-Error-Code"]
	require.True(t, ok, "X-Error-Code header missing")
	assert.Equal(t, "The machine-readable error code.", hdr.Description)

	// swagger:title on a response is rejected as context-invalid.
	assert.True(t, hasCode(diags, codeContextInvalid),
		"expected parse.context-invalid for swagger:title on a response")
}
