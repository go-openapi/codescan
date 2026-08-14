// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"context"
	"fmt"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/google/pprof/profile"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// record fabricates one profile record over a real call stack, so the frames it names resolve like the runtime's own.
//
// skip 2 puts the caller's own frame innermost, which is the site the table is expected to name.
func record(objects, bytesN int64) runtime.MemProfileRecord {
	var r runtime.MemProfileRecord
	r.AllocObjects, r.AllocBytes = objects, bytesN
	runtime.Callers(2, r.Stack0[:])

	return r
}

// countEveryAllocation pins the sampling rate for a test whose figures are exact ones.
//
// Left at the default, every number below would come back multiplied by the sampling correction, which is tested on
// its own in TestUnsampleCorrectsForSampling.
func countEveryAllocation(t *testing.T) {
	t.Helper()

	saved := runtime.MemProfileRate
	t.Cleanup(func() { runtime.MemProfileRate = saved })
	runtime.MemProfileRate = 1
}

func TestSitesDifferenceTheFences(t *testing.T) {
	countEveryAllocation(t)

	before := []runtime.MemProfileRecord{record(10, 1_000)}
	after := []runtime.MemProfileRecord{record(25, 4_000)}
	// Same stack in both, so the site must report the difference and not the total.
	after[0].Stack0 = before[0].Stack0

	got := sites(before, after)

	require.Len(t, got, 1)
	assert.Equal(t, int64(3_000), got[0].Bytes, "what the phase allocated, not what the process had allocated")
	assert.Equal(t, int64(15), got[0].Objects)
	assert.Contains(t, got[0].Name, "TestSitesDifferenceTheFences", "attributed to the site that asked")

	name, ours := frameName(after[0].Stack())
	assert.False(t, ours)
	assert.NotContains(t, name, "runtime.", "the runtime helper every allocation goes through names nobody")
}

// A stack that only appears after the fence is entirely the phase's own.
func TestSitesCountsANewStackWhole(t *testing.T) {
	countEveryAllocation(t)

	got := sites(nil, []runtime.MemProfileRecord{record(4, 512)})

	require.Len(t, got, 1)
	assert.Equal(t, int64(512), got[0].Bytes)
}

// A stack that did not move between the fences allocated nothing during it, and would otherwise fill the table with
// rows reporting zero.
func TestSitesDropsWhatDidNotMove(t *testing.T) {
	same := []runtime.MemProfileRecord{record(10, 1_000)}

	assert.Empty(t, sites(same, same))
}

// The same snapshot on both sides of a fence must difference to nothing, whatever it holds.
//
// This is the shape a failed run takes - it closes twice on the same reading - and a weak identity for a record turns
// it into a table of allocations that never happened, because two colliding stacks difference against each other.
func TestSitesIdentifiesRecordsByTheirWholeStack(t *testing.T) {
	countEveryAllocation(t)

	snapshot := make([]runtime.MemProfileRecord, 0, 512)
	for i := range 512 {
		var r runtime.MemProfileRecord
		r.AllocObjects, r.AllocBytes = int64(i+1), int64(i+1)*128
		// Synthetic stacks that a weaker key (a few frames, a weak mix) would fold together.
		for depth := range 12 {
			r.Stack0[depth] = uintptr(0x400000 + i*8 + depth)
		}
		snapshot = append(snapshot, r)
	}

	assert.Empty(t, sites(snapshot, snapshot), "nothing happened between a fence and itself")
}

// Observing a run costs allocations of its own - stopping a phase's CPU profile flushes it through a compressor, and
// starting the next allocates its buffers - and those land between two fences like anything else. Reported, they would
// crowd out the short phase's real sites with the observer's own work.
func TestSitesExcludesTheProfilersOwnWork(t *testing.T) {
	countEveryAllocation(t)

	var ours runtime.MemProfileRecord
	// pprof.Do puts a genuine runtime/pprof frame on the stack, which is what marks a record as the profiler's.
	pprof.Do(context.Background(), pprof.Labels("k", "v"), func(context.Context) {
		ours = record(1, 8*1024)
	})
	mine := record(1, 8*1024)

	assert.Empty(t, sites(nil, []runtime.MemProfileRecord{ours}), "the profiler's own allocations are not the run's")
	assert.Len(t, sites(nil, []runtime.MemProfileRecord{mine}), 1, "and everything else still counts")
}

func TestRankOrdersAndCaps(t *testing.T) {
	byFunc := make(map[string]*Site)
	for i := range topN + 4 {
		name := fmt.Sprintf("pkg.Func%02d", i)
		byFunc[name] = &Site{Name: name, Bytes: int64(i+1) * 1_000, Objects: int64(i)}
	}

	got := rank(byFunc)

	require.Len(t, got, topN, "the tail is on disk, not on the card")
	assert.Equal(t, "pkg.Func11", got[0].Name, "biggest first")
	for i := 1; i < len(got); i++ {
		assert.Greater(t, got[i-1].Bytes, got[i].Bytes)
	}
}

