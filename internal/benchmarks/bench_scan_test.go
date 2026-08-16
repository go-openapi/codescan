// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package benchmarks_test

// What one whole scan costs, split into the two phases that answer different questions.
//
//   - Load — scanner.NewScanCtx: resolving and type-checking the package graph. This is where a
//     loader configuration shows up, and it is the bulk of a scan.
//   - Build — spec.NewBuilder(...).Build(): reading annotations and emitting Swagger. This is where
//     the parser shows up, and it is where a corpus with routes differs from one without.
//
// The split is the point. An end-to-end number moves for either reason and says which only by
// accident, and the two phases have moved in opposite directions over this project's history.
//
// Run:
//
//	go test ./internal/benchmarks/ -run TestScanPhases -v
//	go test ./internal/benchmarks/ -run x -bench BenchmarkScan -benchtime=1x -benchmem

import (
	"runtime"
	"testing"
	"time"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/builders/spec"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/require"
)

// benchOptions is the configuration every phase measurement shares: scan everything, models
// included, and leave the loader at its default so the phase split is not about a loader.
func benchOptions(c benchCorpus) *codescan.Options {
	return &codescan.Options{
		WorkDir:    c.Dir,
		Packages:   []string{c.Pattern},
		ScanModels: true,
		// The corpora unpack inside this repository, whose workspace does not list them — see
		// benchEnv.
		GOWORK: "off",
	}
}

// phaseStats captures the cost of one measured phase.
type phaseStats struct {
	name    string
	elapsed time.Duration
	allocPB uint64 // bytes allocated during the phase (cumulative TotalAlloc delta)
	allocsN uint64 // number of heap allocations during the phase (Mallocs delta)
}

// measure runs fn once, fencing it with a GC so the heap counters reflect only fn's own
// allocations, and records elapsed time + allocation deltas.
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
		float64(p.allocPB)/bytesPerMB,
		p.allocsN,
	)
}

// TestScanPhases runs a single scan of every corpus and prints the per-phase cost.
//
// One iteration is enough: the ratios reproduce, and the figures worth quoting are the allocation
// ones, which do not move with machine load the way wall clock does.
//
// Gated like every other report here. CI runs `go test work ./...` with -race on six job
// combinations, and a scan that peaks at 800 MB is not something to instrument six times over on a
// shared runner — where the timing would measure the runner anyway.
func TestScanPhases(t *testing.T) {
	requireBenchEnabled(t)

	for _, corpus := range benchCorpora(t) {
		t.Run(corpus.Name, func(t *testing.T) {
			opts := benchOptions(corpus)

			// Load then Build, sharing one ScanCtx so Build sees exactly what Run would feed it.
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

			t.Logf("=== %s scan phases (single iteration, %s) ===", corpus.Name, corpus.Dir)
			loadStats.report(t)
			buildStats.report(t)
			fullStats.report(t)
		})
	}
}

// BenchmarkScan is the same workload exposed to the Go benchmark framework. Run with
// -benchtime=1x -benchmem for a one-shot allocation profile, or with the default settings for a
// stabilized timing.
func BenchmarkScan(b *testing.B) {
	for _, corpus := range benchCorpora(b) {
		opts := benchOptions(corpus)

		b.Run(corpus.Name+"/Full", func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				built, err := codescan.Run(opts)
				require.NoError(b, err)
				require.NotNil(b, built)
			}
		})

		b.Run(corpus.Name+"/Build", func(b *testing.B) {
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
}
