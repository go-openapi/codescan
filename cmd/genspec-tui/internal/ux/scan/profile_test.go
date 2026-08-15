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
	assert.Equal(t, 500*time.Millisecond, got[1].Spent)
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
	p := newProfiler(&Profiling{})

	p.scanDone()

	assert.Nil(t, p.done())
}

func TestProfilerCapturesARun(t *testing.T) {
	// At the default rate the heap profiler samples one allocation per 512 KiB, so a megabyte of test allocations is a
	// Poisson draw with a mean of two: it comes back empty about one run in seven, whatever the machine or the
	// toolchain. Counting every allocation makes the assertion below about attribution rather than about luck, and the
	// runtime honours rate 1 from the next allocation - there is no threshold left over from the old rate.
	countEveryAllocation(t)

	dir := t.TempDir()

	p := newProfiler(&Profiling{Enabled: true, Dir: dir})
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
	first := newProfiler(&Profiling{Enabled: true, Dir: t.TempDir()})
	second := newProfiler(&Profiling{Enabled: true, Dir: t.TempDir()})

	require.NotNil(t, second.done())
	assert.Contains(t, second.report.Notes[0], "another profiled scan is in flight")
	assert.Empty(t, second.report.Scan.CPUPath)

	report := first.done()
	require.NotNil(t, report)
	assert.NotEmpty(t, report.Scan.CPUPath, "the run that got there first keeps its profile")
}

// A CPU sample is charged where a reader can act on it: the call of ours that led there, and what it called.
func TestChargeFindsTheBoundaryOutOfOurCode(t *testing.T) {
	t.Parallel()

	// Leaf first, as a profile records a stack.
	got, profilers := charge(stack(
		"runtime.mallocgc",
		"go/types.(*Checker).recordTypeAndValue",
		"go/types.(*Checker).checkFiles",
		"github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).Load",
		"github.com/go-openapi/codescan.Run",
	))

	assert.False(t, profilers)
	assert.Equal(t, "github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).Load", got.name,
		"the deepest call of ours, not the outermost: it is the more specific of the two")
	assert.Equal(t, "go/types.(*Checker).checkFiles", got.callee,
		"what it handed the work to, not what was executing four frames further down")
	assert.False(t, got.machinery)
}

func TestChargeOnAStackThatNeverLeavesOurCode(t *testing.T) {
	t.Parallel()

	got, _ := charge(stack(
		"github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).FindDecl",
		"github.com/go-openapi/codescan/internal/builders/spec.(*Builder).Build",
	))

	assert.Equal(t, "github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).FindDecl", got.name)
	assert.Empty(t, got.callee, "there is no boundary to name: the time was spent in our own code")
}

// Garbage collection and the allocator run on goroutines of the runtime's own, with nothing of ours above them.
func TestChargeCollectsTheRuntimeIntoOneRow(t *testing.T) {
	t.Parallel()

	got, _ := charge(stack("runtime.scanobject", "runtime.gcDrain", "runtime.gcBgMarkWorker"))

	assert.True(t, got.machinery)
	assert.Empty(t, got.name, "the row is what it is, not a function to look up")
}

// Work on a goroutine we did not start still happened during the phase, and naming what it is doing says more than
// folding it in with the collector.
//
// The goroutine wrapper is not that name: everything the loader fans out roots in the same errgroup frame, so it
// separates nothing. Neither is the closure the wrapper was handed - two entrances into one piece of work.
func TestChargeNamesWhatAnotherGoroutineIsDoing(t *testing.T) {
	t.Parallel()

	got, _ := charge(stack(
		"internal/poll.(*FD).Read",
		"os.(*File).Read",
		"golang.org/x/tools/go/packages.(*loader).parseFiles.func1",
		"golang.org/x/sync/errgroup.(*Group).Go.func1",
	))

	assert.Equal(t, "golang.org/x/tools/go/packages.(*loader).parseFiles", got.name)
	assert.False(t, got.machinery)
}

// A closure of ours is folded the same way, wherever it runs: the callback the scanner is handed is the scanner's
// work, and a table that splits it across func1 and func2 makes the reader add them up.
func TestChargeFoldsAClosureIntoWhatItWasWrittenIn(t *testing.T) {
	t.Parallel()

	got, _ := charge(stack(
		"go/types.(*Checker).checkFiles",
		"github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).Load.func2.1",
	))

	assert.Equal(t, "github.com/go-openapi/codescan/internal/scanner.(*ScanCtx).Load", got.name)
	assert.Equal(t, "go/types.(*Checker).checkFiles", got.callee)
}

// The sampler catches the profiler writing the profile. The card says the tables exclude its work, so they must.
func TestCPUFuncsExcludesTheProfilersOwnWork(t *testing.T) {
	t.Parallel()

	ours := stack("go/types.(*Checker).checkFiles", "github.com/go-openapi/codescan.Run")
	profilers := stack("compress/flate.(*compressor).deflate", "runtime/pprof.profileWriter")

	got, total, samples := cpuFuncs(&profile.Profile{
		SampleType: []*profile.ValueType{{Type: "samples", Unit: "count"}, {Type: "cpu", Unit: "nanoseconds"}},
		Sample: []*profile.Sample{
			{Value: []int64{1, int64(300 * time.Millisecond)}, Location: ours},
			{Value: []int64{1, int64(100 * time.Millisecond)}, Location: profilers},
		},
	})

	require.Len(t, got, 1)
	assert.Equal(t, 300*time.Millisecond, total, "the profiler's own time is out of the total too")
	assert.Equal(t, int64(1), samples)
	assert.InDelta(t, 1.0, got[0].Share, 0.001, "so the shares describe the run, not the observation")
}

// The symbol is what is walked, never the path in front of it: a module path carries dots of its own.
func TestEnclosingFoldsOnlyTheSymbol(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ in, want string }{
		{"go.yaml.in/yaml/v3.(*parser).parse", "go.yaml.in/yaml/v3.(*parser).parse"},
		{"go.yaml.in/yaml/v3.(*encoder).marshal.func1", "go.yaml.in/yaml/v3.(*encoder).marshal"},
		{"golang.org/x/tools/go/packages.(*loader).refine.func2.1", "golang.org/x/tools/go/packages.(*loader).refine"},
		{"go/types.(*Checker).checkFiles", "go/types.(*Checker).checkFiles"},
		{"main.func1", "main.func1"}, // nothing in front of it to fold into
		{"", ""},
	} {
		assert.Equal(t, c.want, enclosing(c.in))
	}
}

func loc(fn string) *profile.Location {
	return &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: fn}}}}
}

// stack builds a sample's locations, innermost first, as a profile records them.
func stack(frames ...string) []*profile.Location {
	locs := make([]*profile.Location, 0, len(frames))
	for _, f := range frames {
		locs = append(locs, loc(f))
	}

	return locs
}
