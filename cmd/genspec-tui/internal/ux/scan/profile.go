// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/pprof/profile"
)

// Profiling is what a run was asked to capture, and where the artifacts go.
//
// Off by default, and switched at launch rather than from the options overlay: [runtime.MemProfileRate] is meant to be
// set once and held constant for the life of the process, so a mid-session toggle would make consecutive readings
// incomparable - and a rate low enough to be exact is expensive enough to change what it measures.
type Profiling struct {
	Enabled bool

	// Dir is where the .pprof artifacts are written, so the same run can be opened in go tool pprof.
	Dir string
}

// ProfileReport is what a profiled run observed, already reduced to what a reader is shown.
//
// This is the account that the [Cost] figures cannot give. Cost is a process-wide scalar: while a scan runs, the
// redraw loop and the file watcher allocate on other goroutines, and no arithmetic separates them out afterwards. A
// profile is per stack, so the same noise arrives named - a row to discount rather than a confound.
type ProfileReport struct {
	Dir string

	// MemPaths are the heap snapshots, in fence order, for opening the same run in the real tool.
	MemPaths []string

	// Scan and Render are the two phases, measured the same way. Rendering is the shorter by far - often too short
	// for the CPU sampler to catch anything - but that is a fact about a particular run, and one the reader is told
	// rather than one this design decides for them.
	Scan   PhaseProfile
	Render PhaseProfile

	// Exact says the allocation figures are counted rather than estimated (MemProfileRate 1).
	Exact bool

	// Notes carries whatever went wrong. A profile is an observation of the run, never a precondition for it, so a
	// failure here is reported and nothing more.
	Notes []string
}

// PhaseProfile is what the profiler saw during one phase of a run.
type PhaseProfile struct {
	// Label names the phase in the report, and in the note when it could not be sampled.
	Label string

	// CPUPath is the profile written for this phase, empty when it could not be captured.
	CPUPath string

	// CPUTotal is the CPU time the samples account for, which is not the wall clock: it counts every goroutine that
	// ran, and counts nothing while they were all blocked.
	CPUTotal time.Duration
	// CPUSamples is how many samples that time is made of. The sampler runs at 100 Hz, so a phase shorter than its
	// interval yields none at all, and a handful ranks nothing: the count is what says how much to believe the table.
	CPUSamples int64
	CPU        []Func

	// Alloc attributes what the phase allocated to the site that asked for it.
	Alloc []Site
}

// Sampled reports whether the CPU sampler caught anything in this phase.
func (p PhaseProfile) Sampled() bool { return len(p.CPU) > 0 }

// Func is one function's flat share of the CPU samples.
type Func struct {
	Name  string
	Flat  time.Duration
	Share float64
}

// Site is one allocation site: the function that asked for the memory, and how much of it.
type Site struct {
	Name    string
	Bytes   int64
	Objects int64
}

// MemSnapshots is how many heap snapshots a complete run leaves behind: before, after scanning, after rendering.
//
// Exported because a reader of the report has to know that the first two bracket the scan and the last two the
// render - which is what makes `go tool pprof -base` reproduce either phase.
const MemSnapshots = 3

// topN is how many rows of each table the card shows. Past this the tail is noise a reader would not act on, and the
// artifacts are on disk for anyone who wants all of it.
const topN = 8

// cpuProfiler serializes profiled runs.
//
// The CPU profiler is a process-wide singleton, and two scans can genuinely be in flight at once - changing an option
// while one is running starts another. Rather than have the second fail (or, worse, stop the first one's profile), it
// runs unprofiled and says so.
//
//nolint:gochecknoglobals // the resource being guarded is itself process-global
var cpuProfiler sync.Mutex

// profiler captures a run. The zero value is the disabled profiler, whose methods all do nothing.
type profiler struct {
	report *ProfileReport

	cpuFile  *os.File
	cpuPhase *PhaseProfile // the phase the open profile belongs to
	held     bool          // whether the CPU profiler singleton is ours

	before  []runtime.MemProfileRecord
	scanned []runtime.MemProfileRecord
}

