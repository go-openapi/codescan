# Scanner measurement harness — the axes, how to run them, how to read them

This is the control loop for the on-demand scanner work: what a scan pays, and what it gets for it.
It exists because none of the figures that steer that work were reproducible from the repo.

Two halves, and only the pair is a control loop:

| half | what it answers | where |
|---|---|---|
| **cost** | what a load costs and what shape it leaves behind | `bench_corpus_test.go`, `bench_loaders_test.go`, `bench_discovery_test.go`, `bench_workingset_test.go` |
| **output** | whether a cheaper configuration still produces the same spec | `loader_agreement_test.go` |

A lazy load that is faster *and* silently drops a `swagger:strfmt` format is precisely what must not
ship, so a cost improvement is only reportable alongside the output half.

The older [`bench_dockerctl_report.md`](./bench_dockerctl_report.md) and
[`bench_dockerctl_design_notes.md`](./bench_dockerctl_design_notes.md) measure a different thing —
the regexp→grammar parser migration — and still stand. They are no longer the whole story: they
measure the *build* phase, and everything here measures the *load*.

## Corpora

Two external checkouts, neither vendored. Both are skipped (never failed) when absent, which is the
normal case in CI.

| corpus | default location | override | shape |
|---|---|---|---|
| dockerctl | `/home/fred/src/github.com/go-swagger/dockerctl` | `CODESCAN_BENCH_DIR` | generated client: 336-package closure, **2** annotated packages |
| go-swagger | `/home/fred/src/github.com/go-swagger/go-swagger` | `CODESCAN_BENCH_GOSWAGGER_DIR` | hand-written: 440-package closure, **14** annotated packages |

Two shapes on purpose. dockerctl is lucky — nearly all of it is unannotated generated CLI code — and
a conclusion that only holds there is not a conclusion.

## Running it

Everything cost-related is gated behind `CODESCAN_BENCH=1`. The gate is not about CI (the corpora
are absent there anyway); it is so that a plain `go test ./...` on a machine that *does* have them
does not grow by minutes.

```sh
# the whole cost half
CODESCAN_BENCH=1 go test ./internal/integration/ -run 'TestBench' -v -timeout 30m

# one axis
CODESCAN_BENCH=1 go test ./internal/integration/ -run TestBenchLoadLadders -v
```

| axis | what it measures | entry point |
|---|---|---|
| **A** | loader strategy: `go/packages` vs the toolchain-free loader | `TestBenchLoadLadders` |
| **B** | dependency types: from source vs from compiled export data, **warm and cold** | `TestBenchLoadLadders`, `TestBenchLoadLaddersCold` |
| **C** | a bundled export-data blob (the toolchain-free route) | `TestBenchLoadLadders` |
| **D** | the raw `go/packages` LoadMode ladder | `TestBenchLoadLadders` |
| **E** | the *shape* of a load, not its cost | every rung of the above |
| **F** | discovery cost, separated from loading | `TestBenchDiscovery` |
| **G** | the working set: annotated roots → type-reference closure | `TestBenchWorkingSet` |
| **H** | what the per-dependency export-data policy loads vs what a scan reaches | `TestBenchDependencyRouting` |

Plus `BenchmarkLoadStrategies`, the same A/B/C rungs as a real benchmark for `benchstat`:

```sh
go test ./internal/integration/ -run x -bench BenchmarkLoadStrategies -benchtime=1x
```

### Why some axes are tests rather than benchmarks

What most of these produce is a *table* — a ladder of rungs with four or five columns each — not a
scalar that gets faster or slower. `go test -bench` has one output slot per iteration, and forcing
Axis G ("4 packages of 336") through it would lose the thing worth reading. Axes A–C are also a
like-for-like comparison, so those additionally exist as `BenchmarkLoadStrategies`.

### Cold vs warm build cache

This distinction matters more than any other single knob, and the harness refuses to leave it
implicit.

- **Warm** is the default, and it is *enforced*: every rung in `TestBenchLoadLadders` runs once with
  the result discarded before the measured run. A number in that report is never accidentally cold.
- **Cold** is `TestBenchLoadLaddersCold`, opt-in via `CODESCAN_BENCH_COLD=1`. It gives each rung a
  private, empty `GOCACHE` (a `t.TempDir()`), so "cold" means a well-defined extreme rather than
  "whatever had not been built yet". `GOMODCACHE` is left alone, so nothing is downloaded.

