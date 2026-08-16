// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	yaml "go.yaml.in/yaml/v3"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
)

// ResultMsg carries the outcome of a whole-scope scan.
//
// The spec rendered as JSON, path and definition counts for the header, how long the scan took, what it
// cost to run, every diagnostic the build emitted in source order, and any hard error from codescan.Run.
type ResultMsg struct {
	JSON       string
	Paths      int
	Defs       int
	Elapsed    time.Duration
	Cost       Cost
	Profile    *ProfileReport // nil unless the run was profiled
	Diags      []grammar.Diagnostic
	Provenance []scanner.Provenance
	Err        error
}

// Run scans the whole scope, renders the spec, and times the work.
//
// Whole scope means one spec for the entire scanned set, rather than one per package.
//
// It runs in a tea.Cmd goroutine so packages.Load latency never blocks the event loop, which is why cfg is snapshotted
// HERE rather than inside the command: the caller hands over the model's live configuration, and the options overlay
// writes to it from the event loop while the scan reads it. Taking the copy on the caller's goroutine, before the
// command exists, is what keeps the two apart - a copy taken inside the command would be the race it is meant to avoid.
func Run(cfg *codescan.Options, prof *Profiling) tea.Cmd {
	snapshot := *cfg

	return func() tea.Msg {
		start := time.Now()
		res := Do(&snapshot, prof)
		res.Elapsed = time.Since(start)
		return res
	}
}

// Do performs the scan and rendering, returning the result without timing (runScan stamps the elapsed time around it).
//
// It fences the work three times - before the scan, after it, and once the document has been rendered - so the result
// carries what the run cost as well as what it produced. The inner split is what makes the reading actionable: it
// separates what codescan spent from what serializing the document spent on top.
//
// A profiled run brackets the same two phases with the profiler as well, so what the sampler says and what the fences
// say describe the same halves of the same work.
//
// It is exposed by this package to allow for e2e tests.
func Do(cfg *codescan.Options, prof *Profiling) ResultMsg {
	// Worked on by value, because the two callbacks below are installed on it: the caller's Options is the model's live
	// configuration, and a run that wrote its own collectors into it would both mutate what the options overlay shows
	// and hand a second, overlapping run the first one's slices to append into.
	local := *cfg
	cfg = &local

	// OnDiagnostic fires synchronously inside codescan.Run, on this same goroutine, so a plain append is race-free.
	//
	// Diagnostics collected before a hard error are still worth surfacing, so we carry them on every return.
	var diags []grammar.Diagnostic
	cfg.OnDiagnostic = func(d grammar.Diagnostic) {
		diags = append(diags, d)
	}
	// OnProvenance also fires synchronously inside codescan.Run, so a plain append is race-free.
	//
	// This is the source-side half of the cross-ref linker (pointer → source position); the model turns it into a
	// SourceIndex.
	var provs []scanner.Provenance
	cfg.OnProvenance = func(p scanner.Provenance) {
		provs = append(provs, p)
	}

	// The profiler brackets each phase from OUTSIDE the fences: its own snapshots collect and allocate, and a fence
	// that read after one of them would be reporting the profiler as well as the run.
	p := newProfiler(prof)
	before := fence()

	scanStart := time.Now()
	sw, err := codescan.Run(cfg)
	scanFor := time.Since(scanStart)
	scanned := fence()

	if err != nil {
		return ResultMsg{
			Diags: diags, Provenance: provs, Err: err,
			Cost: costOf(before, scanned, scanned, scanFor, 0), Profile: p.done(),
		}
	}
	p.scanDone()

	renderStart := time.Now()

	jb, err := json.MarshalIndent(sw, "", "  ")
	if err != nil {
		return ResultMsg{
			Diags: diags, Provenance: provs, Err: err,
			Cost: costOf(before, scanned, fence(), scanFor, time.Since(renderStart)), Profile: p.done(),
		}
	}

	res := ResultMsg{JSON: string(jb), Defs: len(sw.Definitions), Diags: diags, Provenance: provs}
	if sw.Paths != nil {
		res.Paths = len(sw.Paths.Paths)
	}
	renderFor := time.Since(renderStart)

	res.Cost = costOf(before, scanned, fence(), scanFor, renderFor)
	res.Profile = p.done()

	return res
}

// RenderYAML reserializes a rendered JSON document as YAML.
//
// Called when the YAML view is first asked for rather than alongside the JSON, because it costs more than the document
// it is made from - a full reparse into map[string]any, then the emitter - and most sessions never open it. On a large
// specification that was the larger half of every rescan, spent on a view nobody had looked at.
//
// Map keys come out alphabetically (yaml v3's deterministic order), which is good enough for a human-readable viewer.
func RenderYAML(jsonBody string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(jsonBody), &v); err != nil {
		return "", err
	}

	yb, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(yb), nil
}
