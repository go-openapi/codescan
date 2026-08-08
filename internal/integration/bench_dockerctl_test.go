// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Cross-worktree comparison benchmark.
//
// This program scans a sizable, real-world generated client (the docker
// engine API client produced by go-swagger, at github.com/go-swagger/dockerctl)
// and reports the wall-clock time and heap allocations for a single full scan.
//
// It is intentionally self-contained: it only touches the public Run entry
// point plus the two internal phases (scanner.NewScanCtx + spec.NewBuilder),
// all of which exist identically in both the grammar-based worktree and the
// pre-grammar (v0.34.0, regexp-based) worktree. Copy this file verbatim into
// the other worktree's internal/integration directory and run the same
// command to compare.
//
// The scan splits into two phases:
//
//   - Load:  scanner.NewScanCtx — go/packages type-checking. Identical work
//            in both worktrees; serves as a stable baseline.
//   - Build: spec.NewBuilder(...).Build() — comment-annotation parsing and
//            Swagger emission. THIS is the phase that changed from regexp to
//            grammar; it is where any win or regression shows up.
//
//   - Full:  codescan.Run — the whole thing, fresh (Load+Build), for a
//            single end-to-end number.
//
// Run it (single iteration is enough):
//
//	go test ./internal/integration/ -run TestDockerctlBenchReport -v
//
// or as a true Go benchmark:
//
//	go test ./internal/integration/ -run x -bench BenchmarkDockerctl -benchtime=1x -benchmem
//
// Point it at the reference client with CODESCAN_BENCH_DIR if it is not at the
// default location below; the program skips (rather than fails) when absent.

import (
	"runtime"
	"testing"
	"time"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/builders/spec"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/require"
)

// dockerctlDir, and the corpus locations it resolves, live in bench_corpus_test.go — they are
// shared with the loader/discovery axes added alongside this benchmark.

func benchOptions(dir string) *codescan.Options {
	return &codescan.Options{
		WorkDir:    dir,
		Packages:   []string{"./..."},
		ScanModels: true,
	}
}

// phaseStats captures the cost of one measured phase.
type phaseStats struct {
	name    string
	elapsed time.Duration
	allocPB uint64 // bytes allocated during the phase (cumulative TotalAlloc delta)
	allocsN uint64 // number of heap allocations during the phase (Mallocs delta)
}

// measure runs fn once, fencing it with two GCs so the heap counters reflect
// only fn's own allocations, and records elapsed time + allocation deltas.
func measure(name string, fn func()) phaseStats {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	start := time.Now()
	fn()
	elapsed := time.Since(start)

	runtime.ReadMemStats(&after)

	return phaseStats{
		name:    name,
		elapsed: elapsed,
		allocPB: after.TotalAlloc - before.TotalAlloc,
		allocsN: after.Mallocs - before.Mallocs,
	}
}

func (p phaseStats) report(tb testing.TB) {
	tb.Helper()
	tb.Logf("%-6s  time=%-12s  alloc=%6.1f MB  allocs=%d",
		p.name,
		p.elapsed.Round(time.Microsecond),
		float64(p.allocPB)/(1024*1024),
		p.allocsN,
	)
}

// TestDockerctlBenchReport runs a single scan of the reference client and
// prints a human-readable timing + allocation report for each phase. This is
// the primary cross-worktree comparison vehicle: one iteration is enough.
func TestDockerctlBenchReport(t *testing.T) {
	dir := dockerctlDir(t)
	opts := benchOptions(dir)

	t.Logf("scanning reference client at %s", dir)

	// Phase split: Load then Build, sharing one ScanCtx so Build sees exactly
	// what Run would feed it.
	var ctx *scanner.ScanCtx
	loadStats := measure("Load", func() {
		var err error
		ctx, err = scanner.NewScanCtx(opts)
		require.NoError(t, err)
	})

	var sp any
	buildStats := measure("Build", func() {
		builder := spec.NewBuilder(opts.InputSpec, ctx, opts.ScanModels)
		built, err := builder.Build()
		require.NoError(t, err)
		sp = built
	})
	require.NotNil(t, sp)

	// Full end-to-end, fresh, for a single combined number.
	fullStats := measure("Full", func() {
		built, err := codescan.Run(opts)
		require.NoError(t, err)
		require.NotNil(t, built)
	})

	t.Log("=== dockerctl scan benchmark (single iteration) ===")
	loadStats.report(t)
	buildStats.report(t)
	fullStats.report(t)
}

// BenchmarkDockerctl is the same workload exposed to the Go benchmark
// framework. Run with -benchtime=1x -benchmem for a one-shot allocation
// profile, or with the default settings for a stabilized timing.
func BenchmarkDockerctl(b *testing.B) {
	dir := dockerctlDir(b)
	opts := benchOptions(dir)

	b.Run("Full", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			built, err := codescan.Run(opts)
			require.NoError(b, err)
			require.NotNil(b, built)
		}
	})

	b.Run("Build", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			b.StopTimer()
			ctx, err := scanner.NewScanCtx(opts)
			require.NoError(b, err)
			b.StartTimer()

			builder := spec.NewBuilder(opts.InputSpec, ctx, opts.ScanModels)
			built, err := builder.Build()
			require.NoError(b, err)
			require.NotNil(b, built)
		}
	})
}
