# benchmarks — what a scan costs

What codescan pays to turn a Go tree into a Swagger document, measured on two real generated
projects that ship with this repository.

Two questions, and the whole of this document answers them:

1. **Which loader should I use?** They differ by 3× in memory and by an order of magnitude in time,
   and no single one wins in every state of the build cache.
2. **Has this got better?** Yes — a scan of the larger corpus allocates **3.7× less** than it did in
   v0.33.3 and takes **4.7× less** wall clock, emitting the identical document.

## The corpus

Both trees are go-swagger-generated projects. They ship as one archive
(`testdata/corpus.tgz`) and unpack on demand, so a measurement needs a clone and nothing else — no
external checkout, no environment variable naming somebody's home directory, no network. Both
vendor their dependencies, which is what makes that last one true.

| corpus | shape | own `.go` files | LOC | vendored | emits |
|---|---|---|---|---|---|
| `dockerctl` | generated **client** for the docker engine API | 501 | 161k | 985 files | 198 definitions, 0 paths |
| `kubeapi` | generated **server** for the kubernetes API | 2352 | 344k | 646 files | 222 definitions, 260 paths |

Two shapes on purpose. `dockerctl` is light code over a reasonably large API and carries its
annotations on models only; `kubeapi` is heavy code over a very large API, and 260 route/operation
blocks means the annotation grammar has real work to do. A conclusion that only holds on one of them
is not a conclusion — and, as the history section shows, the two disagree sharply.

Every table below carries the emitted **shape** (`198d/0p`) alongside the cost, because a
configuration that got faster by scanning less is not an improvement. Across every row of every
table, each corpus emits its identical document.

## Choosing a loader

Three configurations of the working tree, the same document out of each.

### Warm build cache (n=3)

| configuration | wall | peak RSS | allocated |
|---|---|---|---|
| **dockerctl** — source dependencies *(the default)* | 0.972 s | 679 MB | 1103 MB |
| pure-Go loader (`ToolchainFreeLoader`) | 1.134 s | **403 MB** | **589 MB** |
| compiled dependencies | **0.438 s** | **170 MB** | **237 MB** |
| **kubeapi** — source dependencies *(the default)* | 1.506 s | 751 MB | 1215 MB |
| pure-Go loader (`ToolchainFreeLoader`) | 1.331 s | 412 MB | 643 MB |
| compiled dependencies | **0.970 s** | **306 MB** | **447 MB** |

### Cold build cache (n=1, private empty `GOCACHE`)

| configuration | wall | build cache it writes |
|---|---|---|
| **dockerctl** — source dependencies *(the default)* | 1.490 s | 7.5 MB |
| pure-Go loader | **1.198 s** | **4 KB** |
| compiled dependencies | 9.361 s | 216 MB |
| **kubeapi** — source dependencies *(the default)* | 2.208 s | 7.7 MB |
| pure-Go loader | **1.359 s** | **4 KB** |
| compiled dependencies | 14.511 s | 231 MB |

Memory does not depend on cache state and is omitted here; it matches the warm figures.

### Reading it

**Compiled dependencies win warm and lose cold, by a lot.** Taking dependency types from the
compiler instead of their source is 35–55% faster and 2.5–4× smaller on a warm cache. On a cold one
it is **more than 6× slower** than reading source (6.3× and 6.6× on the two corpora), because
`go list -export` must *compile* the closure rather
than type-check it, and it materialises up to 231 MB of build cache doing so. `CompiledDependencies`
opts in, and a build cache that is warm by construction — a developer's machine, a watch loop, a
pipeline that restores its cache — is the case that wants it.

**The pure-Go loader is the only one whose cost is predictable.** It writes 4 KB of build cache and
its cold time equals its warm time, because it never invokes the go command — there is no metadata
to populate and nothing to compile. Everything else is priming a toolchain it depends on. That
inverts the ranking, and no configuration wins both states:

| cache | fastest → slowest |
|---|---|
| **warm** | compiled dependencies · source dependencies · pure-Go loader |
| **cold** | **pure-Go loader** · source dependencies · compiled dependencies |

Its memory advantage is flat — 45–47% less allocated on both corpora — while its wall clock is
corpus-dependent: *slower* than the standard loader warm on the smaller client, *faster* on the
larger server. Scale decides whether the wall-clock deficit survives; on the larger corpus it does
not.

