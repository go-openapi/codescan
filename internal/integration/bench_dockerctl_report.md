# Grammar vs. regexp parser — dockerctl scan benchmark

Comparison of the comment-annotation parser before and after the
regexp→grammar migration, run against a sizable real-world corpus.

> [!NOTE]
> This report measures the **Build** phase — the parser. It is no longer the
> whole story: [`bench_scanner_axes.md`](./bench_scanner_axes.md) measures the
> **Load** phase, which is 94% of a scan, along the axes the on-demand scanner
> work is steered by. Read that one for anything about loading, dependency
> types or memory retention; this one stands as the parser's own record.

## Method

- **Harness:** [`bench_dockerctl_test.go`](./bench_dockerctl_test.go) — a
  self-contained program that touches only the public `Run` entry point plus
  the two internal phases (`scanner.NewScanCtx` + `spec.NewBuilder`). These
  exist identically in both trees, so the file compiles **verbatim** in each
  worktree.
- **Corpus:** the go-swagger **dockerctl** client
  (`github.com/go-swagger/dockerctl`) — ~502 Go files, ~160k LOC, 121 files
  carrying `swagger:` annotations.
- **Worktrees compared:**
  - **regexp** — tag `v0.34.0` (after the package-layout refactor landed,
    before the grammar parser; still fully regexp-based).
  - **grammar** — `master` (grammar-based parser).
- **Phase split:** the scan is measured in two parts so the parser change is
  isolated from the unchanged front end:
  - **Load** — `scanner.NewScanCtx`: `go/packages` type-checking. *Identical*
    work in both trees; a stable baseline. It dominates wall-clock, which is
    why the end-to-end `Full` number barely moves.
  - **Build** — `spec.NewBuilder(...).Build()`: annotation parsing + Swagger
    emission. **This is the phase that changed**, and where the win shows up.
- **Measurement:** wall-clock time plus heap allocations (`TotalAlloc` bytes
  and `Mallocs` count deltas, GC-fenced). Figures below are the stabilized
  10-iteration `-benchmem` run; a single iteration is enough and reproduces
  the same ratios.

## Results — Build phase (the regexp→grammar delta)

| Metric  | Regexp (v0.34.0) | Grammar (master) | Improvement            |
| ------- | ---------------- | ---------------- | ---------------------- |
| Time    | 225.8 ms/op      | 115.8 ms/op      | **1.95× faster** (−49%) |
| Memory  | 67.7 MB/op       | 28.8 MB/op       | **2.35× less** (−57%)   |
| Allocs  | 601,674/op       | 298,105/op       | **2.02× fewer** (−50%)  |

Single-iteration figures agree: regexp 212 ms / 65 MB / 601k allocs vs.
grammar 128 ms / 28 MB / 298k allocs.

## Load phase (baseline — unchanged front end)

| Metric  | Regexp        | Grammar       |
| ------- | ------------- | ------------- |
| Memory  | ~1088 MB      | ~1088 MB      |
| Allocs  | ~12.19M       | ~12.19M       |

Effectively identical, as expected — both call the same `go/packages`
loader. This is the bulk of total cost, so it is the right thing to factor
out when judging the parser.

## Verdict

The grammar parser **wins decisively** on this corpus: it nearly halves
parse time and more than halves both bytes allocated and allocation count.
The regexp engine's repeated pattern-matching and intermediate string
allocation is exactly what the grammar-based single-pass approach removes.

## Reproduce

From either worktree:

```sh
# one-shot human-readable report
go test ./internal/integration/ -run TestDockerctlBenchReport -v

# stabilized Go benchmark
go test ./internal/integration/ -run x -bench BenchmarkDockerctl/Build -benchtime=10x -benchmem
```

Point the harness at the client with `CODESCAN_BENCH_DIR` if it is not at the
default path; the program skips (rather than fails) when the corpus is absent.

## Profiling

`go test -bench` cannot collect a `-cpuprofile`/`-memprofile` across multiple
packages, so [`bench_dockerctl_profile_test.go`](./bench_dockerctl_profile_test.go)
drives `runtime/pprof` directly (stdlib only — no `github.com/pkg/profile`
dependency). It profiles the **Build** phase by default, loading the
`ScanCtx` once and replaying `Build()` in a loop so the 100 Hz CPU sampler
gathers enough samples on the ~115 ms workload.

```sh
# CPU profile of the grammar parser (Build phase)
CODESCAN_BENCH_CPUPROFILE=cpu.out \
  go test ./internal/integration/ -run TestDockerctlProfile -v
go tool pprof -http=: cpu.out

# heap profile (alloc_space)
CODESCAN_BENCH_MEMPROFILE=mem.out CODESCAN_BENCH_ITERS=200 \
  go test ./internal/integration/ -run TestDockerctlProfile -v
go tool pprof -sample_index=alloc_space -http=: mem.out
```

Knobs: `CODESCAN_BENCH_PHASE=build|full` (default `build`),
`CODESCAN_BENCH_ITERS=N` (default 30). The test skips unless at least one
profile path is set.

**Caveat:** the CPU profile brackets only the replay loop and so cleanly
isolates the chosen phase (Load runs before `StartCPUProfile`). The heap
profile is process-cumulative, so for `phase=build` its `alloc_space` still
includes the one-time Load (`go/types`) allocations — raise
`CODESCAN_BENCH_ITERS` so the replayed Build dominates, or compare the same
iter count across worktrees and read the delta.
