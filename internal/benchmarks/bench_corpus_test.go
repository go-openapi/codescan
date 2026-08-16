// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package benchmarks_test

// Corpus resolution and measurement primitives shared by the scanner measurement harness.
//
// The harness answers one question in several dimensions: what does a scan pay, and what does it
// get for it. This file holds only what every measurement needs:
//
//   - which trees to measure, which is no longer a question — they ship with the repository and
//     unpack on demand (see internal/benchmarks/corpus);
//   - how to time a phase and read the heap it RETAINS rather than the bytes it churned;
//   - how to describe the SHAPE of a loaded graph, which is what makes a package with files and
//     types but no AST visible at all.
//
// Everything here is test-only and imports nothing that production code does not already use.

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/go-openapi/codescan/internal/benchmarks/corpus"
	"github.com/go-openapi/testify/v2/require"
)

// Environment knobs. Every one of them has a working default; they exist so a run can be steered
// without editing source.
const (
	// envBenchEnabled gates the reporting tests, which load whole package graphs repeatedly. They
	// are tests rather than benchmarks because what they produce is a table, not a scalar.
	envBenchEnabled = "CODESCAN_BENCH"

	// Per-corpus export-data blobs. A blob covers one corpus's dependency closure and no other, so
	// pointing every corpus at the same file would silently measure a fallback-to-source run and
	// call it export data.
	envDockerctlExportData = "CODESCAN_BENCH_EXPORTDATA_DOCKERCTL"
	envKubeapiExportData   = "CODESCAN_BENCH_EXPORTDATA_KUBEAPI"
)

// bytesPerMB is the only unit any of these measurements is legible in.
const bytesPerMB = 1 << 20

// benchCorpus is one tree to measure, and everything needed to measure it.
type benchCorpus struct {
	corpus.Corpus

	exportEnv string // env var naming this corpus's export-data blob
}

// exportEnvFor names a corpus's blob variable. A corpus with no entry simply has no export-data
// rung.
func exportEnvFor(name string) string {
	return map[string]string{
		"dockerctl": envDockerctlExportData,
		"kubeapi":   envKubeapiExportData,
	}[name]
}

// benchCorpora unpacks the shipped corpora and returns them.
//
// Absence used to be the normal case, back when these were external checkouts named through the
// environment. It is now a failure: the archive is in the repository, so a corpus that cannot be
// resolved means something is wrong with the checkout rather than with the machine.
func benchCorpora(tb testing.TB) []benchCorpus {
	tb.Helper()

	found, err := corpus.Ensure()
	require.NoError(tb, err, "unpacking the benchmark corpora")

	out := make([]benchCorpus, 0, len(found))
	for _, c := range found {
		out = append(out, benchCorpus{Corpus: c, exportEnv: exportEnvFor(c.Name)})
	}

	return out
}

// benchEnv is the environment every load runs under.
//
// GOWORK=off is not optional. The corpora unpack INSIDE this repository, and the workspace at its
// root does not list them, so a load that inherits it fails outright:
//
//	directory prefix . does not contain modules listed in go.work or their selected dependencies
//
// Turning it off is also what keeps each corpus resolving its own dependency versions, which a
// workspace spanning both would merge. Everything else is inherited, so the operator's GOMODCACHE
// and toolchain still apply.
func benchEnv() []string {
	return append(os.Environ(), "GOWORK=off")
}

// envOr reads an environment variable, falling back when it is unset or empty.
func envOr(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}

	return fallback
}

// requireBenchEnabled skips unless the operator asked for the measurement explicitly.
//
// These reporting tests load whole third-party package graphs repeatedly; letting them run under a
// plain `go test ./...` would add minutes to every unrelated run.
func requireBenchEnabled(tb testing.TB) {
	tb.Helper()

	if os.Getenv(envBenchEnabled) != "1" {
		tb.Skipf("%s!=1: skipping the scanner measurement report (see README.md)", envBenchEnabled)
	}
}

