// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestSpec(t *testing.T) {
	t.Parallel()

	t.Run("should write indented JSON by default", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		asJSON, err := Spec(document(t), Config{Stdout: &out})
		require.NoError(t, err)

		assert.Contains(t, out.String(), "\n  ", "indented, which is what a human reads")
		assert.True(t, json.Valid(asJSON))
		assert.Equal(t, strings.TrimRight(out.String(), "\n"), string(asJSON),
			"and what was returned is what was written")
	})

	t.Run("should write one line when compact", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		_, err := Spec(document(t), Config{CompactJSON: true, Stdout: &out})
		require.NoError(t, err)

		assert.NotContains(t, out.String(), "\n  ", "compact means one line, not a differently indented one")
		assert.True(t, json.Valid([]byte(strings.TrimSpace(out.String()))))
	})

	t.Run("should return the JSON even when it writes YAML", func(t *testing.T) {
		// The contract the caller depends on: -validate checks the same bytes a JSON run would have
		// written, whatever the document on disk ended up looking like.
		t.Parallel()

		var out bytes.Buffer
		asJSON, err := Spec(document(t), Config{WantsYAML: true, Stdout: &out})
		require.NoError(t, err)

		assert.True(t, json.Valid(asJSON), "returned as JSON")
		assert.True(t, strings.HasPrefix(out.String(), "info:"), "written as YAML: %.30s", out.String())
		assert.NotContains(t, out.String(), `"swagger"`, "and not as both")
	})

	t.Run("should write to the file -output names", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "swagger.json")

		var out bytes.Buffer
		_, err := Spec(document(t), Config{Output: path, Stdout: &out})
		require.NoError(t, err)

		assert.Empty(t, out.String(), "the document went to the file, so nothing goes to standard output")

		written, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.True(t, json.Valid(written))
		assert.True(t, strings.HasSuffix(string(written), "\n"), "files end with a newline")
	})

	t.Run("should report a file it cannot write", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "nowhere", "swagger.json")

		_, err := Spec(document(t), Config{Output: path})

		require.Error(t, err)
		assert.Contains(t, err.Error(), path, "which file, since -output named it")
	})

	t.Run("should hand back the JSON even when the write fails", func(t *testing.T) {
		// A YAML run that cannot write still knows what the document was, and the caller is about to
		// report on it: the failure is about the destination, not about the specification.
		t.Parallel()

		path := filepath.Join(t.TempDir(), "nowhere", "swagger.yaml")

		asJSON, err := Spec(document(t), Config{WantsYAML: true, Output: path})

		require.Error(t, err)
		assert.True(t, json.Valid(asJSON), "got %q", asJSON)
	})

	t.Run("should write nowhere rather than fail with no stream", func(t *testing.T) {
		// Standard output is the caller's to provide, and a caller that provides none is asking for the
		// document to be produced, not printed.
		t.Parallel()

		asJSON, err := Spec(document(t), Config{})

		require.NoError(t, err)
		assert.True(t, json.Valid(asJSON))
	})
}

// marshalYAML is reached with the JSON rendering this package has just produced, so it can only fail on a document
// that never came from here - which is worth saying rather than leaving to a panic somewhere downstream.
func TestMarshalYAML(t *testing.T) {
	t.Parallel()

	_, err := marshalYAML([]byte("\tnot a document"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot re-read the document as YAML")
}

// document is the smallest specification that renders as one: enough to tell JSON from YAML, and no more.
func document(t *testing.T) *spec.Swagger {
	t.Helper()

	var doc spec.Swagger
	require.NoError(t, json.Unmarshal(
		[]byte(`{"swagger":"2.0","info":{"title":"Pets","version":"1.0"}}`), &doc,
	))

	return &doc
}
