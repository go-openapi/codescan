// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const pruneUnusedPkg = "./enhancements/prune-unused/..."

// runPrune scans the prune-unused fixture, capturing diagnostics by code.
func runPrune(t *testing.T, scanModels, prune bool, input *oaispec.Swagger) (*oaispec.Swagger, map[grammar.Code][]grammar.Diagnostic) {
	t.Helper()
	byCode := map[grammar.Code][]grammar.Diagnostic{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:          []string{pruneUnusedPkg},
		WorkDir:           scantest.FixturesDir(),
		InputSpec:         input,
		ScanModels:        scanModels,
		PruneUnusedModels: prune,
		OnDiagnostic: func(d grammar.Diagnostic) {
			byCode[d.Code] = append(byCode[d.Code], d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc, byCode
}

// TestPruneUnused_Off is the control: with ScanModels and no prune, every
// swagger:model is emitted — including the unreferenced Unused / OnlyByUnused —
// and the a.Thing / b.Thing collision is deconflicted to AThing / BThing (two
// rename Hints). This is the shape the prune pass reduces.
func TestPruneUnused_Off(t *testing.T) {
	doc, byCode := runPrune(t, true, false, nil)

	for _, name := range []string{"Used", "B", "C", "Node", "Unused", "OnlyByUnused", "AThing", "BThing"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.Len(t, doc.Definitions, 8)
	assert.Empty(t, byCode[grammar.CodePrunedUnused], "nothing is pruned without the flag")
	assert.Len(t, byCode[grammar.CodeRenamedDefinition], 2,
		"the a.Thing / b.Thing collision is deconflicted into two renames")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_prune_unused_all.json")
}

// TestPruneUnused_On is the core case. ScanModels + PruneUnusedModels keeps only
// the route-reachable closure (Used -> B -> C, the recursive Node, and a.Thing),
// drops the dead set (Unused, OnlyByUnused, b.Thing), and — because the prune
// runs BEFORE name reduction — the a.Thing / b.Thing collision never surfaces,
// so a.Thing keeps the bare name "Thing" with no AThing / BThing churn.
func TestPruneUnused_On(t *testing.T) {
	doc, byCode := runPrune(t, true, true, nil)

	// Reachable closure survives.
	for _, name := range []string{"Used", "B", "C", "Node", "Thing"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.Len(t, doc.Definitions, 5)

	// Dead set is gone.
	for _, name := range []string{"Unused", "OnlyByUnused"} {
		assert.NotContains(t, doc.Definitions, name)
	}

	// The collision evaporated: bare "Thing", no deconflicted concat, no rename.
	assert.NotContains(t, doc.Definitions, "AThing")
	assert.NotContains(t, doc.Definitions, "BThing")
	assert.Empty(t, byCode[grammar.CodeRenamedDefinition],
		"pruning the unused twin removes the collision, so nothing is renamed")

	// The kept reference re-points to the bare name; the cycle is intact.
	thingRef := doc.Definitions["Used"].Properties["thing"].Ref
	nodeRef := doc.Definitions["Node"].Properties["next"].Ref
	assert.Equal(t, "#/definitions/Thing", thingRef.String())
	assert.Equal(t, "#/definitions/Node", nodeRef.String(),
		"the self-recursive cycle survives and back-refs to its own definition")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_prune_unused.json")
}

// TestPruneUnused_Diagnostics asserts the prune is loud: one located Hint per
// pruned definition (Unused, OnlyByUnused, b.Thing), each a Hint severity with a
// source line, and no rename Hints (the collision was pruned away).
func TestPruneUnused_Diagnostics(t *testing.T) {
	_, byCode := runPrune(t, true, true, nil)

	hints := byCode[grammar.CodePrunedUnused]
	require.Len(t, hints, 3, "one Hint per pruned definition")
	for _, d := range hints {
		assert.Equal(t, grammar.SeverityHint, d.Severity, "a prune is informational")
		assert.Positive(t, d.Pos.Line, "the Hint is located at the pruned type's declaration")
	}
	assert.Empty(t, byCode[grammar.CodeRenamedDefinition])
}

// TestPruneUnused_NoOpWithoutScanModels confirms the flag is inert without
// ScanModels (the emitted set is already reachable-only): the output matches the
// plain reachable scan, and exactly one positionless Hint explains the no-op.
func TestPruneUnused_NoOpWithoutScanModels(t *testing.T) {
	doc, byCode := runPrune(t, false, true, nil)

	// Same reachable closure as a plain no-ScanModels scan — nothing extra to prune.
	for _, name := range []string{"Used", "B", "C", "Node", "Thing"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.NotContains(t, doc.Definitions, "Unused")

	hints := byCode[grammar.CodePrunedUnused]
	require.Len(t, hints, 1, "a single no-op notice")
	assert.Equal(t, grammar.SeverityHint, hints[0].Severity)
	assert.Zero(t, hints[0].Pos.Line, "the no-op notice is positionless")
}

// TestPruneUnused_InputSpecPinned confirms decision 2: a definition supplied via
// InputSpec is pinned — never pruned even when no path references it — and seeds
// the reachability roots.
func TestPruneUnused_InputSpecPinned(t *testing.T) {
	input := &oaispec.Swagger{
		SwaggerProps: oaispec.SwaggerProps{
			Definitions: oaispec.Definitions{
				"Pinned": {SchemaProps: oaispec.SchemaProps{Type: oaispec.StringOrArray{"object"}}},
			},
		},
	}
	doc, _ := runPrune(t, true, true, input)

	assert.Contains(t, doc.Definitions, "Pinned", "an overlay-supplied definition is never pruned")
	// The discovered dead set is still pruned around the pinned definition.
	assert.NotContains(t, doc.Definitions, "Unused")
}

// TestPruneUnused_Provenance confirms a pruned definition leaves no orphan
// provenance: no anchor references a pruned node, and every emitted anchor
// resolves in the rendered spec.
func TestPruneUnused_Provenance(t *testing.T) {
	var recorded []scanner.Provenance
	doc, err := codescan.Run(&codescan.Options{
		Packages:          []string{pruneUnusedPkg},
		WorkDir:           scantest.FixturesDir(),
		ScanModels:        true,
		PruneUnusedModels: true,
		OnProvenance: func(p scanner.Provenance) {
			recorded = append(recorded, p)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotEmpty(t, recorded)

	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	var root any
	require.NoError(t, json.Unmarshal(raw, &root))

	var sawKept bool
	for _, p := range recorded {
		assert.Truef(t, resolveJSONPointer(root, p.Pointer),
			"anchor %q (from %s:%d) does not resolve — orphaned by a prune?",
			p.Pointer, p.Pos.Filename, p.Pos.Line)
		switch p.Pointer {
		case "/definitions/Used", "/definitions/Thing":
			sawKept = true
		}
	}
	assert.True(t, sawKept, "kept definitions still carry provenance")
}
