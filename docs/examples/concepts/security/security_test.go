// SPDX-License-Identifier: Apache-2.0

package security

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

// TestSecurityInCode scans this package and emits the golden fragments the
// "Security" tutorial renders: the meta-declared schemes + default requirement,
// and the per-route override.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestSecurityInCode(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./concepts/security"},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.SecurityDefinitions, "api_key")
	require.Contains(t, doc.SecurityDefinitions, "oauth2")
	require.Len(t, doc.Security, 1, "document-wide default requirement")

	create := doc.Paths.Paths["/reports"].Post
	require.NotNil(t, create)
	require.Len(t, create.Security, 1, "createReport overrides with oauth2")

	archive := doc.Paths.Paths["/reports/archive"].Post
	require.NotNil(t, archive)
	require.Len(t, archive.Security, 1, "two ANDed schemes form a single requirement object")
	require.Contains(t, archive.Security[0], "api_key")
	require.Contains(t, archive.Security[0], "oauth2")

	public := doc.Paths.Paths["/reports/public"].Get
	require.NotNil(t, public)
	require.NotNil(t, public.Security, "Security: [] emits an explicit (non-nil) empty requirement")
	require.Empty(t, public.Security, "an empty requirement opts the operation out of the default")

	goldenRaw(t, "schemes", map[string]any{
		"securityDefinitions": doc.SecurityDefinitions,
		"security":            doc.Security,
	})
	goldenRaw(t, "route", create.Security)
	goldenRaw(t, "and", archive.Security)
	goldenRaw(t, "public", map[string]any{"security": public.Security})
}

// TestSecurityByOverlay shows the "keep security out of app code" path: the
// app package (concepts/routes) carries NO security annotations, yet the merged
// spec is secured because an InputSpec base supplies the schemes and the
// default requirement.
func TestSecurityByOverlay(t *testing.T) {
	const baseSpec = `{
      "swagger": "2.0",
      "securityDefinitions": {"api_key": {"type": "apiKey", "in": "header", "name": "X-API-Key"}},
      "security": [{"api_key": []}]
    }`
	var base spec.Swagger
	require.NoError(t, json.Unmarshal([]byte(baseSpec), &base))

	doc, err := codescan.Run(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./concepts/routes"},
		ScanModels: true,
		InputSpec:  &base,
	})
	require.NoError(t, err)
	require.Contains(t, doc.SecurityDefinitions, "api_key", "schemes come from the base, not the code")
	require.Len(t, doc.Security, 1, "default requirement comes from the base")
	require.NotEmpty(t, doc.Paths.Paths, "the scanned routes are merged on top")

	goldenRaw(t, "overlay", map[string]any{
		"securityDefinitions": doc.SecurityDefinitions,
		"security":            doc.Security,
	})
}
