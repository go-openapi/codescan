# loader-benchmark

Measures what a codescan scan costs, comparing a **released baseline against the working tree** over
external corpora, warm and cold.

It exists because the benchmarks under `internal/integration/` cannot answer *"how much did we
improve?"*. Those live inside the module, so they do not exist in any earlier release; they can
measure this version against itself and nothing else. This harness talks only to the public
`codescan.Run` API — the one surface every version has — so the same source compiles against a
release tag and against the working tree.

Not run in CI, by design. On shared runners a benchmark measures the runner. This is an operator
tool, run deliberately, roughly once per minor release.

**Looking for which loader to use, rather than for the evidence?** The primer is
`internal/scanner/README.md#loader`; this document is where its figures come from.

## Running it

Corpora are external, large, and never vendored here — the point of measuring them is that they are
real trees. Name each one through the environment; anything unset is skipped by name rather than
silently defaulting to some other checkout.

```sh
export CODESCAN_BENCH_DOCKERCTL_DIR=/path/to/generated/client
export CODESCAN_BENCH_KUBEAPI_DIR=/path/to/generated/server
export CODESCAN_BENCH_KUBEAPI_VENDORED_DIR=/path/to/the/same/server/vendored

./run.sh                          # warm matrix
CODESCAN_BENCH_COLD=1 ./run.sh    # add the cold pass (slow — it compiles dependency closures)
```

| variable | meaning | default |
|---|---|---|
| `CODESCAN_BENCH_BASELINE` | release the working tree is compared against | `v0.36.3` |
| `CODESCAN_BENCH_ROUNDS` | measured rounds per warm cell | `3` |
| `CODESCAN_BENCH_COLD` | `1` adds the cold-cache pass | off |
| `CODESCAN_BENCH_PATTERN` | package pattern handed to the scan | `./...` |

Raw measurements accumulate in `results/` as JSON lines, and the table is a pure function of them:

```sh
go run . -summarize results/warm.jsonl,results/cold.jsonl
```

Both `results/` and `.work/` (the generated baseline module and built probes) are gitignored.

## How it measures, and why that way

**One scan per process.** Peak RSS is a process-wide high-water mark, so a second scan in the same
process would report the first one's peak plus its own garbage.

**A discarded warm-up per configuration, then alternating rounds.** Measuring config-by-config lets
cache and thermal drift land entirely on whichever ran last; alternating spreads it evenly.

**Shape is recorded with cost.** Every row carries its definition and path counts, because a
configuration that got faster by *scanning less* is not an improvement. In the results below every
configuration emits the identical document for its corpus.

**The cold pass gives each cell a private empty `GOCACHE`** and leaves `GOMODCACHE` alone, so it
isolates the build cache and nothing else. The operator's own cache is never touched.

**`retained` is not the loader's peak.** It is measured after `Run` returns, when the package graph is
garbage and only the spec is live — a couple of MB in every configuration. Use peak RSS and total
allocation for the memory story.

### Two traps worth knowing

**The baseline build must be built with `GOWORK=off`.** The workspace at the repository root would
substitute the local tree for the released library and quietly measure the working tree twice, with a
label claiming otherwise. `run.sh` handles this; anything reimplementing it must too.

**Options younger than the baseline sit behind a build tag.** `options_current.go` sets them;
`options_baseline.go` (tag `baseline`) accepts the same flags and *exits* rather than ignoring them.
A baseline binary that silently accepted `-compiled-deps` would emit a default-configuration
measurement under a label saying otherwise — an error invisible in the output.

## Results

Measured 2026-08-08, go1.26.5, baseline **v0.36.3**, on one machine (Ryzen 7, 31 GB). v0.36.3 has
exactly one way to load and scan, which is what makes it a baseline.

**Rows are labelled by configuration, not by default, and they predate v0.36.4 flipping the default.**
So `current` is the source-loading scan, which is now what `SkipCompiledDependencies` selects, and
`current + CompiledDependencies` is what a plain run does today. The harness flag keeps meaning what
it says for the same reason — the measurements stay comparable across the change.

| corpus | shape | `.go` files | emitted |
|---|---|---|---|
| `dockerctl` | generated client for a container-engine API | ~1490 | 198 defs / 0 paths |
| `kubeapi` | larger generated server, module, not vendored | ~2350 | 222 defs / 260 paths |
| `kubeapi-vendored` | the same tree with vendored dependencies | +646 (14 MB) | 222 defs / 260 paths |

### Warm build cache (n=3)

| configuration | wall | Δ | peak RSS | Δ | total alloc | Δ |
|---|---|---|---|---|---|---|
| **dockerctl** — v0.36.3 | 1.228 s | — | 684 MB | — | 1103 MB | — |
| current | 0.995 s | −19% | 682 MB | 0% | 1103 MB | 0% |
| current + `ToolchainFreeLoader` | 1.131 s | −8% | 394 MB | −42% | 585 MB | −47% |
| current + `CompiledDependencies` | 0.452 s | −63% | 169 MB | −75% | 237 MB | −78% |
| **kubeapi** — v0.36.3 | 1.704 s | — | 768 MB | — | 1214 MB | — |
| current | 1.430 s | −16% | 753 MB | −2% | 1214 MB | 0% |
| current + `ToolchainFreeLoader` | 1.309 s | −23% | 426 MB | −45% | 644 MB | −47% |
| current + `CompiledDependencies` | 0.916 s | −46% | 307 MB | −60% | 446 MB | −63% |
| **kubeapi-vendored** — v0.36.3 | 1.763 s | — | 752 MB | — | 1214 MB | — |
| current | 1.546 s | −12% | 754 MB | 0% | 1214 MB | 0% |
| current + `ToolchainFreeLoader` | 1.333 s | −24% | 422 MB | −44% | 640 MB | −47% |
| current + `CompiledDependencies` | 0.966 s | −45% | 303 MB | −60% | 446 MB | −63% |

