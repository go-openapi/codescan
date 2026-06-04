// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Loading axes of the scanner measurement harness.
//
//	Axis A — loader strategy:        go/packages vs internal/packages (ToolchainFreeLoader)
//	Axis B — dependency types:       read from source vs taken from compiled export data
//	Axis C — bundled export data:    a pre-computed blob, the toolchain-free route
//	Axis D — the raw LoadMode ladder: what go/packages charges for each capability
//	Axis E — the SHAPE of a load:     closure size, files, types, syntax — not just the cost
//
// Axis E is not a separate rung: every rung reports it, because the cost of a load is only half of
// what distinguishes one rung from another. The state this stream exists to serve — a package with
// complete types and NO AST — is invisible in a timing and obvious in a shape.
//
// See bench_scanner_axes.md for how to run these and how to read the output.

import (
	"archive/zip"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"

	ownpackages "github.com/go-openapi/codescan/internal/packages"
	"github.com/go-openapi/testify/v2/require"
)

const (
	// envColdCache asks for the cold-cache pass as well as the warm one. Opt-in because each cold
	// rung compiles a whole dependency closure from nothing.
	envColdCache = "CODESCAN_BENCH_COLD"

	// envExportData points at an export-data blob (a directory, or a .zip) for Axis C, and is the
	// fallback for a corpus with no blob variable of its own. Produce one with hack/genexportdata;
	// see the message exportDataBlob prints for the exact invocation.
	envExportData = "CODESCAN_BENCH_EXPORTDATA"

	// envSpotlight names one dependency to describe in detail under each rung. The default is the
	// dependency whose annotations the go/packages export-data route gives up, which is the case
	// the whole stream turns on.
	envSpotlight = "CODESCAN_BENCH_SPOTLIGHT"

	defaultSpotlight = "github.com/go-openapi/strfmt"
)

// loadResult is what a rung produced: the graph, plus the loader's own report of how much of that
// graph it could not read at all.
//
// exportOnly comes from WithOnExportOnly rather than from the packages, because it is a statement
// about what was NOT there — a package whose types were served and whose source could not be found.
// Nothing in the resulting *Package distinguishes that from a package deliberately left unparsed.
type loadResult struct {
	pkgs       []*packages.Package
	exportOnly int
}

// loadRung is one measurable way of loading a package graph.
//
// env is the environment the load runs under, and is what makes the cold/warm distinction
// expressible: a rung handed a fresh GOCACHE has to compile whatever export data it wants, where
// the same rung on the ambient environment finds it already built.
type loadRung struct {
	axis string
	name string
	load func(corpus benchCorpus, env []string) (loadResult, error)
}

// goPackagesLadder is Axis D: the raw go/packages LoadMode ladder, from everything the scanner asks
// for today down to just the file list.
//
// The gap between the first and the last row is the budget this stream is spending. Every row below
// the first is a capability we still have at a fraction of the cost — PROVIDED discovery and
// loading are separated, which is the stream's thesis.
func goPackagesLadder() []loadRung {
	return []loadRung{
		{
			axis: "D",
			name: "current (Deps+Types+Syntax+TypesInfo)",
			load: rawLoad(packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps |
				packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo),
		},
		{
			axis: "D",
			name: "all types, no syntax anywhere",
			load: rawLoad(packages.NeedName | packages.NeedFiles | packages.NeedImports |
				packages.NeedDeps | packages.NeedTypes),
		},
		{
			// The same mode internal/packages calls compiledDepsMode: dropping NeedDeps is what makes
			// go/packages ask `go list` for -export and take dependency types from the compiler.
			axis: "D",
			name: "root syntax+types, dep types from export data",
			load: rawLoad(packages.NeedName | packages.NeedFiles | packages.NeedImports |
				packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo),
		},
		{
			axis: "D",
			name: "syntax only, no types",
			load: rawLoad(packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedSyntax),
		},
		{
			axis: "D",
			name: "go list only (name+files+imports+deps)",
			load: rawLoad(packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps),
		},
	}
}

// rawLoad builds a rung that calls go/packages directly, bypassing internal/packages.
func rawLoad(mode packages.LoadMode) func(benchCorpus, []string) (loadResult, error) {
	return func(corpus benchCorpus, env []string) (loadResult, error) {
		pkgs, err := packages.Load(&packages.Config{Dir: corpus.dir, Mode: mode, Env: env}, corpus.pattern)

		return loadResult{pkgs: pkgs}, err
	}
}

