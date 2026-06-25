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

func runAfterDeclComments(t *testing.T, on bool) *spec.Swagger {
	t.Helper()
	doc, err := codescan.Run(&codescan.Options{
		Packages:          []string{"./enhancements/after-decl-comments/..."},
		WorkDir:           scantest.FixturesDir(),
		ScanModels:        true,
		AfterDeclComments: on,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc
}

// TestCoverage_AfterDeclComments_Off is the control: with the option off, the
// inside-body / inlined annotations are inert — the carriers carry no annotation
// in their (clean) godoc, so nothing is discovered.
func TestCoverage_AfterDeclComments_Off(t *testing.T) {
	doc := runAfterDeclComments(t, false)
	assert.MapNotContainsT(t, doc.Definitions, "widgetModel")
	assert.MapNotContainsT(t, doc.Definitions, "countType")
}

// TestCoverage_AfterDeclComments_On exercises Phase A: with the option on, the
// struct's inside-body annotation and the defined type's inlined trailing
// annotation are discovered, named, and validated; the route inside a func body
// is discovered (already location-agnostic).
func TestCoverage_AfterDeclComments_On(t *testing.T) {
	doc := runAfterDeclComments(t, true)

	// struct inside-body: swagger:model widgetModel + maxProperties: 5
	require.MapContainsT(t, doc.Definitions, "widgetModel")
	widget := doc.Definitions["widgetModel"]
	require.NotNil(t, widget.MaxProperties)
	assert.EqualT(t, int64(5), *widget.MaxProperties)

	// field-level inlined trailing comment (Phase B): swagger:strfmt date
	require.MapContainsT(t, widget.Properties, "created")
	assert.EqualT(t, "date", widget.Properties["created"].Format)

	// defined type inlined trailing: swagger:model countType
	require.MapContainsT(t, doc.Definitions, "countType")
	assert.SliceContainsT(t, doc.Definitions["countType"].Type, "integer")

	// type alias inlined trailing: swagger:model stampType
	require.MapContainsT(t, doc.Definitions, "stampType")
	assert.SliceContainsT(t, doc.Definitions["stampType"].Type, "string")

	// route inside the func body is discovered (location-agnostic today).
	require.NotNil(t, doc.Paths)
	require.MapContainsT(t, doc.Paths.Paths, "/widgets")
	require.NotNil(t, doc.Paths.Paths["/widgets"].Get)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_after_decl_comments.json")
}