// loadShape describes what a load produced, not what it cost.
//
// Since the per-dependency export-data policy landed this is a THREE-way state per package rather
// than two:
//
//   - fromSource   — types and syntax, the ordinary load;
//   - exportServed — complete types, no AST. The package said nothing about its own types, so its
//     source was never parsed;
//   - exportOnly   — served from export data AND its source could not be found at all, so nothing
//     could ever be read from it. The loader names this state itself
//     ([packages.ExportOnly] / WithOnExportOnly); the count is collected by the caller,
//     since it arrives as a callback rather than as a field.
//
// withFiles is reported alongside withSyntax on purpose: a package with files and no AST is a
// package whose source is LOCATABLE and unparsed, which is the difference between "not yet" and
// "never".
type loadShape struct {
	graph        int // packages reachable through the import graph
	withFiles    int // ... of which have GoFiles (so their source can be found later)
	withTypes    int // ... of which have complete type information
	withSyntax   int // ... of which have an AST (loaded from source)
	exportServed int // ... of which have types and no AST
	exportOnly   int // ... of which the loader reported as having no reachable source at all
	astFiles     int // total parsed files held
}

// shapeOf walks the whole import graph from roots and describes it.
//
// exportOnly is not derivable from the graph — it is a loader callback — so it is left zero here
// and filled in by whoever registered WithOnExportOnly.
//
// One number needs reading carefully. graph counts packages REACHABLE through Package.Imports, and
// a package served from export data has no Imports map to walk (internal/packages.exportedPackage
// builds an empty one — nothing downstream reads it). So under WithExportData graph does not mean
// "the size of the closure"; it means "how many packages the loader materialised", which is the
// figure the retained heap corresponds to and is worth knowing on its own.
func shapeOf(roots []*packages.Package) loadShape {
	all := walkGraph(roots)

	shape := loadShape{graph: len(all)}
	for _, p := range all {
		if len(p.GoFiles) > 0 {
			shape.withFiles++
		}

		typed := p.Types != nil && p.Types.Complete()
		if typed {
			shape.withTypes++
		}

		switch {
		case len(p.Syntax) > 0:
			shape.withSyntax++
			shape.astFiles += len(p.Syntax)
		case typed:
			shape.exportServed++
		}
	}

	return shape
}

// String renders a shape as the columns the reports share.
func (s loadShape) String() string {
	return fmt.Sprintf("graph=%3d  goFiles=%3d  types=%3d  fromSource=%3d  exportServed=%3d  exportOnly=%3d  astFiles=%d",
		s.graph, s.withFiles, s.withTypes, s.withSyntax, s.exportServed, s.exportOnly, s.astFiles)
}

// walkGraph returns every package reachable from roots through the import graph — the same
// traversal TypeIndex.build performs, and therefore the same set the current scanner retains.
func walkGraph(roots []*packages.Package) map[string]*packages.Package {
	all := make(map[string]*packages.Package, len(roots))

	var visit func(*packages.Package)
	visit = func(p *packages.Package) {
		if _, seen := all[p.PkgPath]; seen {
			return
		}
		all[p.PkgPath] = p
		for _, imp := range p.Imports {
			visit(imp)
		}
	}

	for _, p := range roots {
		visit(p)
	}

	return all
}

// reportRow is the one row format every cost table in the harness shares: what it was, what it
// cost in wall clock, and what it retained.
func reportRow(tb testing.TB, label string, elapsed time.Duration, retained uint64, tail string) {
	tb.Helper()

	tb.Logf("%-44s %7.2fs  retained=%6.0f MB  %s", label, elapsed.Seconds(), mbOf(retained), tail)
}

// mbOf renders a byte count in MB.
func mbOf(v uint64) float64 { return float64(v) / bytesPerMB }

// retainedHeap returns the currently retained heap, after a GC so the figure reflects what is HELD
// rather than what was allocated on the way.
//
// Retained heap is the number a load is steered by: the scanner's problem is that it keeps the
// whole package graph alive, not that it churns. Callers take a reading before and after and report
// the delta, so the figure means "what this rung holds" rather than "what this process holds".
func retainedHeap() uint64 {
	runtime.GC()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.HeapAlloc
}

// retainedSince returns the retained-heap growth since a baseline reading, floored at zero.
//
// The floor is not cosmetic: a rung can end with LESS live heap than it started with, because the
// fencing GC also collects whatever the previous rung had just dropped. A negative "retained" is
// noise, not information.
func retainedSince(base uint64) uint64 {
	now := retainedHeap()
	if now < base {
		return 0
	}

	return now - base
}