**Which is why the default is the standard loader reading dependencies from source.** No
configuration wins everywhere, so the default is the one with no bad case: it is never an order of
magnitude off, it writes 7 MB of build cache rather than 231, and it does not care whether the cache
was there. The two better answers are both conditional, and each is one flag away — compiled
dependencies where the cache is warm by construction, and the pure-Go loader where memory is the
binding constraint or the cost has to be predictable. The pure-Go loader is the better trade of the
two on these figures and would be the natural default; it remains experimental, which is the only
reason it is not.

**Where the memory goes.** The load ladders (`bench_loaders_test.go`) report the shape behind those
figures — how much of the closure ends up as parsed syntax:

| corpus | configuration | closure | from source | export-served | AST files | retained |
|---|---|---|---|---|---|---|
| dockerctl | source dependencies | 335 | 334 | 1 | 2353 | 513 MB |
| dockerctl | compiled dependencies | 335 | 19 | 22 | 510 | 106 MB |
| dockerctl | pure-Go loader | 321 | 321 | 0 | 2286 | 277 MB |
| kubeapi | source dependencies | 297 | 296 | 1 | 3908 | 548 MB |
| kubeapi | compiled dependencies | 297 | 31 | 37 | 2368 | 188 MB |
| kubeapi | pure-Go loader | 283 | 283 | 0 | 3842 | 278 MB |