// newProfiler starts a profiled run, or returns the disabled profiler when one was not asked for (or cannot be had).
func newProfiler(p Profiling) *profiler {
	if !p.Enabled {
		return &profiler{}
	}

	prof := &profiler{report: &ProfileReport{Dir: p.Dir, Exact: runtime.MemProfileRate == 1}}

	if err := os.MkdirAll(p.Dir, 0o750); err != nil {
		prof.notef("no profile directory: %v", err)

		return prof
	}

	prof.report.Scan.Label, prof.report.Render.Label = "scanning", "rendering"

	// Held for the WHOLE run, not per phase: the two phases are profiled one after the other, and letting go between
	// them would let another scan take the sampler halfway through this one's measurement.
	if !cpuProfiler.TryLock() {
		prof.notef("another profiled scan is in flight; this one is not CPU profiled")
	} else {
		prof.held = true
		prof.startCPU(&prof.report.Scan, "cpu-scan.pprof")
	}

	prof.before = prof.snapshot(filepath.Join(p.Dir, "mem-before.pprof"))

	return prof
}

// startCPU begins sampling into phase's own profile, giving up quietly (with a note) if it cannot.
func (p *profiler) startCPU(phase *PhaseProfile, name string) {
	if !p.held {
		return // the sampler belongs to another run
	}

	path := filepath.Join(p.report.Dir, name)
	f, err := os.Create(path)
	if err != nil {
		p.notef("no CPU profile for %s: %v", phase.Label, err)

		return
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		p.notef("no CPU profile for %s: %v", phase.Label, err)
		_ = f.Close()

		return
	}
	p.cpuFile = f
	p.cpuPhase = phase
	phase.CPUPath = path
}

// scanDone closes the scanning phase and opens the rendering one.
//
// Each phase gets its own CPU profile rather than one profile split by reading the stacks: the phases are sequential,
// so the bracket is exact, and rendering - the shorter by far - is then measured the same way as the scan rather than
// being assumed too small to look at.
func (p *profiler) scanDone() {
	if p.report == nil {
		return
	}

	// The sampler stops before the heap snapshot, so the snapshot's own work is not attributed to the scan, and the
	// next phase's sampler starts after it for the same reason.
	p.stopCPU()
	p.scanned = p.snapshot(filepath.Join(p.report.Dir, "mem-after-scan.pprof"))
	p.startCPU(&p.report.Render, "cpu-render.pprof")
}

// done closes the run and reduces everything captured into the report.
func (p *profiler) done() *ProfileReport {
	if p.report == nil {
		return nil
	}

	p.stopCPU() // no-op unless the run failed before scanDone
	rendered := p.snapshot(filepath.Join(p.report.Dir, "mem-after-render.pprof"))
	if p.scanned == nil {
		// The run never reached rendering, so everything it allocated belongs to the scan - the same convention the
		// cost fences follow when they close on the phase that failed.
		p.scanned = rendered
	}

	p.report.Scan.Alloc = sites(p.before, p.scanned)
	p.report.Render.Alloc = sites(p.scanned, rendered)
	p.readCPU(&p.report.Scan)
	p.readCPU(&p.report.Render)
	p.release()

	return p.report
}

// stopCPU ends sampling for the phase in flight. Safe to call twice.
func (p *profiler) stopCPU() {
	if p.cpuFile == nil {
		return
	}

	pprof.StopCPUProfile()
	if err := p.cpuFile.Close(); err != nil {
		p.notef("CPU profile for %s not written: %v", p.cpuPhase.Label, err)
		p.cpuPhase.CPUPath = ""
	}
	p.cpuFile = nil
	p.cpuPhase = nil
}

// release hands the sampler back, once the whole run is measured.
func (p *profiler) release() {
	if p.held {
		cpuProfiler.Unlock()
		p.held = false
	}
}

