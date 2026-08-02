// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
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
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/route-name-shapes/..."},
		WorkDir:  scantest.FixturesDir(),
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

	// The per-path assertions above say the annotations parsed; the golden says what they produced.
	scantest.CompareOrDumpJSON(t, doc, "enhancements_route_name_shapes.json")
}
