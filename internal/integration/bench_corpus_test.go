// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Corpus resolution and measurement primitives shared by the scanner measurement harness.
//
// The harness answers one question in several dimensions: what does a scan pay, and what does it
// get for it. Its axes are laid out in bench_scanner_axes.md, which is also where the reference
// figures live. This file holds only what every axis needs:
//
//   - which external corpus to measure, and how to skip cleanly when it is not checked out;
//   - how to time a phase and read the heap it RETAINS rather than the bytes it churned;
//   - how to describe the SHAPE of a loaded graph, which is the axis that makes the on-demand
//     stream's central state visible — a package with files and types but no AST.
//
// Everything here is test-only and imports nothing that production code does not already use.

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"
)

// Environment knobs. Every one of them has a working default; they exist so the harness can be
// pointed at a different checkout without editing source.
const (
	// envBenchEnabled gates the reporting tests, which take minutes and need an external corpus.
	// They are tests rather than benchmarks because what they produce is a table, not a scalar
	// (see bench_scanner_axes.md, "Why some axes are tests").
	envBenchEnabled = "CODESCAN_BENCH"

	// envDockerctlDir overrides where the dockerctl corpus is checked out.
	// Named without a corpus suffix for compatibility: it predates the second corpus.
	envDockerctlDir = "CODESCAN_BENCH_DIR"

	// envGoSwaggerDir overrides where the go-swagger corpus is checked out.
	envGoSwaggerDir = "CODESCAN_BENCH_GOSWAGGER_DIR"
)

// Per-corpus export-data blobs (Axis C). A blob covers one corpus's dependency closure and no
// other, so pointing every corpus at the same file would silently measure a fallback-to-source run
// and call it export data. envExportData is honoured as a fallback for the single-corpus case.
const (
	envDockerctlExportData = "CODESCAN_BENCH_EXPORTDATA_DOCKERCTL"
	envGoSwaggerExportData = "CODESCAN_BENCH_EXPORTDATA_GOSWAGGER"
)

// Default checkout locations of the two external corpora. Neither is vendored: they are large
// third-party trees, and the point of measuring them is that they are real.
const (
	// defaultDockerctlDir is the go-swagger-generated docker engine API client — ~500 Go files
	// over a 336-package closure, of which two packages carry annotations.
	defaultDockerctlDir = "/home/fred/src/github.com/go-swagger/dockerctl"

	// defaultGoSwaggerDir is go-swagger itself — a hand-written tree with a 440-package closure,
	// and a second shape to check any conclusion against.
	defaultGoSwaggerDir = "/home/fred/src/github.com/go-swagger/go-swagger"
)

// bytesPerMB is the only unit any of these measurements is legible in.
const bytesPerMB = 1 << 20

// benchCorpus is one external tree to measure, and everything needed to measure it.
type benchCorpus struct {
	name      string
	dir       string
	pattern   string
	exportEnv string // env var naming this corpus's export-data blob (Axis C)
}

// knownCorpora is the whole corpus table. Adding a third tree is adding a row.
func knownCorpora() []benchCorpus {
	return []benchCorpus{
		{
			name:      "dockerctl",
			dir:       envOr(envDockerctlDir, defaultDockerctlDir),
			pattern:   "./...",
			exportEnv: envDockerctlExportData,
		},
		{
			name:      "go-swagger",
			dir:       envOr(envGoSwaggerDir, defaultGoSwaggerDir),
			pattern:   "./...",
			exportEnv: envGoSwaggerExportData,
		},
	}
}

func envOr(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}

	return fallback
}

// dockerctlCorpus resolves the dockerctl checkout and SKIPS the caller when it is absent. Absence
// is the normal case in CI and must never be an error.
func dockerctlCorpus(tb testing.TB) benchCorpus {
	tb.Helper()

	corpus := knownCorpora()[0]
	if _, err := os.Stat(corpus.dir); err != nil {
		tb.Skipf("corpus not found at %q (set %s): %v", corpus.dir, envDockerctlDir, err)
	}

	return corpus
}

// dockerctlDir is the same resolution as a bare directory, for the callers that only need a path.
func dockerctlDir(tb testing.TB) string {
	tb.Helper()

	return dockerctlCorpus(tb).dir
}

// benchCorpora returns every corpus present on this machine, skipping the caller when none is.
//
// Callers that need one specific tree use dockerctlCorpus directly; this is for the axes whose
// whole point is that a conclusion holds on more than one shape of project.
func benchCorpora(tb testing.TB) []benchCorpus {
	tb.Helper()

	var found []benchCorpus
	for _, c := range knownCorpora() {
		if _, err := os.Stat(c.dir); err == nil {
			found = append(found, c)
		} else {
			tb.Logf("corpus %s absent at %q — not measured", c.name, c.dir)
		}
	}

	if len(found) == 0 {
		tb.Skipf("no measurement corpus checked out (set %s and/or %s)", envDockerctlDir, envGoSwaggerDir)
	}

	return found
}

// requireBenchEnabled skips unless the operator asked for the measurement explicitly.
//
// These reporting tests load whole third-party package graphs repeatedly; on a machine where the
// corpora ARE checked out, letting them run under a plain `go test ./...` would add minutes to
// every unrelated run.
func requireBenchEnabled(tb testing.TB) {
	tb.Helper()

	if os.Getenv(envBenchEnabled) != "1" {
		tb.Skipf("%s!=1: skipping the scanner measurement report (see bench_scanner_axes.md)", envBenchEnabled)
	}
}

// mbOf renders a byte count in MB.
func mbOf(v uint64) float64 { return float64(v) / bytesPerMB }

// retainedHeap returns the currently retained heap, after a GC so the figure reflects what is HELD
// rather than what was allocated on the way.
//
// Retained heap is the number this stream is steered by: the scanner's problem is that it keeps the
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

// allocBaseline takes the cumulative-allocation reading that allocatedSince is measured against.
func allocBaseline() uint64 {
	runtime.GC()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.TotalAlloc
}

// allocatedSince returns the bytes allocated since a baseline — the CHURN figure, which is the
// right one for phases whose whole point is that they retain nothing (discovery, the prefilter).
func allocatedSince(base uint64) uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return m.TotalAlloc - base
}

// loadShape describes what a load produced, not what it cost.
//
// This is the axis that makes the on-demand stream's central state visible, and since the
// per-dependency export-data policy landed it is a THREE-way state per package rather than two:
//
//   - fromSource   — types and syntax, the ordinary load;
//   - exportServed — complete types, no AST. The package said nothing about its own types, so its
//     source was never parsed. This is what a scan holds instead of a package;
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
// figure the retained heap corresponds to and is worth knowing on its own. The closure size for
// that corpus comes from a rung that walks the whole graph.
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

// isStdlibPath reports whether an import path belongs to the standard library — no dot in its first
// segment, the same test cmd/go uses. Worth splitting out in reports because stdlib is the bulk of
// any closure and carries no annotations.
func isStdlibPath(pkgPath string) bool {
	return !strings.Contains(strings.SplitN(pkgPath, "/", 2)[0], ".")
}

// reportRow is the one row format every cost table in the harness shares: what it was, what it
// cost in wall clock, and what it retained.
func reportRow(tb testing.TB, label string, elapsed time.Duration, retained uint64, tail string) {
	tb.Helper()

	tb.Logf("%-44s %7.2fs  retained=%6.0f MB  %s", label, elapsed.Seconds(), mbOf(retained), tail)
}