// loaderLadder is Axes A, B and C: the same question asked of internal/packages, which is what the
// scanner actually runs.
//
// The raw ladder above says what is on offer; this says what is taken.
func loaderLadder(tb testing.TB, corpus benchCorpus) []loadRung {
	tb.Helper()

	ladder := []loadRung{
		{axis: "A", name: "loader: go/packages (default)", load: ownLoad()},
		{
			axis: "B",
			name: "loader: go/packages + CompiledDependencies",
			load: ownLoad(ownpackages.WithCompiledDependencies()),
		},
		{
			axis: "A",
			name: "loader: toolchain-free",
			load: ownLoad(ownpackages.WithStrategy(ownpackages.StrategyToolchainFree)),
		},
	}

	if blob := exportDataBlob(tb, corpus); blob != nil {
		ladder = append(ladder, loadRung{
			axis: "C",
			name: "loader: toolchain-free + ExportData",
			load: ownLoad(
				ownpackages.WithStrategy(ownpackages.StrategyToolchainFree),
				ownpackages.WithExportData(blob),
			),
		})
	}

	return ladder
}

// ownLoad builds a rung that goes through internal/packages, the seam the scanner uses.
//
// WithOnExportOnly is registered unconditionally: on the rungs where it fires it is the third state
// of Axis E, and on the rungs where it does not, a zero is itself the finding.
func ownLoad(opts ...ownpackages.Option) func(benchCorpus, []string) (loadResult, error) {
	return func(corpus benchCorpus, env []string) (loadResult, error) {
		var res loadResult
		all := append([]ownpackages.Option{
			ownpackages.WithOnExportOnly(func(ownpackages.ExportOnly) { res.exportOnly++ }),
		}, opts...)

		pkgs, err := ownpackages.NewLoader(all...).
			Load(&packages.Config{Dir: corpus.dir, Env: env}, corpus.pattern)
		res.pkgs = pkgs

		return res, err
	}
}

// exportDataBlob opens this corpus's Axis C blob, or returns nil after saying how to make one.
//
// The blob is not checked in: it is toolchain-version-specific (the export format is tied to the Go
// release), and it covers ONE corpus's dependency closure. Handing a corpus somebody else's blob
// does not fail — every package it does not cover simply falls back to source — so the harness
// takes a blob per corpus rather than pretending one file serves all of them.
func exportDataBlob(tb testing.TB, corpus benchCorpus) fs.FS {
	tb.Helper()

	path := envOr(corpus.exportEnv, os.Getenv(envExportData))
	if path == "" {
		tb.Logf("Axis C not measured for %s: no export-data blob. Produce one and set %s:\n"+
			"    go run ./hack/genexportdata -dir %s -out /tmp/%s-exportdata.zip %s\n"+
			"    %s=/tmp/%s-exportdata.zip",
			corpus.name, corpus.exportEnv,
			corpus.dir, corpus.name, corpus.pattern,
			corpus.exportEnv, corpus.name)

		return nil
	}

	// The path is operator-supplied, on purpose: this whole harness is pointed at trees outside the
	// repo. There is nothing to sanitise it against.
	info, err := os.Stat(path) //nolint:gosec // an operator-supplied measurement input, not user data
	if err != nil {
		tb.Logf("Axis C not measured for %s: %s=%q is unreadable: %v", corpus.name, corpus.exportEnv, path, err)

		return nil
	}

	if info.IsDir() {
		return os.DirFS(path)
	}

	// archive/zip's Reader is already an fs.FS, which is why genexportdata can emit one file that
	// still serves per-package reads.
	r, err := zip.OpenReader(path)
	if err != nil {
		tb.Logf("Axis C not measured for %s: %q is neither a directory nor a readable zip: %v", corpus.name, path, err)

		return nil
	}
	tb.Cleanup(func() { _ = r.Close() })

	return r
}

// measureRung runs one rung and reports what it cost and what shape it produced.
//
// warm decides whether the rung gets a discarded practice run first. That is the whole cold/warm
// apparatus: without it the report would silently show whichever state the build cache happened to
// be in, and for the export-data rungs those two states differ by an order of magnitude.
func measureRung(tb testing.TB, rung loadRung, corpus benchCorpus, env []string, warm bool) {
	tb.Helper()

	if warm {
		if _, err := rung.load(corpus, env); err != nil {
			tb.Fatalf("%s: warm-up load: %v", rung.name, err)
		}
		runtime.GC()
	}

	base := retainedHeap()

	start := time.Now()
	res, err := rung.load(corpus, env)
	elapsed := time.Since(start)
	require.NoErrorf(tb, err, "%s: load", rung.name)

	shape := shapeOf(res.pkgs)
	shape.exportOnly = res.exportOnly

	reportRow(tb, "["+rung.axis+"] "+rung.name, elapsed, retainedSince(base), shape.String())
	reportSpotlight(tb, res.pkgs)

	runtime.KeepAlive(res.pkgs)
}

