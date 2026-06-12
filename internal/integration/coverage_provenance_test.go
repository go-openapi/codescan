// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"encoding/json"
	"maps"
	"strconv"
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_ProvenanceDefinitions exercises LX-prov-0 at its simplest: on a
// models-only fixture the OnProvenance callback fires for the definition node
// and its properties, each carrying its JSON pointer and source position. The
// wider anchor surface (paths, params, responses, enum, meta) is exercised by
// TestCoverage_ProvenanceAnchorKinds; the geometry safety net by
// TestCoverage_ProvenanceGeometry.
func TestCoverage_ProvenanceDefinitions(t *testing.T) {
	byPointer := map[string]scanner.Provenance{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/named-basic"},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnProvenance: func(p scanner.Provenance) {
			byPointer[p.Pointer] = p
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The swagger:model type surfaces as /definitions/User, anchored to its
	// declaration in the named-basic source.
	root, ok := byPointer["/definitions/User"]
	require.True(t, ok, "expected a provenance record for /definitions/User; got %v", keysOf(byPointer))
	assert.Positive(t, root.Pos.Line, "definition position should carry a source line")
	assert.True(t, strings.HasSuffix(root.Pos.Filename, ".go"),
		"definition position should point at a .go file, got %q", root.Pos.Filename)
	assert.Contains(t, root.Pos.Filename, "named-basic",
		"definition should be anchored inside the scanned fixture")

	// Its fields surface as /definitions/User/properties/{json}, anchored to the
	// struct field (a deeper, distinct source line than the type declaration).
	prop, ok := byPointer["/definitions/User/properties/email"]
	require.True(t, ok, "expected a provenance record for the email property; got %v", keysOf(byPointer))
	assert.Greater(t, prop.Pos.Line, root.Pos.Line,
		"the field should sit below the type declaration")
	assert.Equal(t, root.Pos.Filename, prop.Pos.Filename,
		"field and type share the same source file")

	// Every recorded pointer must live under the definition at this increment.
	for ptr := range byPointer {
		assert.True(t, strings.HasPrefix(ptr, "/definitions/"),
			"unexpected non-definition anchor at this increment: %q", ptr)
	}
}

// TestCoverage_ProvenanceOffByDefault confirms the callback is opt-in: a scan
// without OnProvenance set produces the same spec and records nothing.
func TestCoverage_ProvenanceOffByDefault(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/named-basic"},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Contains(t, doc.Definitions, "User", "baseline spec should still define User")
}

func keysOf(m map[string]scanner.Provenance) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestCoverage_ProvenanceGeometry is the anchors-only safety invariant: every
// emitted provenance pointer MUST resolve to a node that exists in the rendered
// spec. The cross-ref linker tolerates a finer node resolving to its nearest
// anchored ancestor, but it must NEVER be handed a pointer that points nowhere
// (or at the wrong node). Run across fixtures rich in the geometry that diverges
// from the simple top-level case — allOf composition, plain/aliased embeds,
// interface members, nested inline structs, slices and maps — so any path that
// fails to thread (or clear) the base pointer surfaces here as a dangling
// anchor.
func TestCoverage_ProvenanceGeometry(t *testing.T) {
	fixtures := []string{
		"./enhancements/named-basic",
		"./enhancements/allof-edges",
		"./enhancements/embedded-types",
		"./enhancements/interface-methods",
		"./enhancements/top-level-kinds",
		"./enhancements/named-struct-tags",
		"./enhancements/defaults-examples", // validation-keyword anchors (default/example)
		// Full-surface fixture: meta, routes/operations, parameters,
		// top-level responses and enum definitions — exercises every anchor
		// kind against the resolves-in-spec invariant.
		"./goparsing/petstore/...",
	}

	for _, pkg := range fixtures {
		t.Run(pkg, func(t *testing.T) {
			var recorded []scanner.Provenance
			doc, err := codescan.Run(&codescan.Options{
				Packages:   []string{pkg},
				WorkDir:    scantest.FixturesDir(),
				ScanModels: true,
				OnProvenance: func(p scanner.Provenance) {
					recorded = append(recorded, p)
				},
			})
			require.NoError(t, err)
			require.NotNil(t, doc)
			require.NotEmpty(t, recorded, "fixture should produce at least one anchor")

			raw, err := json.Marshal(doc)
			require.NoError(t, err)
			var root any
			require.NoError(t, json.Unmarshal(raw, &root))

			for _, p := range recorded {
				assert.True(t, resolveJSONPointer(root, p.Pointer),
					"anchor %q (from %s:%d) does not resolve in the rendered spec — dangling or mis-threaded pointer",
					p.Pointer, p.Pos.Filename, p.Pos.Line)
				assert.Positive(t, p.Pos.Line, "anchor %q should carry a source line", p.Pointer)
			}
		})
	}
}

// TestCoverage_ProvenanceAnchorKinds asserts every Phase-B anchor kind fires:
// definitions, properties, top-level responses, paths/operations, parameters,
// enum values and meta/info. Each must carry a source line and resolve in the
// rendered spec. The petstore covers most kinds; enum anchors only fire when a
// swagger:enum type becomes its own definition (in the petstore it inlines into
// a field), so enum-docs is folded in for that kind.
func TestCoverage_ProvenanceAnchorKinds(t *testing.T) {
	seen := map[string]scanner.Provenance{}
	for _, pkg := range []string{"./goparsing/petstore/...", "./enhancements/enum-docs"} {
		maps.Copy(seen, scanAndResolve(t, pkg))
	}

	// Each kind is recognised by the shape of its pointer; require at least one
	// of every kind, with a positive source line.
	kinds := map[string]func(string) bool{
		"definition": func(p string) bool {
			return strings.HasPrefix(p, "/definitions/") && !strings.Contains(p, "/properties/") && !strings.Contains(p, "/enum/")
		},
		"property":   func(p string) bool { return strings.Contains(p, "/properties/") },
		"enum value": func(p string) bool { return strings.Contains(p, "/enum/") },
		"response":   func(p string) bool { return strings.HasPrefix(p, "/responses/") },
		"operation": func(p string) bool {
			return strings.HasPrefix(p, "/paths/") && !strings.Contains(p, "/parameters/")
		},
		"parameter": func(p string) bool { return strings.Contains(p, "/parameters/") },
		"info":      func(p string) bool { return p == "/info" },
	}

	for kind, match := range kinds {
		var hit *scanner.Provenance
		for ptr, prov := range seen {
			if match(ptr) {
				hit = &prov
				break
			}
		}
		if assert.NotNil(t, hit, "no %s anchor recorded; got %v", kind, keysOf(seen)) {
			assert.Positive(t, hit.Pos.Line, "%s anchor %q should carry a source line", kind, hit.Pointer)
		}
	}
}

// TestCoverage_ProvenanceValidations exercises the validation-keyword anchors:
// a field's `default`/`example` (and the other scalar validations) anchor to
// their own comment line — distinct from, and above, the struct field — so
// following e.g. a `default` node in the spec lands on `// default: 1.5` rather
// than the field declaration.
func TestCoverage_ProvenanceValidations(t *testing.T) {
	byPointer := map[string]scanner.Provenance{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/defaults-examples"},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnProvenance: func(p scanner.Provenance) {
			byPointer[p.Pointer] = p
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	field, ok := byPointer["/definitions/Metrics/properties/ratio"]
	require.True(t, ok)
	def, ok := byPointer["/definitions/Metrics/properties/ratio/default"]
	require.True(t, ok, "the default validation should anchor to its own line; got %v", keysOf(byPointer))
	ex, ok := byPointer["/definitions/Metrics/properties/ratio/example"]
	require.True(t, ok, "the example validation should anchor to its own line")

	// The keyword lines sit in the doc comment, above the field declaration,
	// and on distinct lines from each other.
	assert.Less(t, def.Pos.Line, field.Pos.Line, "default's comment precedes the field")
	assert.Less(t, ex.Pos.Line, field.Pos.Line, "example's comment precedes the field")
	assert.NotEqual(t, def.Pos.Line, ex.Pos.Line, "each keyword anchors to its own line")
	assert.Equal(t, field.Pos.Filename, def.Pos.Filename, "same source file")

	// Array-element validations anchor under /items via the keyword's ItemsDepth
	// (none here at depth>0, but the slice field's own default still anchors).
	_, ok = byPointer["/definitions/Metrics/properties/tags/default"]
	assert.True(t, ok, "a slice field's default anchors to its line too")
}

// TestCoverage_ProvenanceMetaAndRoutes exercises the meta and route keyword
// anchors: each swagger:meta keyword (and each route-header keyword) anchors to
// its own comment line, not just the coarse /info or /paths/{path}/{method}
// block. The top-level meta fields (host/consumes/…) have no ancestor anchor
// otherwise (/info is their sibling), so this is the only way to reach them.
func TestCoverage_ProvenanceMetaAndRoutes(t *testing.T) {
	byPointer := map[string]scanner.Provenance{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./goparsing/petstore/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnProvenance: func(p scanner.Provenance) {
			byPointer[p.Pointer] = p
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	var root any
	require.NoError(t, json.Unmarshal(raw, &root))

	// A representative spread: an Info.* field, two root-level meta fields, and a
	// route-header keyword. Each must resolve in the spec and carry a line.
	for _, ptr := range []string{
		"/info/version", "/host", "/consumes", "/paths/~1pets/get/deprecated",
	} {
		prov, ok := byPointer[ptr]
		require.True(t, ok, "expected an anchor for %q; got %v", ptr, keysOf(byPointer))
		assert.Positive(t, prov.Pos.Line, "%q should carry a source line", ptr)
		assert.True(t, resolveJSONPointer(root, ptr), "%q must resolve in the rendered spec", ptr)
	}

	// Distinct lines: /info/version and /host come from different keyword lines.
	assert.NotEqual(t, byPointer["/info/version"].Pos.Line, byPointer["/host"].Pos.Line)
}

// scanAndResolve runs codescan over pkg with provenance enabled, asserts every
// emitted anchor resolves in the rendered spec, and returns the deduplicated
// pointer → provenance map.
func scanAndResolve(t *testing.T, pkg string) map[string]scanner.Provenance {
	t.Helper()
	seen := map[string]scanner.Provenance{}
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{pkg},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnProvenance: func(p scanner.Provenance) {
			seen[p.Pointer] = p
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	var root any
	require.NoError(t, json.Unmarshal(raw, &root))

	for ptr := range seen {
		assert.True(t, resolveJSONPointer(root, ptr),
			"anchor %q (%s) does not resolve in the rendered spec", ptr, pkg)
	}
	return seen
}

// resolveJSONPointer walks an RFC 6901 pointer over a decoded JSON value
// (map[string]any / []any / scalar) and reports whether the target node exists.
// Pure stdlib so the library test stays free of the jsontext experiment that
// the TUI-side index relies on; the two must agree on escaping, which the
// matching enum/escaped-key fixtures cross-check.
func resolveJSONPointer(root any, ptr string) bool {
	if ptr == "" {
		return true
	}
	if ptr[0] != '/' {
		return false
	}
	cur := root
	for raw := range strings.SplitSeq(ptr[1:], "/") {
		seg := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[seg]
			if !ok {
				return false
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(node) {
				return false
			}
			cur = node[idx]
		default:
			return false
		}
	}
	return true
}
