# Scanner performance — findings & redesign notes

Companion to [`bench_dockerctl_report.md`](./bench_dockerctl_report.md). The
report measures *what is*; this records *what we concluded* and *where the next
win is*, so it is on hand when we start designing the new scanner.

> [!NOTE]
> Written before the loader work started, and several of its predictions have
> since been measured rather than estimated — notably §6, which asked for a
> retention-aware harness. That harness now exists:
> [`bench_scanner_axes.md`](./bench_scanner_axes.md) reports retained heap per
> load configuration, the shape each one leaves behind, and the working set
> (§2's "what fraction is skippable" question, answered: 4 packages of 336).
> The estimates in §3 and §4 stand as estimates; the axes doc has the figures.

Corpus throughout: the go-swagger **dockerctl** client — 502 non-test `.go`
files, ~160k LOC, 121 annotated files (~36k LOC), 19 packages (13 annotated).

## 1. Where the cost actually is

A full `codescan.Run` of dockerctl, by phase:

| Phase | Time | Alloc (cumulative) | Allocs |
|-------|------|--------------------|--------|
| **Load** (`scanner.NewScanCtx`, go/packages full type-check) | ~4.1–5.4 s | ~1088 MB | ~12.19 M |
| **Build** (`spec.NewBuilder().Build()`, grammar parse + emit) | ~116 ms | ~28.8 MB | ~298 k |
| **Full** (fresh, Load+Build) | ~4.4 s | ~1115 MB | ~12.5 M |

**Load dominates: ~95%+ of time and ~97% of allocations.** The comment parser
is not the bottleneck — type resolution is.

Profiling confirms it (heap, `alloc_space`):

- `go/types` (type-checking) ≈ **73%** of the cumulative heap.
- A single function, `go/types.(*Checker).recordTypeAndValue`, ≈ **20%** (316 MB)
  on its own — it records type+value for *every typed expression*, including
  function bodies.
- Our `grammar` parser ≈ **6%** of total / ~18% of *Build-only* allocations /
  ~10% of Build CPU.

The grammar parser looking "invisible" in the profile is the **good** result:
it dropped below the type-resolution floor. The migration win is real but only
visible as the cross-worktree Build *delta* (everything else is byte-identical):
alloc −57%, allocs −50%, time −49% vs regexp v0.34.0.

**Yardstick:** further micro-optimization of the grammar parser is worth at most
~10–15% of *total* scan time. The lever that matters is type resolution.

## 2. The opportunity: AST-first discovery, targeted resolution

Replace the systematic full type-check of every package with:

1. a cheap **AST-only** pass over all files to discover annotations (finding
   `swagger:` comments is pure text/AST — no type info needed), then
2. type resolution **only on the interesting surface**.

### Granularity reality

`go/types` resolves at **package** granularity — you cannot type-check a single
decl without its package. Discovery is file/decl-level; resolution bottoms out
at the package. "Finer than file" buys nothing on the resolution side. The way
you go *finer than a package* is not sub-package checking — it's turning off
func-body checking and trimming the `types.Info` maps (below).

### dockerctl LOC split (what's skippable)

| Bucket | LOC | Share | Type-check needed? |
|--------|-----|-------|--------------------|
| `cli/` (cobra wrappers, generated) | 77,582 | **48%** | No — zero annotations |
| `models/` | 21,082 | 13% | Yes |
| `client/*` (16 sub-pkgs) | ~61,900 | 38% | Yes |
| `cmd/`, misc | ~80 | <1% | No |

`cli/` is ~half the code and entirely skippable. But 13/19 packages *are*
annotated, so at package granularity the unskippable surface is still ~52% of
LOC — package-skipping alone is bounded.

### Three levers (independently landable & measurable)

1. **`types.Config{IgnoreFuncBodies: true}`** — the scanner reads type decls,
   struct fields, and comments; it never analyzes statement bodies. Generated
   client/CLI code is overwhelmingly function bodies, which are
   *expression-dense*, so this saves **more than proportionally** to LOC and
   directly attacks `recordTypeAndValue`. Smallest diff, likely best ROI — do
   it first.
2. **Trim `types.Info`** — `recordTypeAndValue` only populates `Info.Types` if
   that map is provided. If fields/enums can be resolved from `Defs`/`Uses`
   alone, dropping the per-expression `Types` map largely evaporates that 20%.
   (Validate against the alias/enum/const paths.)
3. **AST-discovery + selective package type-checking** — skip the unannotated
   packages (`cli/`) entirely. The big structural change; do it last, once 1–2
   have de-risked the resolution semantics.

## 3. Gross gain estimate — TIME

- Time floor = **parse-all** (must read every file's comments). Unavoidable;
  caps the time win.
- Package-skipping alone: ~1.6–1.8×. Full scan ~4.4 s → ~2.6 s.
- Combined (skip + `IgnoreFuncBodies` + trimmed `Info`): ~3–4×. Full scan
  plausibly ~4.4 s → **~1.2–1.5 s**.

**Realistic: ½–⅓ of original time.** Not a screamer, not bad. At that point the
parser becomes the new ~10–15% tail and grammar work starts to matter to the
total.

Corpus-dependent: dockerctl is lucky (half is unannotated `cli/`). A typical
generated *server* (handlers + models, nearly all annotated) has little to skip
→ package-skipping ~1.2×. The `IgnoreFuncBodies` / Info-trim levers are the ones
that **generalize** — every corpus has fat function bodies.

## 4. Gross gain estimate — MEMORY (the bigger prize)

Memory has **more headroom than time**, because the two have different floors:

- Time floor = parse-all (CPU you pay regardless).
- Memory floor = the **retained** set. Parsing is *transient* — parse a file,
  extract annotations + a lean descriptor, then **free the AST**. The current
  design can't: `go/packages` pins every AST, every `types.Info` map, and the
  whole type graph alive for the entire scan. That retention *is* the 1 GB.

So a "**pull-as-you-go, cache, and free-as-you-go**" design attacks retention —
the thing that dominates memory — and can beat the time multiplier.

Decisive fork on what "pull a type" means:

1. **Lazy `go/types` per package, cache its `Info`** — still materializes a
   whole package's heavyweight objects + expression `Info` on first touch. Saves
   only fully-untouched packages. **Memory ≈ time ≈ ~½×.**
2. **Lean AST-derived type model, fall back to `go/types` only for hard cases**
   — common cases (basic type, local named type, well-known format) hold a slim
   descriptor we control; never instantiate `types.Named`/`Struct`/`Signature`
   or per-expression `Info`. **Memory ~⅓ to ⅕× — beats the time win.** This is
   the variant to build.

Favorable for go-swagger corpora:
- **Well-known formats resolve by name** (`time.Time→date-time`,
  `strfmt.UUID→uuid`, …) — string match, no resolution, never pulls `strfmt`'s
  types into the graph. A large fraction of field types short-circuit.
- **Models reference models** — the reachable subgraph is fairly self-contained.

Erosion risks — the cases that force the lean model back into `go/types`, each
potentially dragging a whole package's memory in (and the **correctness-risk
hotspots**: validate against goldens):
- Embedded field promotion (underlying-type query).
- Type aliases (`RefAliases` / `TransparentAliases`).
- Cross-package field types that aren't well-known formats.
- Enum/const values needing evaluated expression info.

## 5. Bounding peak high-water (the invariant to design to)

"Bounded" doesn't mean constant — it means **decoupled from total source size
and coupled to spec complexity**:

```
peak  ≈  |reachable type cache|  +  max_over_packages( transient working set )
```

- The first term grows **monotonically** (can't safely evict — a type
  referenced by the last annotation may have been first seen early), so it's
  bounded *by the reachable set*, not capped to a constant.
- The second term is **freeable per package**.
- Neither scales with `cli/`'s unreferenced bloat — peak tracks how big the spec
  actually is, not how big the module is. That's the property to enforce.

Two consequences:

1. **The lever is per-entry weight, not eviction.** Since the cache can't be
   evicted, minimize peak by making each entry a *lean descriptor* (kind, name,
   field refs, format), not a retained `types.Named` + `Info`. "Bounded
   reachable set × small constant per entry" is how you reach ⅓–⅕ rather than ½.
2. **Two phases keep the transient term tiny.** Phase A streams packages: parse
   → extract annotation records + root-type identities → **free the AST** (peak
   = one package's AST). Phase B resolves roots transitively into the lean cache
   (peak = reachable cache + one resolution's working set). Without the split you
   risk holding everything concurrently and peak collapses back to eager.

**The trap:** a cache only wins if `reachable ⊊ whole`, strictly. If the
annotated surface transitively touches most types, the cache grows to ≈ eager
**plus** bookkeeping — strictly worse.

## 6. Measurement

The current harness captures cumulative `TotalAlloc`, which is **blind to
retention** — the exact thing the redesign optimizes. Before prototyping:

- Add a **peak high-water sampler**: a background goroutine reading
  `runtime.MemStats.HeapInuse` (or `HeapSys` / maxRSS via `getrusage`) on a
  ticker, reporting the **max** across the scan. That gives the current eager
  design's peak as the number to beat.
- Re-run it after each lever (`IgnoreFuncBodies`, package-skip, lean-model) to
  confirm peak is actually falling, not just cumulative allocs.
- Use the heap-profile **baseline diff** (`-base=load.out`) to isolate Build /
  resolution allocations from the one-time Load.

**De-risking measurement to take first:** what fraction of field types are
well-known-format / local-named (short-circuit) vs. forced into `go/types`
(embedded / alias / cross-pkg / enum)? That ratio *is* the memory multiplier.

## 7. Context & invariants to preserve

Framing that is obvious now and easy to lose by the time the redesign starts.

### The grammar was not a performance project

It was driven by **readability, correctness, and completeness** — it does a lot
more than the regexp parser, notably **diagnostics tracking** (new). The speed
win is a *byproduct* of Go's `regexp` being slow, not a design goal. Two
consequences for reading the benchmark:

- Don't regression-gate the grammar on raw parse throughput; gate it on
  correctness/diagnostics. The parser is ~6% of the scan — it is not the budget.
- Feature work on the parser (diagnostics, completeness, the provenance feature
  below) is **nearly free** against the type-resolution baseline. The x2–x3
  redesign win comes from *resolution*, which is orthogonal to parser features —
  they don't compete for the same budget.

### Invariant 1 — the comment-parse cache must survive the redesign

Parsed annotation blocks are **cached and reused** as the same type/field is
reached from multiple places while drilling down the type graph. This makes
grammar cost `O(distinct annotations)`, not `O(type-graph traversals)` — it is
*why* the parser stays invisible.

A lazy "pull-types-as-you-go" resolver **re-drills the same types repeatedly**.
If the parse cache is keyed on the traversal path rather than the comment block,
lazy resolution will re-trigger parsing and the grammar's footprint stops being
invisible. **Constraint: key the parse cache on the comment block (declaration
identity), not on the resolution path, and carry it across the lazy resolver.**

### Invariant 2 — provenance must store positions, not AST handles

A feature in parallel development tracks **mappings between originating code and
spec node** (source ↔ spec provenance). Its cost scales as `O(spec size)` (≈ one
position per emitted node), not `O(source size)` — i.e. on the *well-behaved*
axis (output complexity, not module bloat). It will be a real but **bounded**
regression. The one trap:

- If provenance stores `ast.Node` handles or raw `token.Pos` into live
  `*ast.File`s, it **pins ASTs alive** and silently defeats the free-as-you-go
  goal of §5.
- If it stores resolved `file:line:col` (or a compact interned position), it
  stays cheap and retention-friendly.

**Constraint: provenance records compact resolved positions, never live AST
node references.** Cheap to get right early, expensive to retrofit.

## Summary

| Axis | Conservative | Realistic | Why |
|------|--------------|-----------|-----|
| Time | ~½× | **⅓–½×** | parse-all is the floor |
| Memory | ~½× (lazy over go/types) | **⅓–⅕×** (lean model + free-as-you-go) | retention, not CPU, is the floor |
| Grammar micro-opt | — | ~10–15% of total | it's no longer the bottleneck |

Sequencing: `IgnoreFuncBodies` → trim `types.Info` → AST-discovery + selective
checking → lean-model lazy resolver with bounded reachable cache. Land each
independently, re-baseline goldens between them (the alias/enum/embedded paths
are where correctness breaks), and measure **peak**, not cumulative.
