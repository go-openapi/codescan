// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/cmd/genspec-tui/internal/ux/panels"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// D3 — the whole chain against a REAL scan.
//
// Every other test in this package feeds the model hand-written JSON. That
// proves the indexes and the navigation agree with each other, but not that
// either agrees with what codescan actually emits: whether $refs arrive bare or
// allOf-wrapped, whether definition names survive as written, whether the
// provenance pointers line up with the rendered document. This scans the
// petstore fixture and drives the finished model over the result.

// fixturesDir resolves the repo-level fixtures/ directory from this file's own
// location, so the test runs from any working directory (CI runs it from
// cmd/genspec-tui, not the repo root). Deliberately local rather than borrowing
// scantest.FixturesDir — the TUI module should not grow a dependency on the
// library's test helpers for one path join.
func fixturesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot resolve the caller's file path")

	// thisFile is <repo>/cmd/genspec-tui/internal/ux/model_e2e_test.go
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "fixtures"))
}

// petstoreScan caches the scan across the whole package. A real scan means a
// real packages.Load, which costs seconds under -race; running one per test
// took this package from ~1s to ~36s. The result is immutable, and each test
// still gets its own Model built from it. Mirrors the caching the library's own
// scantest helpers do for the same reason.
var (
	petstoreOnce sync.Once     //nolint:gochecknoglobals // test-only scan cache
	petstoreRes  scanResultMsg //nolint:gochecknoglobals // test-only scan cache
)

// scanPetstore hands the cached scan to a fresh Model through the same message
// the bubbletea loop delivers.
func scanPetstore(t *testing.T) *Model {
	t.Helper()

	petstoreOnce.Do(func() {
		petstoreRes = doScan(codescan.Options{
			WorkDir:    fixturesDir(t),
			Packages:   []string{"./goparsing/petstore/..."},
			ScanModels: true,
		})
	})
	res := petstoreRes
	require.NoError(t, res.err, "the petstore fixture must scan cleanly")
	require.NotEmpty(t, res.json)

	m := &Model{
		spec:        panels.NewSpec(),
		fileView:    panels.NewFileView(),
		searchInput: textinput.New(),
	}
	m.cfg.WorkDir = fixturesDir(t) // so relTo renders paths relative to the scan root
	m.spec.SetSize(100, 30)
	m.fileView.SetSize(100, 30)
	m.focused = paneSpec
	_, _ = m.Update(res)

	return m
}

// specLines is the rendered document the indexes were built from.
func specLines(m *Model) []string { return strings.Split(m.specJSON, "\n") }

func TestE2E_RefIndexMatchesTheRenderedSpec(t *testing.T) {
	m := scanPetstore(t)
	lines := specLines(m)

	// The petstore references /definitions/pet from several operations. The
	// exact count is fixture-dependent, so assert the property that matters.
	sites := m.refIndex.RefsToPointer("/definitions/pet")
	require.GreaterOrEqual(t, len(sites), 2, "pet must be referenced from at least two places")

	var last int
	for i, site := range sites {
		require.Less(t, site.Line, len(lines), "site line is inside the document")

		// Every recorded site must really be a $ref pointing where we claim.
		text := lines[site.Line]
		assert.Contains(t, text, `"$ref"`, "line %d", site.Line)
		assert.Contains(t, text, "#/definitions/pet", "line %d", site.Line)

		if i > 0 {
			assert.Greater(t, site.Line, last, "sites are ordered by rendered line")
		}
		last = site.Line

		// And the node holding it must be addressable in the spec index.
		_, ok := m.specIndex.LineForPointer(site.Pointer)
		assert.True(t, ok, "holder %q is a real node", site.Pointer)
	}
}

// codescan emits $refs to responses as well as definitions; both must index.
func TestE2E_ResponseRefsIndex(t *testing.T) {
	m := scanPetstore(t)

	sites := m.refIndex.RefsToPointer("/responses/genericError")
	require.GreaterOrEqual(t, len(sites), 2, "the shared error response is reused across operations")

	for _, site := range sites {
		assert.True(t, site.Target.Local)
		assert.Equal(t, "/responses/genericError", site.Target.Pointer)
	}
}

