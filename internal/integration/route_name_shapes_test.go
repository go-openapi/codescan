// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A tag or an operationId of a single character is ordinary in OAS 2.0 — both are free-form strings
// — and used to void the whole `swagger:route` it appeared in.
//
// The failure was not local to the short name, which is why it was so quiet. The tags group is
// optional, so the parse did not stop when `e` failed to match: it fell back to matching with NO
// tags, leaving the operationId pattern to swallow `e listOne`, whose alphabet has no space in it.
// The line then matched nothing at all — and a `swagger:route` that matches nothing is not a
// malformed route, it is simply not a route. Nothing downstream could tell it apart from prose, so
// there was nothing to report and the path just never appeared.
func TestRouteNameShapes(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/route-name-shapes/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d codescan.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	for _, tc := range []struct {
		path    string
		wantID  string
		wantTag []string
	}{
		{path: "/short-tag", wantID: "listOne", wantTag: []string{"e"}},
		{path: "/short-id", wantID: "l", wantTag: []string{"shapes"}},
		{path: "/short-both", wantID: "l", wantTag: []string{"e"}},
		{path: "/short-id-no-tags", wantID: "q"},
		{path: "/short-among", wantID: "listAmong", wantTag: []string{"a", "shapes"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			item, ok := doc.Paths.Paths[tc.path]
			require.True(t, ok, "no path %s: the annotation did not parse", tc.path)
			require.NotNil(t, item.Get)
			assert.Equal(t, tc.wantID, item.Get.ID)
			assert.Equal(t, tc.wantTag, item.Get.Tags)
		})
	}

	// The negative case: still no path — an operationId of `42` is not one — but no longer silent.
	t.Run("unparsed annotation is reported", func(t *testing.T) {
		_, ok := doc.Paths.Paths["/unparsed"]
		assert.False(t, ok, "an unparsable annotation must not produce a path")

		var said bool
		for _, d := range diags {
			if string(d.Code) == "scan.unparsed-path-annotation" && strings.Contains(d.Message, "/unparsed") {
				said = true
				assert.Equal(t, codescan.SeverityWarning, d.Severity)
				assert.NotZero(t, d.Pos.Line, "the diagnostic must point at the offending line")

				break
			}
		}
		assert.True(t, said, "an unparsable path annotation must be reported; got %v", diags)
	})

	// The per-path assertions above say the annotations parsed; the golden says what they produced.
	scantest.CompareOrDumpJSON(t, doc, "enhancements_route_name_shapes.json")

	// Prose is not an annotation. This package's own doc comment opens a line with `swagger:route`
	// while describing the quirk, and warning about it would make the diagnostic useless in exactly
	// the files most likely to discuss annotations.
	t.Run("prose is not reported", func(t *testing.T) {
		for _, d := range diags {
			if string(d.Code) != "scan.unparsed-path-annotation" {
				continue
			}
			assert.Contains(t, d.Message, "/unparsed",
				"only the genuinely unparsable annotation may be reported, got: %s", d.Message)
		}
	})
}