// snapshot records the allocation profile at one fence, and writes it out for go tool pprof.
//
// It forces a collection first - the exact opposite of the rule the unprofiled fences follow, and right here for the
// same reason it is wrong there. [runtime.MemProfile] is only current as of the last collection, so a phase measured
// without one loses its tail; and a profiled run is asking to be measured accurately, having already accepted that it
// is not measuring itself.
func (p *profiler) snapshot(path string) []runtime.MemProfileRecord {
	runtime.GC()

	records := memRecords()

	f, err := os.Create(path)
	if err != nil {
		p.notef("no heap snapshot: %v", err)

		return records
	}
	defer func() { _ = f.Close() }()

	if err := pprof.WriteHeapProfile(f); err != nil {
		p.notef("no heap snapshot: %v", err)

		return records
	}
	p.report.MemPaths = append(p.report.MemPaths, path)

	return records
}

// readCPU parses back the profile we just wrote and reduces it to the functions that spent the time.
//
// Parsing our own output rather than sampling in parallel: the file is the artifact anyway, and one reduction of one
// source cannot disagree with itself.
func (p *profiler) readCPU(phase *PhaseProfile) {
	if phase.CPUPath == "" {
		return
	}

	raw, err := os.ReadFile(phase.CPUPath)
	if err != nil {
		p.notef("CPU profile for %s unreadable: %v", phase.Label, err)

		return
	}

	prof, err := profile.Parse(bytes.NewReader(raw))
	if err != nil {
		p.notef("CPU profile for %s unparseable: %v", phase.Label, err)

		return
	}

	phase.CPU, phase.CPUTotal, phase.CPUSamples = cpuFuncs(prof)
}

func (p *profiler) notef(format string, args ...any) {
	if p.report == nil {
		return
	}
	p.report.Notes = append(p.report.Notes, fmt.Sprintf(format, args...))
}

// memRecords reads the whole allocation profile.
//
// The retry is the documented dance: the count can grow between asking how many records there are and reading them.
func memRecords() []runtime.MemProfileRecord {
	// inuseZero true: a phase's allocations count whether or not they survived it, which is the question being asked.
	n, _ := runtime.MemProfile(nil, true)
	for range 3 {
		records := make([]runtime.MemProfileRecord, n+64)
		var ok bool
		n, ok = runtime.MemProfile(records, true)
		if ok {
			return records[:n]
		}
	}

	return nil
}

// sites attributes what was allocated between two fences to the function that asked for it.
//
// Flat attribution, to the innermost frame: "who called make" is the question a reader acts on, and a cumulative tree
// in a terminal table would answer a different one badly.
func sites(before, after []runtime.MemProfileRecord) []Site {
	base := make(map[uint64]runtime.MemProfileRecord, len(before))
	for _, r := range before {
		base[key(r)] = r
	}

	byFunc := make(map[string]*Site)
	for _, r := range after {
		bytesN, objects := r.AllocBytes, r.AllocObjects
		if b, seen := base[key(r)]; seen {
			bytesN -= b.AllocBytes
			objects -= b.AllocObjects
		}
		if objects <= 0 || bytesN <= 0 {
			continue
		}
		objects, bytesN = unsample(objects, bytesN)

		name, ours := frameName(r.Stack())
		if ours {
			// The profiler's own allocations, excluded rather than reported: stopping one phase's CPU profile flushes it
			// through a compressor, and starting the next allocates its buffers, all of it between two fences. Left in,
			// the rendering table - the short phase, where it is a visible share - would be reporting the observer.
			continue
		}
		site, seen := byFunc[name]
		if !seen {
			site = &Site{Name: name}
			byFunc[name] = site
		}
		site.Bytes += bytesN
		site.Objects += objects
	}

	return rank(byFunc)
}

// rank orders the sites by what they allocated and keeps the head of the table.
//
// The tail is not lost, only unshown: every record is in the heap snapshots on disk, where a reader who wants the long
// version has the real tool to read it with.
func rank(byFunc map[string]*Site) []Site {
	out := make([]Site, 0, len(byFunc))
	for _, s := range byFunc {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}

		return out[i].Name < out[j].Name
	})

	return out[:min(len(out), topN)]
}

// key identifies a record across two snapshots, by hashing its whole call stack.
//
// The whole stack, and with a real mixer, because a collision here does not merely misattribute a row: two records
// sharing a key make one of them difference against the other's counters, which manufactures allocation that never
// happened - including in the phase where a fence was closed twice on the same snapshot and the answer must be zero.
func key(r runtime.MemProfileRecord) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)

	h := uint64(offset64)
	for _, pc := range r.Stack() {
		h ^= uint64(pc)
		h *= prime64
	}

	return h
}

