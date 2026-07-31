// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	yaml "go.yaml.in/yaml/v3"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner"
)

// scanResultMsg carries the outcome of a whole-scope scan: the spec rendered
// as both JSON and YAML, path/definition counts for the header, how long the
// scan took, every grammar.Diagnostic the build emitted (in source order), plus
// any hard error from codescan.Run.
type scanResultMsg struct {
	json       string
	yaml       string
	paths      int
	defs       int
	elapsed    time.Duration
	diags      []grammar.Diagnostic
	provenance []scanner.Provenance
	err        error
}

// runScan runs codescan over the whole scope (the decision-C model: one spec
// for the whole scanned set) and renders it, timing the work. It runs in a
// tea.Cmd goroutine so packages.Load latency never blocks the event loop. cfg
// is taken by value so the goroutine has a stable snapshot even if the model
// mutates its options.
func runScan(cfg codescan.Options) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		res := doScan(cfg)
		res.elapsed = time.Since(start)
		return res
	}
}

// doScan performs the scan and rendering, returning the result without timing
// (runScan stamps the elapsed time around it).
func doScan(cfg codescan.Options) scanResultMsg {
	// OnDiagnostic fires synchronously inside codescan.Run, on this same
	// goroutine, so a plain append is race-free. Diagnostics collected before a
	// hard error are still worth surfacing, so we carry them on every return.
	var diags []grammar.Diagnostic
	cfg.OnDiagnostic = func(d grammar.Diagnostic) {
		diags = append(diags, d)
	}
	// OnProvenance also fires synchronously inside codescan.Run, so a plain
	// append is race-free. This is the source-side half of the cross-ref linker
	// (pointer → source position); the model turns it into a SourceIndex.
	var provs []scanner.Provenance
	cfg.OnProvenance = func(p scanner.Provenance) {
		provs = append(provs, p)
	}

	sw, err := codescan.Run(&cfg)
	if err != nil {
		return scanResultMsg{diags: diags, provenance: provs, err: err}
	}

	jb, err := json.MarshalIndent(sw, "", "  ")
	if err != nil {
		return scanResultMsg{diags: diags, provenance: provs, err: err}
	}

	res := scanResultMsg{json: string(jb), defs: len(sw.Definitions), diags: diags, provenance: provs}
	if sw.Paths != nil {
		res.paths = len(sw.Paths.Paths)
	}
	if yb, yerr := jsonToYAML(jb); yerr == nil {
		res.yaml = string(yb)
	}
	return res
}

// jsonToYAML reserializes ordered JSON bytes as YAML. Map keys come out
// alphabetically (yaml v3's deterministic order), which is good enough for a
// human-readable viewer.
func jsonToYAML(jb []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(jb, &v); err != nil {
		return nil, err
	}
	return yaml.Marshal(v)
}