// The heap profiler samples one allocation per MemProfileRate bytes, so a record stands for more than it saw. Without
// the correction the table would disagree with the figures above it by orders of magnitude.
func TestUnsampleCorrectsForSampling(t *testing.T) {
	saved := runtime.MemProfileRate
	t.Cleanup(func() { runtime.MemProfileRate = saved })

	runtime.MemProfileRate = 1
	objects, bytesN := unsample(3, 300)
	assert.Equal(t, int64(3), objects, "every allocation was recorded: nothing to estimate")
	assert.Equal(t, int64(300), bytesN)

	runtime.MemProfileRate = 512 * 1024
	smallObjects, smallBytes := unsample(1, 64) // a 64-byte object is sampled once in ~8000
	assert.Greater(t, smallBytes, int64(64*1_000), "small allocations are scaled up hard")
	assert.Greater(t, smallObjects, int64(1_000))

	bigObjects, bigBytes := unsample(1, 8*1024*1024) // far above the rate: seen almost every time
	assert.Less(t, bigBytes, int64(9*1024*1024), "a large allocation is barely scaled")
	assert.Equal(t, int64(1), bigObjects)
}

func TestCPUFuncsRanksByTime(t *testing.T) {
	prof := &profile.Profile{
		SampleType: []*profile.ValueType{{Type: "samples", Unit: "count"}, {Type: "cpu", Unit: "nanoseconds"}},
		Sample: []*profile.Sample{
			{Value: []int64{1, int64(300 * time.Millisecond)}, Location: []*profile.Location{loc("scan")}},
			{Value: []int64{1, int64(200 * time.Millisecond)}, Location: []*profile.Location{loc("scan")}},
			{Value: []int64{1, int64(500 * time.Millisecond)}, Location: []*profile.Location{loc("render")}},
		},
	}

	got, total, samples := cpuFuncs(prof)

	require.Len(t, got, 2)
	assert.Equal(t, time.Second, total)
	assert.Equal(t, int64(3), samples, "how much the ranking is worth")
	assert.Equal(t, "render", got[0].Name)
	assert.InDelta(t, 0.5, got[0].Share, 0.001)
	assert.Equal(t, "scan", got[1].Name, "the two scan samples fold into one row")
	assert.Equal(t, 500*time.Millisecond, got[1].Flat)
}

func TestCPUFuncsOnNothingSampled(t *testing.T) {
	got, total, samples := cpuFuncs(&profile.Profile{
		SampleType: []*profile.ValueType{{Type: "samples"}, {Type: "cpu"}},
	})

	assert.Empty(t, got)
	assert.Zero(t, total)
	assert.Zero(t, samples)
	assert.Empty(t, func() []Func { g, _, _ := cpuFuncs(nil); return g }(),
		"a profile that never parsed is not a panic")
}

// The disabled profiler is the common case: it must cost nothing and produce no report.
func TestProfilerDisabled(t *testing.T) {
	p := newProfiler(Profiling{})

	p.scanDone()

	assert.Nil(t, p.done())
}

func TestProfilerCapturesARun(t *testing.T) {
	dir := t.TempDir()

	p := newProfiler(Profiling{Enabled: true, Dir: dir})
	sink := make([][]byte, 0, 128)
	for range 128 {
		sink = append(sink, make([]byte, 8*1024))
	}
	p.scanDone()
	sink = append(sink, make([]byte, 64*1024))
	report := p.done()
	runtime.KeepAlive(sink)

	require.NotNil(t, report)
	assert.Equal(t, dir, report.Dir)
	assert.Len(t, report.MemPaths, MemSnapshots, "one heap snapshot per fence, for go tool pprof -base")
	assert.NotEmpty(t, report.Scan.Alloc, "the allocations between the first two fences are attributed")
	assert.FileExists(t, report.Scan.CPUPath)
	assert.FileExists(t, report.Render.CPUPath, "the rendering phase is bracketed too")
}

// The CPU profiler is a process-wide singleton, so a second profiled run cannot have it - and must not take it from
// the first, nor fail the scan it belongs to.
func TestProfilerYieldsTheCPUSingleton(t *testing.T) {
	first := newProfiler(Profiling{Enabled: true, Dir: t.TempDir()})
	second := newProfiler(Profiling{Enabled: true, Dir: t.TempDir()})

	require.NotNil(t, second.done())
	assert.Contains(t, second.report.Notes[0], "another profiled scan is in flight")
	assert.Empty(t, second.report.Scan.CPUPath)

	report := first.done()
	require.NotNil(t, report)
	assert.NotEmpty(t, report.Scan.CPUPath, "the run that got there first keeps its profile")
}

func loc(fn string) *profile.Location {
	return &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: fn}}}}
}
