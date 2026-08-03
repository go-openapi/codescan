// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// The stdlib stream types are recognized by identity, so they never reach structural drilling.
//
// Two answers, decided by what the position allows rather than by anything in the declaration:
// `type: file` for a formData parameter — the only place OAS 2.0 permits it, and the canonical
// file-upload shape — and `{type: string, format: byte}` everywhere else.
//
// `byte` rather than `binary`: in a JSON body a raw octet sequence has no representation, and
// `byte` is the base64-encoded string OAS 2.0 defines for exactly that. It is not a claim about
// the content — a stream is opaque, and this is the standard way of saying so.
func TestOpaqueStreams(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/opaque-streams/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	t.Run("every recognized type is opaque bytes on a model field", func(t *testing.T) {
		props := doc.Definitions["StreamModel"].Properties
		require.NotEmpty(t, props)

		for name := range props {
			assert.Equal(t, "string/byte", schemaSignature(props[name], doc.Definitions, 0),
				"%s must be opaque bytes, not a drilled structure", name)
		}
	})

	t.Run("x-go-type records which stream it was", func(t *testing.T) {
		// Neither answer can carry it: `byte` says base64 bytes, `file` says an upload, and all twelve
		// types collapse onto the same schema. Without the extension a consumer cannot tell an
		// io.Reader field from a multipart.File one — the `recognizeError` criterion exactly.
		props := doc.Definitions["StreamModel"].Properties

		for field, want := range map[string]string{
			"payload":    "io.Reader",
			"envelope":   "io.ReadCloser",
			"excerpt":    "io.LimitedReader",
			"upload":     "mime/multipart.File",
			"attachment": "github.com/go-openapi/runtime.NamedReadCloser",
		} {
			assert.Equal(t, want, props[field].Extensions["x-go-type"],
				"%s must record its Go type", field)
		}

		// x-go-name stays the FIELD name — the fixture names every field unlike its type so the two
		// extensions can never be confused for one another.
		assert.Equal(t, "Payload", props["payload"].Extensions["x-go-name"])
	})

	t.Run("SkipExtensions suppresses the x-go-type stamp", func(t *testing.T) {
		bare, err := codescan.Run(&codescan.Options{
			Packages:       []string{"./enhancements/opaque-streams/..."},
			WorkDir:        scantest.FixturesDir(),
			ScanModels:     true,
			SkipExtensions: true,
		})
		require.NoError(t, err)

		p := bare.Definitions["StreamModel"].Properties["payload"]
		assert.NotContains(t, p.Extensions, "x-go-type")
		assert.Equal(t, "string/byte", schemaSignature(p, bare.Definitions, 0),
			"suppressing the extension must not change the type")
	})

	t.Run("no stdlib interface is published as a definition", func(t *testing.T) {
		// The defect that made this visible: `io`'s own interfaces became definitions carrying io's
		// godoc, and ReadCloser grew a `close` property of type string out of `Close() error`.
		for _, leaked := range []string{"Reader", "ReadCloser", "ReadSeeker", "LimitedReader", "File", "NamedReadCloser"} {
			assert.NotContains(t, doc.Definitions, leaked,
				"a recognized stream type must not reach model discovery")
		}
	})

	t.Run("an explicit annotation still wins", func(t *testing.T) {
		props := doc.Definitions["OverriddenModel"].Properties
		assert.Equal(t, "string/base64", schemaSignature(props["blob"], doc.Definitions, 0))
		assert.Equal(t, "string/", schemaSignature(props["handle"], doc.Definitions, 0))

		// The two overrides treat the recognizer's stamp differently, and the difference follows from
		// what each one means: swagger:strfmt adjusts the format, leaving the Go type it came from
		// intact and recorded; swagger:type replaces the schema outright, so the record goes too.
		assert.Equal(t, "io.Reader", props["blob"].Extensions["x-go-type"],
			"a format override leaves the Go type recorded")
		assert.NotContains(t, props["handle"].Extensions, "x-go-type",
			"a type override replaces the schema, stamp included")
	})

	t.Run("a formData parameter is a file", func(t *testing.T) {
		params := postParamsByName(t, doc, "/streams", "uploadStream")

		for _, name := range []string{"upload", "doc"} {
			p, ok := params[name]
			require.True(t, ok, "missing parameter %s", name)
			assert.Equal(t, "formData", p.In)
			assert.Equal(t, "file", p.Type,
				"the canonical upload shape is type: file — and SimpleSchema requires SOME type here")
		}
	})

	t.Run("a body parameter is opaque bytes", func(t *testing.T) {
		params := postParamsByName(t, doc, "/streams", "uploadStream")

		p, ok := params["body"]
		require.True(t, ok)
		require.NotNil(t, p.Schema)
		assert.Equal(t, "string/byte", schemaSignature(*p.Schema, doc.Definitions, 0),
			"`file` is not legal on a body parameter")
	})

	t.Run("a non-body, non-formData parameter is opaque bytes", func(t *testing.T) {
		params := postParamsByName(t, doc, "/streams", "uploadStream")

		p, ok := params["marker"]
		require.True(t, ok)
		assert.Equal(t, simpleSignature("string", "byte", nil), simpleSignature(p.Type, p.Format, p.Items))
	})

	t.Run("a response body and a response header are opaque bytes", func(t *testing.T) {
		resp, ok := doc.Responses["streamResponse"]
		require.True(t, ok)
		require.NotNil(t, resp.Schema)
		assert.Equal(t, "string/byte", schemaSignature(*resp.Schema, doc.Definitions, 0))

		h, ok := resp.Headers["XChecksum"]
		require.True(t, ok, "missing response header; got %v", resp.Headers)
		assert.Equal(t, simpleSignature("string", "byte", nil), simpleSignature(h.Type, h.Format, h.Items))
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_opaque_streams.json")
}

// postParamsByName indexes a POST operation's parameters by name.
func postParamsByName(t *testing.T, doc *oaispec.Swagger, path, opID string) map[string]oaispec.Parameter {
	t.Helper()

	item, ok := doc.Paths.Paths[path]
	require.True(t, ok, "missing path %s", path)
	require.NotNil(t, item.Post, "missing POST %s", path)
	require.Equal(t, opID, item.Post.ID)

	params := make(map[string]oaispec.Parameter, len(item.Post.Parameters))
	for _, p := range item.Post.Parameters {
		params[p.Name] = p
	}

	return params
}