// reportSpotlight describes one dependency under the current rung.
//
// The aggregate shape says how many packages have types without an AST; this says WHICH, and is
// what turns a column of counts into "strfmt has nine readable files, complete types and no AST" —
// the sentence the on-demand deliverable is written against.
func reportSpotlight(tb testing.TB, roots []*packages.Package) {
	tb.Helper()

	want := os.Getenv(envSpotlight)
	if want == "" {
		want = defaultSpotlight
	}

	p, ok := walkGraph(roots)[want]
	if !ok {
		return
	}

	tb.Logf("      %s: GoFiles=%d  Types=%v  Syntax=%d",
		want, len(p.GoFiles), p.Types != nil && p.Types.Complete(), len(p.Syntax))
}

// TestBenchLoadLadders is Axes A–E, warm.
//
// Every rung is preceded by a discarded practice run, so a number in this report is always a
// steady-state number. TestBenchLoadLaddersCold is the other half.
func TestBenchLoadLadders(t *testing.T) {
	requireBenchEnabled(t)

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.name, func(t *testing.T) {
			t.Logf("=== load ladders, WARM cache — %s (%s %s) ===", corpus.name, corpus.dir, corpus.pattern)

			for _, rung := range goPackagesLadder() {
				measureRung(t, rung, corpus, nil, true)
			}
			for _, rung := range loaderLadder(t, corpus) {
				measureRung(t, rung, corpus, nil, true)
			}
		})
	}
}

// TestBenchLoadLaddersCold is the cold-cache half of Axis B, and the reason the harness reports a
// cache state at all.
//
// Export data does not exist until the compiler has produced it, so the rungs that take dependency
// types from it pay for that compilation the first time they are asked. Warm, CompiledDependencies
// is the cheapest useful rung on the ladder; cold, it is the most expensive. Reporting one number
// without saying which would be worse than reporting none.
//
// "Cold" here means a private, empty GOCACHE — a well-defined extreme rather than "whatever had not
// been built yet". Only GOCACHE is redirected; the module cache stays shared, so nothing is
// downloaded. Each rung gets its own, so no rung warms another.
//
// It is opt-in because an empty cache means compiling a whole closure, standard library included.
func TestBenchLoadLaddersCold(t *testing.T) {
	requireBenchEnabled(t)

	if os.Getenv(envColdCache) != "1" {
		t.Skipf("%s!=1: skipping the cold-cache pass (it compiles a whole closure per rung)", envColdCache)
	}

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.name, func(t *testing.T) {
			// Only the rungs whose cost depends on the build cache. A load that reads dependency
			// source compiles nothing, so a fresh GOCACHE would measure the same thing twice.
			raw := goPackagesLadder()
			coldRungs := []loadRung{
				raw[0], // the default, for contrast: cache state should barely move it
				raw[2], // compiledDepsMode — the rung that must compile before it can read
			}
			for _, rung := range loaderLadder(t, corpus) {
				if rung.axis == "B" {
					coldRungs = append(coldRungs, rung)
				}
			}

			t.Logf("=== load ladders, COLD (private empty GOCACHE) — %s ===", corpus.name)

			for _, rung := range coldRungs {
				env := append(os.Environ(), "GOCACHE="+t.TempDir())
				measureRung(t, rung, corpus, env, false)
			}
		})
	}
}

// BenchmarkLoadStrategies is Axes A–C as a true benchmark, for the times a like-for-like comparison
// across a code change is what is wanted rather than a table.
//
// Run it with -benchtime=1x: these rungs cost the better part of a second each, and the retained
// figure only means anything for a single iteration.
//
//	go test ./internal/integration/ -run x -bench BenchmarkLoadStrategies -benchtime=1x
func BenchmarkLoadStrategies(b *testing.B) {
	corpus := dockerctlCorpus(b)

	for _, rung := range loaderLadder(b, corpus) {
		b.Run(benchName(rung.name), func(b *testing.B) {
			// Warm the cache outside the timer: the first CompiledDependencies load compiles the
			// closure's export data, which is a one-off and not what this benchmark is comparing.
			_, err := rung.load(corpus, nil)
			require.NoError(b, err)

			var shape loadShape
			base := retainedHeap()

			b.ResetTimer()
			for range b.N {
				res, err := rung.load(corpus, nil)
				require.NoError(b, err)
				b.StopTimer()
				shape = shapeOf(res.pkgs)
				b.StartTimer()
			}
			b.StopTimer()

			b.ReportMetric(mbOf(retainedSince(base)), "MB-retained")
			b.ReportMetric(float64(shape.graph), "pkgs")
			b.ReportMetric(float64(shape.withSyntax), "pkgs-from-source")
		})
	}
}

// benchName renders a rung name as a benchmark sub-name (no spaces, which benchstat cannot parse).
func benchName(s string) string {
	s = strings.TrimPrefix(s, "loader: ")

	return strings.NewReplacer(" ", "_", "/", "-", "+", "plus").Replace(s)
}
