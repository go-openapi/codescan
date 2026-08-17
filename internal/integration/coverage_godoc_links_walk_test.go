// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A doc-link is recomposed in two phases: the builder writes a marker carrying the referenced
// definition's key, and the spec builder rewrites that marker to the definition's final exposed
// name once name reduction has run. The second phase has to walk the WHOLE document, because prose
// lives in a dozen places that name reduction never touches — the info block, the spec-level
// parameter and response namespaces, each operation and its parameters, response headers, and every
// nested schema shape.
//
// The godoc-links fixture is models-only, so it only ever exercised the definitions arm of that
// walk. This one puts a doc-link in each of the other positions.
func TestCoverage_GodocLinksWalk(t *testing.T) {
	doc, err := runScan(&codescan.Options{
		Packages:   []string{"./enhancements/godoc-links-walk/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		CleanGoDoc: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	t.Run("the info block", func(t *testing.T) {
		require.NotNil(t, doc.Info)
		assert.Contains(t, doc.Info.Title, "widget")
		assert.Contains(t, doc.Info.Description, "gadget")
		assert.NotContains(t, doc.Info.Title+doc.Info.Description, "[")
	})

	t.Run("the spec-level parameter namespace", func(t *testing.T) {
		p, ok := doc.Parameters["X-Trace-ID"]
		require.True(t, ok, "the shared header parameter must be registered")
		assert.Contains(t, p.Description, "Widget")
		assert.NotContains(t, p.Description, "[Widget]")
	})

	t.Run("the spec-level response namespace", func(t *testing.T) {
		r, ok := doc.Responses["WidgetError"]
		require.True(t, ok, "the shared response must be registered")
		msg := r.Schema.Properties["message"].Description
		assert.Contains(t, msg, "Widget", "prose nested inside a shared response's schema is walked too")
		assert.NotContains(t, msg, "[Widget]")
	})

	t.Run("a response header", func(t *testing.T) {
		r, ok := doc.Responses["widgetOK"]
		require.True(t, ok)
		h, ok := r.Headers["ETag"]
		require.True(t, ok)
		assert.Contains(t, h.Description, "Widget")
		assert.NotContains(t, h.Description, "[Widget]")
	})

	pi, ok := doc.Paths.Paths["/widgets"]
	require.True(t, ok, "the route must be discovered")

	t.Run("a path-item parameter", func(t *testing.T) {
		require.NotEmpty(t, pi.Parameters, "the path item carries its own parameters")
		assert.Contains(t, pi.Parameters[0].Description, "Widget")
		assert.NotContains(t, pi.Parameters[0].Description, "[Widget]")
	})

	t.Run("an operation's own prose", func(t *testing.T) {
		require.NotNil(t, pi.Get)
		assert.Contains(t, pi.Get.Summary, "widget")
		assert.Contains(t, pi.Get.Description, "gadget")
		assert.NotContains(t, pi.Get.Summary+pi.Get.Description, "[")
	})

	t.Run("nested schema shapes", func(t *testing.T) {
		// An allOf sibling arm is a schema of its own, reached only by recursing through AllOf.
		asm := doc.Definitions["Assembly"]
		require.Len(t, asm.AllOf, 2)
		assert.Contains(t, asm.AllOf[1].Properties["serial"].Description, "Assembly")

		w := doc.Definitions["Widget"]
		assert.Contains(t, w.Properties["parts"].Description, "Gadget", "array items")
		assert.Contains(t, w.Properties["index"].Description, "Gadget", "map values")

		cat := doc.Definitions["Catalogue"]
		require.Contains(t, cat.PatternProperties, "^x-")
		assert.Contains(t, cat.Title, "Gadget", "pattern properties")
	})

	// An inline `description:` inside a route's Parameters block is authored spec text, not godoc
	// prose, so CleanGoDoc leaves it exactly as written — the same boundary that protects a
	// swagger:description override.
	t.Run("an authored description is not godoc prose", func(t *testing.T) {
		require.NotNil(t, pi.Get)
		require.NotEmpty(t, pi.Get.Parameters)
		assert.Contains(t, pi.Get.Parameters[0].Description, "[Widget]",
			"an authored keyword value keeps its brackets")
	})

	// A marker that reached the output would be a two-phase bug: phase one wrote it and phase two
	// failed to find it.
	t.Run("no marker leaks", func(t *testing.T) {
		assert.NotContains(t, mustJSON(t, doc), "\x00")
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_godoc_links_walk.json")
}

// A doc-link resolves at build time against the scanned types, but pruning runs afterwards — so a
// link can name a definition that is no longer in the finished document. The marker then has
// nothing to rewrite to and must collapse to its humanized fallback; leaving it in place would put
// a raw marker in the user's spec.
//
// Assertions rather than a golden: this pins one branch, not a document shape.
func TestCoverage_GodocLinksWalk_PrunedTarget(t *testing.T) {
	doc, err := runScan(&codescan.Options{
		Packages:          []string{"./enhancements/godoc-links-walk/..."},
		WorkDir:           scantest.FixturesDir(),
		ScanModels:        true,
		PruneUnusedModels: true,
		CleanGoDoc:        true,
	})
	require.NoError(t, err)

	require.NotContains(t, doc.Definitions, "Catalogue",
		"precondition: nothing references Catalogue, so pruning drops it")

	w, ok := doc.Definitions["Widget"]
	require.True(t, ok, "Widget is reachable from the widgetOK response and survives")

	assert.Contains(t, w.Description, "catalogue",
		"the link to the pruned definition falls back to its humanized text")
	assert.NotContains(t, mustJSON(t, doc), "\x00",
		"a marker with no surviving target must not reach the output")
}

// mustJSON renders the document so a whole-document scan for leaked markers is one assertion.
func mustJSON(t *testing.T, doc any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	require.NoError(t, err)

	return string(b)
}
