// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"runtime"
	"time"
)

// Cost is what one scan cost to run: how long each half took, and what the process's memory did across it.
//
// Two things it is NOT, both of which the overlay showing it has to say out loud:
//
// It is not codescan's cost. [runtime.ReadMemStats] accounts for the whole process, and while a scan runs the render
// loop, the spinner ticking at ~10Hz and the file watcher all allocate on other goroutines. This is everything the
// process allocated across the scan window - the only figure a process can honestly report about itself.
//
// It is not a peak. Three fences give three readings, and a scan that balloons and drops back between two of them
// leaves no trace here. Sampling would catch that, and would perturb what it measures (ReadMemStats stops the world).
//
// Every field is a difference between fences, computed once in costOf rather than re-derived wherever it is shown.
type Cost struct {
	// Measured distinguishes a real reading from the zero value - the state before the first scan lands, where every
	// difference would otherwise read as a legitimate zero.
	Measured bool

	// ScanFor / RenderFor split the wall clock the same way the memory figures are split.
	ScanFor   time.Duration
	RenderFor time.Duration

	// AllocScan / AllocRender are what was allocated in each half, garbage included: the churn, not the retention.
	AllocScan   uint64
	AllocRender uint64

	// RetainScan / RetainRender are what each half left live behind it. Signed, because a half that triggers a
	// collection can finish with less live than it started with.
	RetainScan   int64
	RetainRender int64

	// LiveBefore / LiveAfter are the live heap at the outer fences, so the retained difference can be read against the
	// size it moved from.
	LiveBefore uint64
	LiveAfter  uint64

	// Objects is the change in live heap objects, which says whether the retained bytes are a few big buffers or a
	// great many small nodes.
	Objects int64

	// GCCycles is how many collections ran across the window, and Sys how much memory the process holds from the OS at
	// the end - effectively a high-water mark, since the Go runtime rarely gives it back.
	GCCycles uint32
	Sys      uint64
}

// Allocated is the whole run's churn.
func (c Cost) Allocated() uint64 { return c.AllocScan + c.AllocRender }

// Retained is what the whole run left live. The two halves sum to it by construction.
func (c Cost) Retained() int64 { return c.RetainScan + c.RetainRender }

// memFence is one reading of the process's memory, taken at one of the three points [Do] measures.
type memFence struct {
	total   uint64 // TotalAlloc: everything ever allocated, garbage included
	live    uint64 // HeapAlloc: what is live at this instant
	objects uint64 // HeapObjects: how many things that live heap is made of
	gc      uint32 // NumGC
	sys     uint64 // Sys: obtained from the OS
}

// fence reads the runtime's account of the process.
//
// Deliberately without a runtime.GC() first, the same call cmd/genspec-wasi makes: forcing a collection would report a
// tidier heap than the one the scan actually ran with, which is the opposite of the question being asked.
func fence() memFence {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return memFence{total: m.TotalAlloc, live: m.HeapAlloc, objects: m.HeapObjects, gc: m.NumGC, sys: m.Sys}
}

// costOf differences the three fences: before the scan, after it, and after the document has been rendered.
//
// A run that failed passes the same fence twice, which reports the failed half honestly (whatever it managed to
// allocate) and the half that never ran as zero.
func costOf(before, scanned, rendered memFence, scanFor, renderFor time.Duration) Cost {
	return Cost{
		Measured:  true,
		ScanFor:   scanFor,
		RenderFor: renderFor,

		AllocScan:   scanned.total - before.total,
		AllocRender: rendered.total - scanned.total,

		RetainScan:   diff(scanned.live, before.live),
		RetainRender: diff(rendered.live, scanned.live),

		LiveBefore: before.live,
		LiveAfter:  rendered.live,

		Objects:  diff(rendered.objects, before.objects),
		GCCycles: rendered.gc - before.gc,
		Sys:      rendered.sys,
	}
}

// diff subtracts two readings as a signed change.
//
// Heap figures are unsigned but their difference is not: a collection between two fences legitimately makes the later
// reading the smaller one.
func diff(after, before uint64) int64 {
	if after >= before {
		//nolint:gosec // a heap difference cannot approach MaxInt64; the guard is on the direction, not the magnitude
		return int64(after - before)
	}

	//nolint:gosec // same
	return -int64(before - after)
}