// The round trip: park on a definition, F3 to one of its uses, Enter to come
// back. If either index disagreed with the render, this would land elsewhere.
func TestE2E_CycleThenGoToDefinitionRoundTrips(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	// F3 → the first use.
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	require.Contains(t, m.refStatus, "of /definitions/pet")
	firstUse := m.spec.TopLine()

	// F3 again → a different use.
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.NotEqual(t, firstUse, m.spec.TopLine(), "the cycle advanced to another site")
	require.Contains(t, m.refStatus, "ref 2/")

	// Enter on that $ref → back to the definition.
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, "→ /definitions/pet", m.notice)
	assert.Contains(t, specLines(m)[defLine], `"pet"`,
		"the line we came back to really is the pet definition")
}

func TestE2E_CycleVisitsEverySiteExactlyOnce(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	want := m.refIndex.RefsToPointer("/definitions/pet")
	require.NotEmpty(t, want)

	seen := make(map[int]int, len(want))
	for range want {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
		seen[m.refSites[m.refCursor].Line]++
	}

	assert.Len(t, seen, len(want), "one full lap visits every site")
	for line, n := range seen {
		assert.Equal(t, 1, n, "site at line %d visited once", line)
	}

	// One more step wraps back to the start.
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyF3})
	assert.Contains(t, m.refStatus, "ref 1/")
}

// Both renders of the same scan must find the same reference sites — only the
// line numbers differ. This is what makes ctrl+j/ctrl+y safe mid-investigation.
func TestE2E_YAMLFindsTheSameSites(t *testing.T) {
	m := scanPetstore(t)
	require.NotEmpty(t, m.specYAML, "the scan produced a YAML render")

	jsonHolders := holderSet(m, "/definitions/pet")
	require.NotEmpty(t, jsonHolders)

	m.setSpecFormat("YAML")
	require.Equal(t, "YAML", m.spec.Format())

	assert.Equal(t, jsonHolders, holderSet(m, "/definitions/pet"),
		"the same nodes reference pet in either render")
}

func holderSet(m *Model, target string) map[string]bool {
	out := make(map[string]bool)
	for _, site := range m.refIndex.RefsToPointer(target) {
		out[site.Pointer] = true
	}

	return out
}

// The two halves of the linker must agree on a real scan: a definition the ref
// index points at should also be a node the provenance index can take to source.
func TestE2E_RefTargetsHaveSource(t *testing.T) {
	m := scanPetstore(t)
	require.Positive(t, m.srcIndex.Len(), "the scan emitted provenance")

	for _, target := range []string{"/definitions/pet", "/definitions/order"} {
		require.NotEmpty(t, m.refIndex.RefsToPointer(target), "%s is referenced", target)

		pos, ok := m.srcIndex.PositionFor(target)
		require.True(t, ok, "%s resolves to source", target)
		assert.True(t, strings.HasSuffix(pos.Filename, ".go"), "%s → %s", target, pos.Filename)
		assert.Positive(t, pos.Line)
	}
}

// Spec→source follow, end to end: park on a definition, turn on follow, and the
// source pane must open the Go file that actually declares it.
func TestE2E_FollowOpensTheDeclaringFile(t *testing.T) {
	m := scanPetstore(t)

	defLine, ok := m.specIndex.LineForPointer("/definitions/pet")
	require.True(t, ok)
	m.spec.SetCursor(defLine)

	m.toggleFollow(followSpec)

	require.Equal(t, followSpec, m.follow)
	assert.True(t, strings.HasSuffix(m.currentFile, ".go"), "opened %q", m.currentFile)
	assert.Contains(t, m.followTarget, "/definitions/pet")
	assert.Contains(t, m.fileView.Content(), "type Pet struct",
		"the follower landed in the file that declares the type")
}
