// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

// Profiling wrapper for the dockerctl scan benchmark.
//
// `go test -bench` cannot collect a usable -cpuprofile/-memprofile when more
// than one package is selected (the flags are rejected / the files get
// clobbered). This wrapper sidesteps that by driving runtime/pprof directly
// from a single test, so the profile is dumped regardless of how many
// packages the surrounding `go test` invocation matched. It uses only the
// standard library — no extra module dependency (e.g. github.com/pkg/profile).
//
// It profiles the Build phase by default — the regexp→grammar delta — by
// loading the ScanCtx once and replaying Build() in a loop so the CPU
// sampler (100 Hz) gathers enough samples on a ~115 ms workload. Set the
// phase to "full" to instead profile the whole Run (Load dominates there).
//
// Knobs (env vars):
//
//	CODESCAN_BENCH_CPUPROFILE=cpu.out   write a CPU profile
//	CODESCAN_BENCH_MEMPROFILE=mem.out   write a heap profile (after the run)
//	CODESCAN_BENCH_PHASE=build|full     what to profile          (default build)
//	CODESCAN_BENCH_ITERS=30             replay count for sampling (default 30)
//	CODESCAN_BENCH_DIR=/path/to/client  reference corpus location
//
// The test skips unless at least one of CPUPROFILE/MEMPROFILE is set.
//
// Examples:
//
//	# CPU profile of the grammar parser (Build phase), then study it
//	CODESCAN_BENCH_CPUPROFILE=cpu.out \
//	  go test ./internal/integration/ -run TestDockerctlProfile -v
//	go tool pprof -http=: cpu.out
//
//	# heap profile; inspect total bytes allocated
//	CODESCAN_BENCH_MEMPROFILE=mem.out \
//	  go test ./internal/integration/ -run TestDockerctlProfile -v
//	go tool pprof -sample_index=alloc_space -http=: mem.out
//
// Caveat — the CPU profile brackets only the replay loop, so it cleanly
// isolates the chosen phase (Load runs before StartCPUProfile and is
// excluded). The heap profile, by contrast, is process-cumulative: for
// phase=build its alloc_space still includes the one-time Load (go/types)
// allocations, which can dwarf the parser. To study Build allocations, raise
// CODESCAN_BENCH_ITERS so the replayed Build dominates the single Load (e.g.
// 200), or compare the same iters across worktrees and read the delta.

import (
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/builders/spec"
	"github.com/go-openapi/codescan/internal/scanner"
	"github.com/go-openapi/testify/v2/require"
)

const (
	envCPUProfile = "CODESCAN_BENCH_CPUPROFILE"
	envMemProfile = "CODESCAN_BENCH_MEMPROFILE"
	envPhase      = "CODESCAN_BENCH_PHASE"
	envIters      = "CODESCAN_BENCH_ITERS"

	defaultProfileIters = 30
)

// envIterCount reads CODESCAN_BENCH_ITERS, falling back to def.
func envIterCount(tb testing.TB, def int) int {
	tb.Helper()

	raw := os.Getenv(envIters)
	if raw == "" {
		return def
	}

	n, err := strconv.Atoi(raw)
	require.NoErrorf(tb, err, "invalid %s=%q", envIters, raw)
	require.Greaterf(tb, n, 0, "%s must be > 0", envIters)

	return n
}

// createProfile opens path for writing, failing the test on error.
func createProfile(tb testing.TB, path string) *os.File {
	tb.Helper()

	f, err := os.Create(path) //nolint:gosec // path is operator-supplied for local profiling
	require.NoErrorf(tb, err, "create profile %q", path)

	return f
}

// TestDockerctlProfile drives a CPU and/or heap profile of the dockerctl scan
// so the parser hot path can be studied with `go tool pprof`. It is a no-op
// (skipped) unless a profile output path is requested via env.
func TestDockerctlProfile(t *testing.T) {
	cpuPath := os.Getenv(envCPUProfile)
	memPath := os.Getenv(envMemProfile)
	if cpuPath == "" && memPath == "" {
		t.Skipf("no profile requested; set %s and/or %s", envCPUProfile, envMemProfile)
	}

	dir := dockerctlDir(t)
	opts := benchOptions(dir)

	phase := os.Getenv(envPhase)
	if phase == "" {
		phase = "build"
	}

	// workload is one replayable unit of the phase under study.
	var workload func()
	switch phase {
	case "build":
		// Load once; replay Build so the CPU sampler sees the parser, not the
		// go/packages front end.
		ctx, err := scanner.NewScanCtx(opts)
		require.NoError(t, err)
		workload = func() {
			builder := spec.NewBuilder(opts.InputSpec, ctx, opts.ScanModels)
			built, err := builder.Build()
			require.NoError(t, err)
			require.NotNil(t, built)
		}
	case "full":
		workload = func() {
			built, err := codescan.Run(opts)
			require.NoError(t, err)
			require.NotNil(t, built)
		}
	default:
		t.Fatalf("invalid %s=%q (want build|full)", envPhase, phase)
	}

	iters := envIterCount(t, defaultProfileIters)
	t.Logf("profiling phase=%s iters=%d corpus=%s", phase, iters, dir)

	if cpuPath != "" {
		f := createProfile(t, cpuPath)
		defer func() { require.NoError(t, f.Close()) }()
		require.NoError(t, pprof.StartCPUProfile(f))
		defer pprof.StopCPUProfile()
	}

	start := time.Now()
	for range iters {
		workload()
	}
	t.Logf("replayed %d iteration(s) in %s (%s/iter)",
		iters,
		time.Since(start).Round(time.Millisecond),
		(time.Since(start) / time.Duration(iters)).Round(time.Microsecond),
	)

	// Stop CPU profiling before writing the heap profile so the heap dump's
	// own allocations are not attributed to the workload.
	if cpuPath != "" {
		pprof.StopCPUProfile()
		t.Logf("wrote CPU profile: %s", cpuPath)
	}

	if memPath != "" {
		f := createProfile(t, memPath)
		defer func() { require.NoError(t, f.Close()) }()
		runtime.GC() // materialize up-to-date heap statistics
		require.NoError(t, pprof.WriteHeapProfile(f))
		t.Logf("wrote heap profile: %s", memPath)
	}
}