Only the rungs whose cost depends on the build cache are run cold; a load that reads dependency
source compiles nothing, so a fresh `GOCACHE` would measure the same thing twice.

```sh
CODESCAN_BENCH=1 CODESCAN_BENCH_COLD=1 \
  go test ./internal/integration/ -run TestBenchLoadLaddersCold -v -timeout 60m
```

### Axis C — the export-data blob

`Options.ExportData` needs a blob, which is not checked in: the export format is tied to the Go
release that produced it, and a blob covers one corpus's dependency closure and no other. Produce
one per corpus and point the harness at it:

```sh
go run ./hack/genexportdata -dir /path/to/dockerctl -out /tmp/dockerctl-exportdata.zip ./...
export CODESCAN_BENCH_EXPORTDATA_DOCKERCTL=/tmp/dockerctl-exportdata.zip

go run ./hack/genexportdata -dir /path/to/go-swagger -out /tmp/goswagger-exportdata.zip ./...
export CODESCAN_BENCH_EXPORTDATA_GOSWAGGER=/tmp/goswagger-exportdata.zip
```

`CODESCAN_BENCH_EXPORTDATA` is honoured as a fallback for whichever corpus has no variable of its
own. Without a blob the axis is skipped with the command printed — it is never silently omitted.

Handing a corpus somebody else's blob does not fail: every package the blob does not cover falls
back to source. That is exactly why the variables are per corpus, because the result would look
like a measurement and be a fallback. Measured, the difference is large — go-swagger under
dockerctl's blob reads 168 packages from source in 0.70 s / 123 MB; under its own, 31 packages in
0.33 s / 30 MB.

## Reading the output

Every cost row shares one format:

```
[B] loader: go/packages + CompiledDependencies    0.42s  retained= 105 MB  graph=336 goFiles=336 types= 42 fromSource= 19 exportServed= 23 exportOnly=  0 astFiles=502
```

| column | meaning |
|---|---|
| `retained` | heap **held** after the load, GC-fenced, measured as a delta against a pre-rung reading. Not `TotalAlloc`: the scanner's problem is retention, not churn |
| `graph` | packages reachable through `Package.Imports` |
| `goFiles` | of those, how many have `GoFiles` — i.e. whose source is *locatable* |
| `types` | of those, how many have complete type information |
| `fromSource` | of those, how many have an AST |
| `exportServed` | of those, how many have complete types and **no** AST |
| `exportOnly` | how many the loader reported (`WithOnExportOnly`) as having no reachable source at all |
| `astFiles` | parsed files held |

`fromSource` / `exportServed` / `exportOnly` are the three-way state the on-demand work turns on:
loaded, served-without-syntax, and unreadable. The middle one is the interesting one — source
locatable, AST absent — and each rung also spotlights one dependency by name
(`CODESCAN_BENCH_SPOTLIGHT`, default `github.com/go-openapi/strfmt`) so the state has a face:

```
github.com/go-openapi/strfmt: GoFiles=9  Types=true  Syntax=0
```

### Two caveats worth knowing before quoting a number

1. **`graph` under `ExportData` is not the closure size.** A package served from export data has no
   `Imports` map to walk (`internal/packages.exportedPackage` builds an empty one — nothing
   downstream reads it), so the walk stops there. Under that rung `graph` means "packages the loader
   materialised", which is the figure the retained heap corresponds to. Closure size comes from a
   rung that walks the whole graph.
2. **`retained` is a whole-process delta.** It is fenced with GCs and taken before/after, so it is
   the heap the rung *holds*, but a rung that ends with less live heap than it started reports 0
   rather than a negative.

## Reference figures

Measured on this branch, go1.26.5, warm build and page cache, single runs. Reproduce with the
commands above; they move by ±10% run to run.

### Axis D — the LoadMode ladder (dockerctl, 336 packages)

| mode | wall | retained | astFiles |
|---|---|---|---|
| current (`Deps+Types+Syntax+TypesInfo`) | 0.91 s | 512 MB | 2354 |
| all types, no syntax | 0.69 s | 80 MB | 0 |
| `&^ NeedDeps` (= `CompiledDependencies`) | 0.39 s | 105 MB | 502 |
| syntax only, no types | 0.16 s | 38 MB | 502 |
| `go list` only | 0.08 s | 1 MB | 0 |

### Axes A–C — what the scanner actually runs

dockerctl:

| loader | wall | retained | graph | fromSource | exportServed |
|---|---|---|---|---|---|
| go/packages (default) | 0.96 s | 512 MB | 336 | 335 | 1 |
| go/packages + `CompiledDependencies` | 0.42 s | 105 MB | 336 | 19 | 23 |
| toolchain-free | 1.12 s | 277 MB | **322** | 322 | 0 |
| toolchain-free + `ExportData` | 0.39 s | 87 MB | 61¹ | 20 | 41 |

go-swagger:

| loader | wall | retained | graph | fromSource | exportServed |
|---|---|---|---|---|---|
| go/packages (default) | 0.81 s | 550 MB | 440 | 439 | 1 |
| go/packages + `CompiledDependencies` | 0.22 s | 30 MB | 440 | 30 | 75 |
| toolchain-free | 1.22 s | 288 MB | **433** | 433 | 0 |
| toolchain-free + `ExportData` | 0.33 s | 30 MB | 119¹ | 31 | 88 |

¹ see caveat 1 above — not a closure size.

The **graph divergence between strategies is real and reproducible**: 336 vs 322 on dockerctl,
440 vs 433 on go-swagger. It belongs to the loader branch, not to this harness, but it is visible
from here.

### Axis B — cold vs warm, the distinction that matters most

Each rung with its own empty `GOCACHE`:

| rung | dockerctl warm | dockerctl cold | go-swagger warm | go-swagger cold |
|---|---|---|---|---|
| default (`Deps+Types+Syntax+TypesInfo`) | 0.93 s | 1.46 s | 0.81 s | 1.54 s |
| `CompiledDependencies` | **0.40 s** | **9.08 s** | **0.22 s** | **8.40 s** |

Warm, `CompiledDependencies` is 2.3× *faster* than the default. Cold, it is 6× *slower*. The option
is not "faster"; it is "faster once the compiler has already done the work", and quoting one figure
without the other is how a benchmark misleads.

The shape is identical in both states — 23 packages export-served either way — so nothing about the
result changes, only when the compilation is paid for.

Note this is a harder "cold" than the plan's earlier 4.06 s reading, which was taken on a partially
warm cache. An empty `GOCACHE` is the well-defined extreme; a working developer's cache sits
somewhere between the two columns.

### Axis F — discovery without loading

| step | dockerctl | go-swagger |
|---|---|---|
| `go list` (graph + `GoFiles`) | 0.08 s — 335 pkgs, 2341 files, 22.3 MB | 0.10 s — 439 pkgs, 2378 files, 23.4 MB |
| byte prefilter over **all** files | 0.05 s — 127/2341 survive (5.4%) | 0.06 s — 142/2378 survive (6.0%) |
| parse + classify survivors | 0.03 s | 0.03 s |
| **total** | **0.16 s**, 38 MB transient, **1 MB retained** | **0.18 s**, 38 MB transient, **1 MB retained** |

A complete annotation index over the whole closure, retaining nothing, for under a fifth of a
second — against 0.93 s and 512 MB for the load that produces the same index as a side effect. (The
`go list` row is the one that moves most between runs; an earlier reading put it at 0.16 s.)

The prefilter can only over-admit (a `swagger:` in a string literal costs one wasted parse) and
never under-admit, since a real annotation contains that literal by definition.

### Axis G — the working set

| corpus | closure | annotated roots | type-reference closure | never needed |
|---|---|---|---|---|
| dockerctl | 336 | 2 (168 decls) | **4** | 332 (98.8%) |
| go-swagger | 440 | 14 (144 decls) | **19** | 421 (95.7%) |

Packages reached purely by following a field's type, carrying no annotation at all — dockerctl:
`github.com/oklog/ulid`, `time`; go-swagger: `strfmt/internal/countries`,
`scan-repo-boundary/makeplans`, `oklog/ulid/v2`, `golang.org/x/text/currency`, `time`.

The walk approximates the schema builder (struct fields, elems, map keys/values, type args,
interface methods, signature results). It is good enough to *size* the closure and not good enough
to prove an incremental loader reaches the same set — that is what the output half is for.

### Axis H — what is parsed vs what is needed

The per-dependency export-data policy routes a dependency on a **substring** test for `swagger:`.
The scanner's real rule requires the literal to *start* the comment line. The gap between those two
rules, and between both and the closure a scan actually reaches, is the cost still on the table:

dockerctl (336 packages, 2351 files):