### Cold build cache (n=1)

| configuration | cold wall | Δ vs baseline | build cache it produces |
|---|---|---|---|
| **dockerctl** — v0.36.3 | 1.987 s | — | 7.4 MB |
| current | 1.793 s | −10% | 7.4 MB |
| current + `ToolchainFreeLoader` | **1.228 s** | **−38%** | **4 KB** |
| current + `CompiledDependencies` | 10.695 s | **+438%** | 216 MB |
| **kubeapi** — v0.36.3 | 2.472 s | — | 7.6 MB |
| current | 2.204 s | −11% | 7.6 MB |
| current + `ToolchainFreeLoader` | **1.482 s** | **−40%** | **4 KB** |
| current + `CompiledDependencies` | 13.838 s | **+460%** | 229 MB |
| **kubeapi-vendored** — v0.36.3 | 2.495 s | — | 7.4 MB |
| current | 2.260 s | −9% | 7.4 MB |
| current + `ToolchainFreeLoader` | **1.391 s** | **−44%** | **4 KB** |
| current + `CompiledDependencies` | 15.631 s | **+527%** | 229 MB |

Memory is unaffected by cache state and is omitted from the cold table; it matches the warm figures.

## Analysis

### 1. The default path does not prune, and that is the whole explanation

Default to default, the current tree is 12–19% faster with **total allocation identical to four
digits** and peak RSS inside the run-to-run spread.

That is not a prefilter defeated by annotation-dense corpora — that would show *partial* savings.
It is that pruning lives in the toolchain-free importer (which chooses source or export data per
import) and in `CompiledDependencies` (which takes dependency types from export data), and the
default configuration selects **neither**. Ask for a pruning loader and the pruning appears at once:
−47% allocation, on every corpus, invariably.

So the default's gain is **CPU, not memory** — the same bytes allocated, less time spent on them.
A user who upgrades and changes nothing should expect a modest time win and no memory win. The −75%
figure belongs to an opt-in option and must never be quoted against the default path.

### 2. Corpus size decides whether the pure-Go loader is also faster

Its memory advantage is flat — −42 to −46%, every corpus, both vendoring modes. Wall clock is not:
on the smaller generated client the toolchain-free route is *slower* than the default (1.131 s vs
0.995 s, the concurrency gap its implementation notes call out), and on the larger server it is
*faster* (1.309 s vs 1.430 s). Scale does not change the memory story; it decides whether the
wall-clock deficit survives. It does not.

### 3. The toolchain-free loader is cache-independent — the strongest result here

It produces **4 KB** of build cache and its cold time equals its warm time, because it never invokes
the go command: there is no metadata to populate and nothing to compile. Every other configuration is
priming a toolchain it depends on.

This inverts the ranking, and no configuration wins both states:

| cache | fastest → slowest |
|---|---|
| **warm** | `CompiledDependencies` · toolchain-free · default · v0.36.3 |
| **cold** | **toolchain-free** · default · v0.36.3 · `CompiledDependencies` |

Cold, the toolchain-free route is 38–44% faster than the baseline and ~25% faster than the current
default, while holding 45% less memory. It is the only configuration whose cost is *predictable*,
which for a tool that runs in CI is worth more than a warm best case.

### 4. Compiled dependencies: the default, and when to turn it off

Fastest warm by a wide margin (−45 to −63%), and **4.4× to 6.3× slower than the baseline cold**,
materialising up to 229 MB of build cache to get there — `go list -export` must *compile* the
closure, not merely type-check it. A first scan of a freshly generated tree is precisely its worst
case, and CI is cold by definition.

What it does not cost is meaning: it emits the same document on every corpus measured here and
across the whole fixture corpus. Whatever the spec needs out of a dependency — its own annotations,
or a declaration the scanned code names — is read, the second kind at the lookup that wants it. That
is what made it the default in v0.36.4. See `internal/scanner/README.md#compiled-dependencies`.

Nor does it cost the ability to scan code that does not compile: needing `go list` to build the
scanned packages would resurrect go-swagger#2874, so a load that fails that way is retried from
source. The retry costs a second load, and only on a tree that was not going to build.

`SkipCompiledDependencies` opts out, and the cold column is the reason to reach for it.

## Caveats

- **Vendoring is inconclusive here, not a win.** Comparing the vendored tree against the same tree
  unvendored, vendoring moves everything by a few percent and helps peak RSS only on the
  toolchain-free route. 14 MB of vendored source against a 2350-file module is too small a share of
  the closure to magnify anything. A corpus whose dependency closure dominates its own source would
  be needed to make this axis speak.
- **The cold pass is n=1** — cold is a single-shot state by definition. Its wall-clock figures carry
  the run-to-run spread visible in the warm tables (a few percent), which does not threaten
  conclusions drawn from 4× gaps but does mean small cold deltas should not be read closely.
- **The workload looks RAM-bound rather than CPU-bound.** Figures taken on a much older CPU with
  comparable RAM and SSD reproduce the cold numbers closely. Treat CPU model as a weak predictor
  here, and memory bandwidth as the thing to hold constant when comparing machines.
- **A corpus that fails to scan still measures a load.** The harness reports the error and marks the
  row `SCAN FAILED` rather than dropping it — but such a row measures load-then-abort, not a scan,
  and must not be compared against a clean one.