Reading 19 packages from source instead of 334 is the entire compiled-dependencies win, and it costs
no meaning: a dependency whose source carries annotations is read anyway, and a declaration the spec
needs is fetched at the lookup that wants it. See
[`internal/scanner/README.md#compiled-dependencies`](../scanner/README.md#compiled-dependencies).

The pure-Go loader resolves a slightly smaller closure than `go list` (321 vs 335, 283 vs 297). That
divergence is real and reproducible, belongs to the loader, and does not change the emitted document
on either corpus.

**Which one to pick** is answered in prose, with the same figures, at
[`internal/scanner/README.md#loader`](../scanner/README.md#loader).

## Six months of scans

The same measurement against the working tree and against two released versions, each built into its
own throwaway module so it links the released library rather than this checkout.

| version | landed | what changed |
|---|---|---|
| `v0.33.3` | the state codescan was extracted from go-swagger in | regexp-based annotation parsing |
| `v0.35.1` | the grammar parser, debugged | annotations parsed by a real grammar, with diagnostics |
| `current` | the loader work | dependency types from export data; a second loader |

### Warm, same document at every point

| configuration | dockerctl wall | alloc | kubeapi wall | alloc |
|---|---|---|---|---|
| v0.33.3 | 1.593 s | 1316 MB | 7.087 s | 4555 MB |
| v0.35.1 | 1.226 s | 1103 MB | 1.758 s | 1217 MB |
| current, source dependencies *(default)* | 0.972 s | 1103 MB | 1.506 s | 1215 MB |
| current, compiled dependencies | **0.438 s** | **237 MB** | **0.970 s** | **447 MB** |

End to end, six months moved the larger corpus from 7.1 s / 4555 MB to 1.51 s / 1215 MB in the
default configuration — **4.7× faster, 3.7× less allocated** — and to 0.97 s / 447 MB with compiled
dependencies asked for, which is **7.3× faster, 10× less allocated**. Every row emits the identical
222 definitions and 260 paths.

### The two halves moved for different reasons, on different corpora

The rows above split cleanly in two, and the split is the interesting part.

**v0.33.3 → v0.35.1 is the annotation parser, and it is only visible where there are routes.** On
`dockerctl` (no paths) the change is 16%; on `kubeapi` (260 paths) it is 3.7×. Isolating it with two
sub-patterns of the same corpus settles what that means:

| scan | emits | v0.33.3 | current |
|---|---|---|---|
| `kubeapi/./models/...` — nothing to build | 0 defs / 0 paths | 764.6 MB | 764.1 MB |
| `kubeapi/./restapi/...` — the whole document | 222 defs / 260 paths | 4475.6 MB | 1215.8 MB |

With nothing to emit, the two versions allocate the same to within 0.5 MB: the loading half is
unchanged, as it must be — both call the same `go/packages`. The whole 3.3 GB difference is the
phase that reads annotations and emits Swagger, and it appears only where route and operation bodies
do. The grammar migration was never a performance project — it was about correctness, diagnostics
and completeness — so this is a byproduct of dropping Go's `regexp` from the hot path, and it is
worth knowing that the byproduct is this large on route-heavy code.

**v0.35.1 → current is the loader**, which changes what a scan *holds* rather than what it does with
it.

### Reading the two gains

Each step against its predecessor, except the last two, which are both against v0.35.1 — they are
alternative configurations of the same tree, not successive ones.

| step | dockerctl wall / alloc / peak RSS | kubeapi wall / alloc / peak RSS |
|---|---|---|
| v0.33.3 → v0.35.1 — the parser | −0.367 s / **−213 MB** / −188 MB | **−5.329 s** / **−3338 MB** / −346 MB |
| v0.35.1 → current, source dependencies | −0.254 s / −0.5 MB / +13 MB | −0.252 s / −1.5 MB / −56 MB |
| v0.35.1 → pure-Go loader | −0.092 s / −514 MB / −263 MB | −0.427 s / −573 MB / −395 MB |
| v0.35.1 → compiled dependencies | **−0.788 s** / −866 MB / **−497 MB** | −0.788 s / −769 MB / **−501 MB** |

Three readings, because the headline table on its own looks like a contradiction.

**Which change won "on memory" depends on which memory column.** The parser's win is overwhelmingly
*churn*: removing 3338 MB of allocation on kubeapi moved the high-water mark by 346 MB, because the
regexp engine allocated and discarded, and the collector kept the peak near the floor that the
retained package graph sets. The loader attacks that floor instead — about −500 MB of peak on both
corpora. So total allocation makes the parser the larger win, by 4× on kubeapi, and peak RSS makes
the loader the larger win everywhere.

**The parser won wall clock too, and on route-heavy code it won most of it** — 5.33 s of kubeapi's
6.12 s. On dockerctl, with models to read and no routes, the ranking is the other way round (−0.37 s
against −0.79 s). "The parser bought memory and the loader bought time" is dockerctl's story, not a
general one.

**The two scale with different things**, which is what makes them predictable on a corpus resembling
neither:

- the **loader** acts on the dependency *closure*, similar in size for both (335 and 297 packages),
  so its win is near-constant in absolute terms — −866 / −769 MB allocated, −497 / −501 MB of peak,
  and −0.788 s on both, to three digits;
- the **parser** acts on the *annotated surface* — −213 MB where there are no paths, −3338 MB where
  there are 260.

So the loader's share of the total looks large on a small project and the parser's on an
annotation-dense one, and on a large generated server both matter, for unrelated reasons.

The middle row deserves its own note: in the same configuration, v0.35.1 → current is
allocation-neutral to within 2 MB and still a quarter-second faster on both corpora. Nothing about
pruning reached the default path over that stretch — only speed. Every memory figure below that row
comes from *asking* for a loader that prunes.

### What a scan is made of today

Per phase, warm, with compiled dependencies — the configuration the loader work was aimed at:

| corpus | Load | Build | Build's share |
|---|---|---|---|
| dockerctl | 415 ms · 223 MB | 33 ms · 14.7 MB | 7% of time, 6% of allocation |
| kubeapi | 802 ms · 390 MB | 156 ms · 56.9 MB | 16% of time, 13% of allocation |

Load is resolving and type-checking the package graph; Build is reading annotations and emitting the
document. Loading still dominates, so the loader holds the remaining leverage, and a parser
micro-optimisation is worth at most a tenth of a scan.

## Running it

Everything here is an operator tool, run deliberately. None of it runs in CI: on a shared runner a
benchmark measures the runner.

### The cross-version matrix

```sh
cd internal/benchmarks/loader-benchmark
./run.sh                                            # warm matrix, every corpus and configuration
CODESCAN_BENCH_COLD=1 ./run.sh                      # add the cold pass (slow: it compiles closures)
CODESCAN_BENCH_HISTORY="v0.33.3 v0.35.1" ./run.sh   # which releases to compare against
```

| variable | meaning | default |
|---|---|---|
| `CODESCAN_BENCH_HISTORY` | released versions to build probes for, oldest first | `v0.33.3 v0.35.1` |
| `CODESCAN_BENCH_ROUNDS` | measured rounds per warm cell | `3` |
| `CODESCAN_BENCH_COLD` | `1` adds the cold-cache pass | off |
| `CODESCAN_BENCH_PATTERN` | package pattern handed to the scan | `./...` |
| `CODESCAN_BENCH_EXTRA_CORPUS` | `name:/path` — measure somebody else's tree alongside the shipped ones | unset |

Raw measurements accumulate in `results/` as JSON lines and the table is a pure function of them, so
a run can be re-read without being repeated:

```sh
go run . -summarize results/warm.jsonl,results/cold.jsonl -baseline-label v0.33.3
```

`results/` and `.work/` (the generated version modules and built probes) are gitignored.

### The in-tree harness

Every report is behind `CODESCAN_BENCH=1`, so a plain `go test ./...` — CI's included — runs none
of it, and the corpus is not even unpacked.

```sh
# per-phase cost of one scan of each corpus
CODESCAN_BENCH=1 go test ./internal/benchmarks/ -run TestScanPhases -v

# the load ladders: what each way of loading charges, and the shape it leaves behind
CODESCAN_BENCH=1 go test ./internal/benchmarks/ -run TestBenchLoadLadders -v -timeout 30m
CODESCAN_BENCH=1 CODESCAN_BENCH_COLD=1 go test ./internal/benchmarks/ -run TestBenchLoadLaddersCold -v -timeout 60m

# as Go benchmarks
go test ./internal/benchmarks/ -run x -bench BenchmarkScan -benchtime=1x -benchmem
go test ./internal/benchmarks/ -run x -bench BenchmarkLoadStrategies -benchtime=1x
```

| variable | meaning |
|---|---|
| `CODESCAN_BENCH=1` | required for the reporting tests — they load whole package graphs repeatedly |
| `CODESCAN_BENCH_COLD=1` | adds the cold-cache pass, each rung with a private empty `GOCACHE` |
| `CODESCAN_BENCH_EXPORTDATA_DOCKERCTL` / `_KUBEAPI` | an export-data blob per corpus, for the bundled-export-data rung |
| `CODESCAN_BENCH_SPOTLIGHT` | one dependency to describe in detail under each rung (default `github.com/go-openapi/strfmt`) |
| `CODESCAN_BENCH_CPUPROFILE` / `_MEMPROFILE` | write a profile of one scan (`TestScanProfile`) |
| `CODESCAN_BENCH_PHASE` / `_ITERS` / `_CORPUS` | what that profile covers, how many replays, and of which corpus |

An export-data blob is not checked in — the format is tied to the Go release that produced it, and a
blob covers one corpus's closure and no other. Produce one with `go run ./hack/genexportdata`; the
skipped rung prints the exact command.

### How the numbers above were taken, and what would invalidate them

Measured 2026-08-16, go1.26.5, AMD Ryzen 7 5800X / 31 GB, one machine, warm page cache. Reproduce
with the commands above.

- **One scan per process.** Peak RSS is a process-wide high-water mark, so a second scan in the same
  process would report the first one's peak plus its own garbage.
- **A discarded warm-up per configuration, then alternating rounds.** Measuring config-by-config
  would let cache and thermal drift land entirely on whichever ran last.
- **Wall clock needs a quiet machine; memory does not.** Under load the timings here move by 3×
  while allocation and peak RSS reproduce to four digits. Distrust a wall-clock delta under 10%;
  the conclusions above rest on gaps of 2× and more.
- **The cold pass is n=1**, cold being a single-shot state by definition. It isolates the *build*
  cache and nothing else: each cell gets a private empty `GOCACHE`, `GOMODCACHE` is left alone, and
  the operator's own cache is never touched.
- **`GOWORK=off` everywhere.** The corpora unpack inside this repository, whose workspace does not
  list them, and a version probe built inside the workspace would link the working tree instead of
  the release it claims to measure. Both are handled; anything reimplementing this must handle them
  too.
- **A version probe rejects options younger than itself.** `options_baseline.go` accepts the same
  flags as the current build and *exits* rather than ignoring them — a probe that silently accepted
  `-compiled-deps` would report a source-dependency measurement under a label saying otherwise, the
  one measurement error that cannot be spotted in the output.
- **Peak RSS is read from `/proc`**, so that column reads zero off Linux.
