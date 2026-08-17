// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/docs/examples/internal/loadertest"
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

func scanStreams(t *testing.T) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(loadertest.Apply(&codescan.Options{
		WorkDir:    examplesRoot(t),
		Packages:   []string{"./shaping/streams"},
		ScanModels: true,
	}))
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestStreams emits testdata/attachment.json and testdata/upload_params.json — the
// two shapes the "File uploads and byte streams" guide renders — and asserts the
// position-dependent split: `file` on a formData parameter, `{string, byte}`
// everywhere else, with an explicit annotation still winning.
//
// Regenerate with: UPDATE_GOLDEN=1 go test ./...
func TestStreams(t *testing.T) {
	doc := scanStreams(t)

	a, ok := doc.Definitions["Attachment"]
	require.True(t, ok, "Attachment definition missing")

	for _, name := range []string{"content", "thumbnail"} {
		p := a.Properties[name]
		assert.Equal(t, "string", p.Type[0], "%s renders as a string", name)
		assert.Equal(t, "byte", p.Format, "%s carries the base64 format", name)
	}

	checksum := a.Properties["checksum"]
	assert.Equal(t, "base64", checksum.Format, "an explicit swagger:strfmt wins over the default")

	assert.NotContains(t, doc.Definitions, "Reader", "a recognized stream type is never published")
	assert.NotContains(t, doc.Definitions, "ReadCloser", "a recognized stream type is never published")

	require.NotNil(t, doc.Paths)
	item, ok := doc.Paths.Paths["/attachments"]
	require.True(t, ok, "/attachments missing")
	require.NotNil(t, item.Post)

	params := make(map[string]spec.Parameter, len(item.Post.Parameters))
	for _, p := range item.Post.Parameters {
		params[p.Name] = p
	}

	upload, ok := params["upload"]
	require.True(t, ok, "upload parameter missing")
	assert.Equal(t, "formData", upload.In)
	assert.Equal(t, "file", upload.Type, "a stream in formData is the canonical upload shape")

	writeGolden(t, filepath.Join("testdata", "attachment.json"), a)
	writeGolden(t, filepath.Join("testdata", "upload_params.json"), item.Post.Parameters)
}

func writeGolden(t *testing.T, path string, v any) {
	t.Helper()

	got, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, got, 0o600))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}