| set | wall | retained | packages | files |
|---|---|---|---|---|
| everything in the closure | 0.39 s | 175 MB | 336 (100%) | 2351 |
| admitted by substring (**the loader's rule**) | 0.04 s | 16 MB | 14 (4.2%) | 299 |
| admitted by line-start (a real annotation) | 0.01 s | 5 MB | 2 (0.6%) | 91 |
| reached by the type-reference closure | 0.02 s | 6 MB | 4 (1.2%) | 102 |

go-swagger (440 packages, 2388 files):

| set | wall | retained | packages | files |
|---|---|---|---|---|
| everything in the closure | 0.43 s | 188 MB | 440 (100%) | 2388 |
| admitted by substring | 0.02 s | 8 MB | 35 (8.0%) | 197 |
| admitted by line-start | 0.02 s | 6 MB | 22 (5.0%) | 147 |
| reached by the type-reference closure | 0.02 s | 6 MB | 19 (4.3%) | 111 |

Read it this way: the per-dependency policy already captures nearly the whole prize (0.39 s → 0.04 s
on dockerctl). What is left is the **substring → line-start** step, worth ~0.03 s and ~11 MB on
dockerctl and ~nothing on go-swagger. That is small, and worth reporting precisely *because* it is
small: it says the next win is not here.

The 14-vs-2 gap on dockerctl is the hazard the plan predicted, now measured: codescan's own godoc,
read as a dependency, discusses `swagger:` constantly, so twelve packages are loaded from source for
prose. `isAnnotationLine` in `bench_discovery_test.go` is the tighter rule, restated without a
regexp.

## The output half — spec agreement across configurations

[`loader_agreement_test.go`](./loader_agreement_test.go) A/Bs whole marshalled documents across the
configurations that change *how dependency types arrive*, over the fixture corpus. It extends
`TestLoaderChoice_AgreeOnTheRealFilesystem`, which had the right instinct (whole documents, not spot
checks) on one fixture and one axis.

Configurations compared against the plain scan: `ToolchainFreeLoader` (the control — it changes the
loader, not where types come from), `CompiledDependencies`, and `ExportData` when a blob is given.

Two tiers:

```sh
# default: the curated targets — the fixtures whose meaning comes from a dependency
go test ./internal/integration/ -run TestLoaderConfigs_AgreeOnTheFixtureCorpus

# the whole corpus (minutes)
CODESCAN_AB_CORPUS=1 go test ./internal/integration/ -run TestLoaderConfigs -timeout 60m

# with the export-data configuration too
go run ./hack/genexportdata -dir fixtures -out /tmp/fixtures-exportdata.zip std
CODESCAN_AB_EXPORTDATA=/tmp/fixtures-exportdata.zip go test ./internal/integration/ -run TestLoaderConfigs
```

A configuration that legitimately differs is recorded in `abExpected` with a reason, and asserted as
a **difference**. When the stream closes one of those gaps the assertion fails — which is the point;
an expectation that quietly survives its own fix is not an expectation. Regenerate the table with
`CODESCAN_AB_REPORT=1`.

### What it currently finds

Over all 306 fixture bundles, **12** (configuration, target) pairs diverge. `ToolchainFreeLoader`
diverges nowhere, which is what makes it a usable control. The other twelve fall into three
families, and the split is the finding:

| family | what happens | affected |
|---|---|---|
| **the declaration contract** | the scan **fails**, it does not degrade — a builder asks for the declaration of a type whose package has types and no AST | 4 targets under `compiled-dependencies`, 3 of them also under `export-data` |
| **a dependency's own annotations are lost** | `format: date-time` / `email` / `uuid` vanish, because the marks live in strfmt's source | 3 targets, `compiled-dependencies` only |
| **a dependency-declared model collapses** | the definition itself goes, not one keyword — `Booking` is declared in `scan-repo-boundary/makeplans` | 2 targets, `compiled-dependencies` only |

Two things worth carrying back to the plan:

1. **`CompiledDependencies` does not merely lose formats — on four of these targets the scan
   errors.** `unable to find package and source file for: time.Duration`,
   `can't find source file for type: interface{Write(p []byte) (n int, err error)}` (io.Writer),
   the same for `reflect.Type`, and `unable to find package and source file for:
   github.com/go-openapi/strfmt.DateTime`. That is a stronger statement than "quietly poorer", and
   it means the declaration contract is a shipping-blocker for that option rather than a tidiness
   item.
2. **The per-dependency policy already fixes the annotation families.** `export-data` loses no
   format and no definition, because a dependency whose source carries `swagger:` is read from
   source. Its three remaining divergences are all the declaration contract, all on **stdlib**
   types, where there is no annotated source to fall back to. That is precisely the split the plan
   predicted.