// unsample corrects a sampled record back to what it estimates was really allocated.
//
// The heap profiler samples one allocation per MemProfileRate bytes, so a record stands for more than it recorded, and
// by how much depends on the size of the objects it saw: at the default rate a stack allocating 8-byte nodes is
// sampled far more rarely per byte than one allocating megabyte buffers. This is the correction pprof itself applies
// (scaleHeapSample) - without it the table would under-report by orders of magnitude and disagree with the card above
// it.
func unsample(objects, bytesN int64) (int64, int64) {
	rate := int64(runtime.MemProfileRate)
	if rate <= 1 || objects == 0 || bytesN == 0 {
		return objects, bytesN // every allocation was recorded: the counts are the truth
	}

	avg := float64(bytesN) / float64(objects)
	scale := 1 / (1 - math.Exp(-avg/float64(rate)))

	return int64(float64(objects) * scale), int64(float64(bytesN) * scale)
}

// frameName names the allocation site: the innermost frame outside the runtime.
//
// Every allocation passes through the same handful of runtime helpers - newobject, growslice, convTstring - so taking
// the innermost frame verbatim labels much of the table with the fact that memory was allocated, which is what the
// table is already about. The caller is the answer to "who asked for it".
//
// A stack that is runtime frames all the way down keeps its innermost one: it is a real answer, just a rarer one.
//
// ours reports that the stack is the profiler's own work rather than the run's.
func frameName(stack []uintptr) (name string, ours bool) {
	if len(stack) == 0 {
		return "(unknown)", false
	}

	frames := runtime.CallersFrames(stack)
	innermost, caller := "", ""
	for {
		f, more := frames.Next()
		if f.Function != "" {
			if strings.HasPrefix(f.Function, "runtime/pprof.") {
				return "", true
			}
			if caller == "" && !strings.HasPrefix(f.Function, "runtime.") && !strings.HasPrefix(f.Function, "internal/") {
				caller = f.Function
			}
			if innermost == "" {
				innermost = f.Function
			}
		}
		if !more {
			break
		}
	}

	// The walk continues past the caller rather than returning at it, because a pprof frame further out is what marks
	// the whole stack as the profiler's own.
	switch {
	case caller != "":
		return caller, false
	case innermost != "":
		return innermost, false
	default:
		return "(unknown)", false
	}
}

// cpuFuncs reduces a parsed CPU profile to the functions that spent the time, flat, biggest first.
//
// The second sample value is the one to read: a CPU profile carries samples/count and cpu/nanoseconds, and time is
// what a reader is comparing against the wall clock on the card.
func cpuFuncs(prof *profile.Profile) ([]Func, time.Duration, int64) {
	const (
		count = 0 // index of samples/count in a CPU profile's sample values
		nanos = 1 // index of cpu/nanoseconds
	)

	if prof == nil || len(prof.SampleType) <= nanos {
		return nil, 0, 0
	}

	var total, samples int64
	byFunc := make(map[string]int64)
	for _, s := range prof.Sample {
		if len(s.Value) <= nanos || len(s.Location) == 0 {
			continue
		}
		v := s.Value[nanos]
		total += v
		samples += s.Value[count]
		byFunc[leafName(s.Location[0])] += v
	}
	if total == 0 {
		return nil, 0, 0
	}

	out := make([]Func, 0, len(byFunc))
	for name, v := range byFunc {
		out = append(out, Func{Name: name, Flat: time.Duration(v), Share: float64(v) / float64(total)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Flat != out[j].Flat {
			return out[i].Flat > out[j].Flat
		}

		return out[i].Name < out[j].Name
	})

	return out[:min(len(out), topN)], time.Duration(total), samples
}

// leafName names the function a sample was taken in.
func leafName(loc *profile.Location) string {
	if loc == nil || len(loc.Line) == 0 || loc.Line[0].Function == nil {
		return "(unknown)"
	}

	return loc.Line[0].Function.Name
}
