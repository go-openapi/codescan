// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_NameFromTags exercises the NameFromTags option end-to-end
// (go-swagger#2912, go-swagger#1391): the emitted name of a schema property,
// query parameter, or response header is derived from the first struct-tag
// type listed in Options.NameFromTags. The fixture's fields each carry a
// json: and a form: tag with differing names.
//
//   - default (nil → ["json"]): json names, the historic behaviour.
//   - ["form","json"]: the form name wins, falling back to json.
//   - explicit empty slice: no tag consulted; the Go field name is used.
//
// The encoding/json directives are independent of the setting: json:"-"
// always excludes (Filter.Internal), and ,omitempty is always read from the
// json tag.
func TestCoverage_NameFromTags(t *testing.T) {
	run := func(t *testing.T, nameTags []string) *spec.Swagger {
		t.Helper()
		doc, err := codescan.Run(&codescan.Options{
			Packages:     []string{"./enhancements/name-from-tags/..."},
			WorkDir:      scantest.FixturesDir(),
			ScanModels:   true,
			NameFromTags: nameTags,
		})
		require.NoError(t, err)
		require.NotNil(t, doc)
		return doc
	}

	// queryParamNames collects the GET /items query parameter names.
	queryParamNames := func(t *testing.T, doc *spec.Swagger) []string {
		t.Helper()
		require.NotNil(t, doc.Paths)
		pi, ok := doc.Paths.Paths["/items"]
		require.TrueT(t, ok, "expected a /items path")
		require.NotNil(t, pi.Get)
		names := make([]string, 0, len(pi.Get.Parameters))
		for _, p := range pi.Get.Parameters {
			names = append(names, p.Name)
		}
		return names
	}

	// responseHeaders returns the header map of the listResponse. The route
	// references it by name, so the response (with its Headers) lands in the
	// top-level responses section and the path holds a $ref to it.
	responseHeaders := func(t *testing.T, doc *spec.Swagger) map[string]spec.Header {
		t.Helper()
		resp, ok := doc.Responses["listResponse"]
		require.TrueT(t, ok, "expected a listResponse in the responses section")
		return resp.Headers
	}

	t.Run("default uses json names", func(t *testing.T) {
		doc := run(t, nil)

		props := doc.Definitions["Filter"].Properties
		require.MapContainsT(t, props, "sortKey")
		require.MapContainsT(t, props, "pageSize")
		require.MapContainsT(t, props, "label")
		assert.MapNotContainsT(t, props, "sort_key")
		assert.MapNotContainsT(t, props, "internal")
		assert.MapNotContainsT(t, props, "Internal", "json:\"-\" always excludes")

		assert.SliceContainsT(t, queryParamNames(t, doc), "sortKey")
		assert.SliceContainsT(t, queryParamNames(t, doc), "pageSize")

		_, ok := responseHeaders(t, doc)["X-Request-Id"]
		assert.TrueT(t, ok)
	})

	t.Run("form-first uses form names, falling back to json", func(t *testing.T) {
		doc := run(t, []string{"form", "json"})

		props := doc.Definitions["Filter"].Properties
		require.MapContainsT(t, props, "sort_key")
		require.MapContainsT(t, props, "page_size")
		// Label has form:"-" (not a usable name) → falls back to json "label".
		require.MapContainsT(t, props, "label")
		assert.MapNotContainsT(t, props, "sortKey")
		assert.MapNotContainsT(t, props, "internal", "json:\"-\" always excludes")

		assert.SliceContainsT(t, queryParamNames(t, doc), "sort_key")
		assert.SliceContainsT(t, queryParamNames(t, doc), "page_size")

		_, ok := responseHeaders(t, doc)["x_request_id"]
		assert.TrueT(t, ok)

		// Capture the full form-first spec as the golden exhibit of the feature
		// (form names across schema properties, query parameters and headers).
		scantest.CompareOrDumpJSON(t, doc, "enhancements_name_from_tags_form_first.json")
	})

	t.Run("empty slice uses Go field names, json directives still apply", func(t *testing.T) {
		doc := run(t, []string{})

		props := doc.Definitions["Filter"].Properties
		require.MapContainsT(t, props, "SortKey")
		require.MapContainsT(t, props, "PageSize")
		require.MapContainsT(t, props, "Label")
		assert.MapNotContainsT(t, props, "Internal", "json:\"-\" always excludes")
		assert.MapNotContainsT(t, props, "sortKey")
		assert.MapNotContainsT(t, props, "sort_key")

		assert.SliceContainsT(t, queryParamNames(t, doc), "SortKey")
		assert.SliceContainsT(t, queryParamNames(t, doc), "PageSize")

		_, ok := responseHeaders(t, doc)["RequestID"]
		assert.TrueT(t, ok)
	})
}
