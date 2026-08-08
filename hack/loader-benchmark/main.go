// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Command loader-benchmark measures one codescan scan and reports what it cost.
//
// It exists because the in-repo benchmarks cannot answer "how much did we improve": they live inside
// the module, so they do not exist in any earlier release. This one talks only to the public
// [codescan.Run] API, which every version has, so the same source compiles against a released tag and
// against the working tree — and run.sh builds it both ways to put the two side by side.
//
// One scan per process, deliberately. Peak RSS is a process-wide high-water mark, so a second scan in
// the same process would report the first one's peak plus its own garbage.
//
// Usage is through run.sh. Running this directly measures a single cell:
//
//	loader-benchmark -dir /path/to/corpus -corpus mycorpus -label master
//	loader-benchmark -summarize results/*.jsonl
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-openapi/codescan"
)

const (
	bytesPerMB   = 1 << 20
	kbPerMB      = 1024
	percentScale = 100
)

// The two operator mistakes worth naming: a corpus that is not there, and a summary of nothing.
var (
	errNoCorpus       = errors.New("no corpus directory given")
	errNoMeasurements = errors.New("no measurements found")
)

// measurement is one cell of the matrix: one version, one configuration, one corpus, one cache state.
//
// The shape fields are not decoration. A configuration that got faster by scanning less is not an
// improvement, and comparing definition and path counts across a row is what catches it.
type measurement struct {
	Label        string  `json:"label"`
	Corpus       string  `json:"corpus"`
	Cache        string  `json:"cache"` // "warm" or "cold"
	WallSeconds  float64 `json:"wallSeconds"`
	RetainedMB   float64 `json:"retainedMb"`
	TotalAllocMB float64 `json:"totalAllocMb"`
	PeakRSSMB    float64 `json:"peakRssMb"`
	Definitions  int     `json:"definitions"`
	Paths        int     `json:"paths"`
	Err          string  `json:"err,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "loader-benchmark:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dir, pattern, label, corpus, cache string
		summarize, baselineLabel           string
		scanModels                         bool
		extra                              extraOptions
	)

	flag.StringVar(&dir, "dir", "", "corpus checkout to scan")
	flag.StringVar(&pattern, "pattern", "./...", "package pattern to scan")
	flag.StringVar(&label, "label", "unlabelled", "version/configuration label for the report")
	flag.StringVar(&corpus, "corpus", "corpus", "corpus name for the report")
	flag.StringVar(&cache, "cache", "warm", "build-cache state this run was made in: warm or cold")
	flag.BoolVar(&scanModels, "scan-models", true, "emit swagger:model definitions too")
	flag.StringVar(&summarize, "summarize", "", "comma-separated result files to tabulate instead of measuring")
	flag.StringVar(&baselineLabel, "baseline-label", "v0.36.3", "label deltas are computed against")
	extra.register()
	flag.Parse()

	if summarize != "" {
		return summarizeFiles(strings.Split(summarize, ","), baselineLabel, os.Stdout)
	}

	if dir == "" {
		return errNoCorpus
	}

	m := measure(dir, pattern, label, corpus, cache, scanModels, extra)

	return json.NewEncoder(os.Stdout).Encode(m)
}

// measure runs the scan and reports it. A failing scan is reported, not fatal: a corpus that does not
// scan cleanly still measures a load, and saying so beats dropping the cell silently.
func measure(dir, pattern, label, corpus, cache string, scanModels bool, extra extraOptions) measurement {
	opts := &codescan.Options{
		Packages:   []string{pattern},
		WorkDir:    dir,
		ScanModels: scanModels,
	}
	extra.apply(opts)

	runtime.GC()

	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	spec, err := codescan.Run(opts)
	wall := time.Since(start)

	// The spec is still referenced here, so what survives this GC is what a caller ends up holding —
	// not what the loader held at its peak. Peak RSS is the figure for that.
	runtime.GC()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	m := measurement{
		Label:        label,
		Corpus:       corpus,
		Cache:        cache,
		WallSeconds:  wall.Seconds(),
		RetainedMB:   float64(after.HeapAlloc) / bytesPerMB,
		TotalAllocMB: float64(after.TotalAlloc-before.TotalAlloc) / bytesPerMB,
		PeakRSSMB:    peakRSSMB(),
	}

	if err != nil {
		m.Err = err.Error()
	}

	if spec != nil {
		m.Definitions = len(spec.Definitions)
		if spec.Paths != nil {
			m.Paths = len(spec.Paths.Paths)
		}
	}

	return m
}

// peakRSSMB reads the process high-water mark from /proc, which is the only place it is recorded.
// Elsewhere the field is simply absent and the memory column reads zero.
func peakRSSMB() float64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0
	}

	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 || fields[0] != "VmHWM:" {
			continue
		}

		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0
		}

		return kb / kbPerMB
	}

	return 0
}

// aggregate is every run of one label on one corpus in one cache state, reduced to its means.
type aggregate struct {
	label   string
	n       int
	wall    float64
	rss     float64
	alloc   float64
	defs    int
	paths   int
	failing bool
}

func summarizeFiles(files []string, baselineLabel string, out *os.File) error {
	rows, err := readAll(files)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		return errNoMeasurements
	}

	for _, group := range groupKeys(rows) {
		writeTable(out, group, aggregateGroup(rows, group), baselineLabel)
	}

	return nil
}

func readAll(files []string) ([]measurement, error) {
	var rows []measurement

	for _, name := range files {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		f, err := os.Open(name)
		if err != nil {
			return nil, fmt.Errorf("reading results: %w", err)
		}

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}

			var m measurement
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				_ = f.Close()

				return nil, fmt.Errorf("parsing %s: %w", name, err)
			}

			rows = append(rows, m)
		}

		_ = f.Close()
	}

	return rows, nil
}

// groupKey is one table: a corpus in a cache state.
type groupKey struct {
	corpus string
	cache  string
}

func groupKeys(rows []measurement) []groupKey {
	seen := map[groupKey]bool{}

	var keys []groupKey

	for _, r := range rows {
		k := groupKey{corpus: r.Corpus, cache: r.Cache}
		if !seen[k] {
			seen[k] = true

			keys = append(keys, k)
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].corpus != keys[j].corpus {
			return keys[i].corpus < keys[j].corpus
		}

		return keys[i].cache > keys[j].cache // warm before cold
	})

	return keys
}

// aggregateGroup keeps first-seen label order, which is the order run.sh measures in and the order a
// reader expects: baseline first, then each configuration.
func aggregateGroup(rows []measurement, k groupKey) []aggregate {
	var (
		order []string
		byLab = map[string]*aggregate{}
	)

	for _, r := range rows {
		if r.Corpus != k.corpus || r.Cache != k.cache {
			continue
		}

		a, ok := byLab[r.Label]
		if !ok {
			a = &aggregate{label: r.Label, defs: r.Definitions, paths: r.Paths}
			byLab[r.Label] = a

			order = append(order, r.Label)
		}

		a.n++
		a.wall += r.WallSeconds
		a.rss += r.PeakRSSMB
		a.alloc += r.TotalAllocMB

		if r.Err != "" {
			a.failing = true
		}
	}

	aggs := make([]aggregate, 0, len(order))

	for _, lab := range order {
		a := byLab[lab]
		if a.n > 0 {
			a.wall /= float64(a.n)
			a.rss /= float64(a.n)
			a.alloc /= float64(a.n)
		}

		aggs = append(aggs, *a)
	}

	return aggs
}

func writeTable(out *os.File, k groupKey, aggs []aggregate, baselineLabel string) {
	var base *aggregate

	for i := range aggs {
		if aggs[i].label == baselineLabel {
			base = &aggs[i]

			break
		}
	}

	fmt.Fprintf(out, "\n### %s — %s build cache\n\n", k.corpus, k.cache)
	fmt.Fprintf(out, "%-26s %8s %7s %9s %7s %9s %7s  %s\n",
		"configuration", "wall", "delta", "peakRSS", "delta", "alloc", "delta", "shape")

	for _, a := range aggs {
		shape := fmt.Sprintf("%dd/%dp", a.defs, a.paths)
		if a.failing {
			shape += " SCAN FAILED"
		}

		fmt.Fprintf(out, "%-26s %7.3fs %7s %8.0fM %7s %8.0fM %7s  %s (n=%d)\n",
			a.label,
			a.wall, delta(a.wall, base != nil, baseOr(base, func(b aggregate) float64 { return b.wall })),
			a.rss, delta(a.rss, base != nil, baseOr(base, func(b aggregate) float64 { return b.rss })),
			a.alloc, delta(a.alloc, base != nil, baseOr(base, func(b aggregate) float64 { return b.alloc })),
			shape, a.n)
	}
}

func baseOr(base *aggregate, pick func(aggregate) float64) float64 {
	if base == nil {
		return 0
	}

	return pick(*base)
}

func delta(v float64, haveBase bool, base float64) string {
	if !haveBase || base == 0 {
		return ""
	}

	return fmt.Sprintf("%+.0f%%", (v/base-1)*percentScale)
}
