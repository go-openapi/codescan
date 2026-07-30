// SPDX-License-Identifier: Apache-2.0

package omit

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

// goldenJSON emits and verifies one golden fragment, rewriting it under UPDATE_GOLDEN.
func goldenJSON(t *testing.T, feature string, v any) {
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

// TestOmitFragments backs the `swagger:omit` reference page: the request body carries only the
// fields the author kept, while the shared type's own definition — which the response $refs — still
// documents all of them. The omission is local to the body that asked for it.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestOmitFragments(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		WorkDir:  examplesRoot(t),
		Packages: []string{"./concepts/omit"},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pi, ok := doc.Paths.Paths["/users"]
	require.True(t, ok)
	require.NotNil(t, pi.Post)

	var body *spec.Schema
	for _, prm := range pi.Post.Parameters {
		if prm.Schema != nil {
			body = prm.Schema
		}
	}
	require.NotNil(t, body, "the body parameter is missing")

	assert.Contains(t, body.Properties, "Name")
	assert.NotContains(t, body.Properties, "ID", "swagger:omit drops the server-assigned field")
	assert.NotContains(t, body.Properties, "Created")

	// The response refers to the shared type's own definition, which is untouched: the omission is
	// local to the body that asked for it, never a mutation of the type everyone else sees.
	resp := doc.Responses["userResponse"]
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "#/definitions/User", resp.Schema.Ref.String())

	user, ok := doc.Definitions["User"]
	require.True(t, ok)
	for _, name := range []string{"ID", "Name", "Created"} {
		assert.Containsf(t, user.Properties, name,
			"the shared definition still documents the whole type: %s", name)
	}

	goldenJSON(t, "body", body) // the trimmed request body
	goldenJSON(t, "user", user) // the shared definition, in full
}
