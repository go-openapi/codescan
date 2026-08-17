---
title: "Options reference"
weight: 20
description: |
  Every option codescan takes - Go field, command-line flag, configuration key.
---

[`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options)
is the single configuration struct passed to [`codescan.Run`](https://pkg.go.dev/github.com/go-openapi/codescan#Run).

**The zero value is a valid configuration** — every flag defaults to `false`, every slice/map to `nil`,
every numeric tunable to its built-in default. You set only what you need.

This page is the field-by-field catalogue, and serves the commands as much as the
library: **Flag** is what to type, and **Section** is where a `.codescan.yaml`
addresses it — the key inside that section is the flag without its dash. See
[Setting an option]({{% relref "setting-options" %}}) for the rules those two
columns follow, and for the handful of options that are nobody's flag.

The [godoc](https://pkg.go.dev/github.com/go-openapi/codescan#Options) is the
normative source; each field here links to the
[how-to guide]({{% relref "/shaping-the-output" %}}) that shows it on real
input where one exists.

{{% notice style="note" %}}
**Default** is the library's. The commands agree with it everywhere but one:
`-scan-models` defaults to `true`, because a command asked for a specification
and handed a package of annotated models should produce their definitions.
{{% /notice %}}

{{% notice style="note" %}}
**Config Section** is where a `.codescan.yaml` addresses the option. A dash means
it has none: either the option is not a value a file can carry (a callback, a
positional argument), or it names a **path**, which is settable on the command
line only. See [what a file may not set]({{% relref "setting-options" %}}#what-a-file-may-not-set).
{{% /notice %}}

{{% notice style="note" %}}
codescan never writes to stdout or stderr. Every scan-time observation — a
dropped construct, a rename, a prune — flows through the `OnDiagnostic`
callback. See [Diagnostics & observability](#diagnostics--observability) below.
{{% /notice %}}

## Inputs & scope

What gets loaded and which packages and types are in play. See
[Scope & discovery]({{% relref "/shaping-the-output/scope-and-discovery" %}}).

| Option | Type | Default | Flag | Config Section | Effect |
|--------|------|---------|------|---------|--------|
| `Packages` | `[]string` | `nil` | *(positional)* | — | Package patterns to scan (e.g. `./...`), resolved relative to `WorkDir`. |
| `WorkDir` | `string` | `""` (cwd) | `-workdir` | — | Working directory the package patterns and module resolution are rooted at. **Command line only**: see [what a file may not set]({{% relref "setting-options" %}}#what-a-file-may-not-set). |
| `BuildTags` | `string` | `""` | `-build-tags` | `scan` | Go build tags to activate while loading, so tag-guarded source is scanned. See [Build tags]({{% relref "build-tags" %}}). |
| `Include` | `[]string` | `nil` | `-include` | `scan` | Allow-list of package path patterns; when non-empty only matching packages are scanned. See [Scoping the scan]({{% relref "scoping-the-scan" %}}). |
| `Exclude` | `[]string` | `nil` | `-exclude` | `scan` | Deny-list of package path patterns, applied after `Include`. See [Scoping the scan]({{% relref "scoping-the-scan" %}}). |
| `IncludeTags` | `[]string` | `nil` | `-include-tags` | `scan` | Allow-list filtering routes/operations by their swagger `tags`. |
| `ExcludeTags` | `[]string` | `nil` | `-exclude-tags` | `scan` | Deny-list filtering routes/operations by their swagger `tags`. |
| `ExcludeDeps` | `bool` | `false` | `-exclude-deps` | `scan` | Skip types reached through module dependencies, keeping the scan to first-party packages. |
| `ScanModels` | `bool` | `false` | `-scan-models` | `emit` | Also emit a definition for every `swagger:model` type, not just route-reachable ones. See [When the scanner emits a type]({{% relref "type-discovery" %}}). |
| `PruneUnusedModels` | `bool` | `false` | `-prune-unused-models` | `emit` | With `ScanModels`, drop discovered definitions not transitively reachable from a path, shared response/parameter, or `InputSpec` root. Runs before name reduction; `InputSpec` definitions are pinned. No-op without `ScanModels`. See [Pruning unused models]({{% relref "pruning-unused-models" %}}). |
| `InputSpec` | `*spec.Swagger` | `nil` | `-input` *(genspec)* | `document` | Base document to overlay scanned discoveries onto; its definitions are pinned and seed pruning roots. See [Overlaying a spec]({{% relref "overlaying-a-spec" %}}). |

## Loading & the go environment

Where the package graph comes from, and which platform it is built for. These
change the emitted spec the way `BuildTags` does — by deciding which files each
package is made of — or change what the scan needs in order to run at all.

They are options rather than inherited process state so that a scan is
reproducible: a value picked up from whatever shell started it is easy to apply on
one code path and forget on another.

### Which loader, and why

Three ways to get a package graph. The table below catalogues the fields; this is
how to choose between them.

**Standard loader** — the default. Loads your code with the Go toolchain, through
`golang.org/x/tools/go/packages`. Maintained by the Go team, and the reference for
how patterns and imports resolve: where either of the others disagrees with it, the
other one is wrong. Requires Go installed, and uses the build cache.

**Pure-Go loader** (`ToolchainFreeLoader`) — loads your code with codescan's own
reimplementation. Cuts memory by roughly 45%, and needs no `go` command and no
subprocess. It still reads `GOROOT/src` for the standard library, so it wants a Go
*installation* — just not a runnable toolchain. Modules only. It uses no build
cache, so cold costs what warm costs: about level with the standard loader on a
warm cache, roughly 30% faster on a cold one, and the only choice whose cost does
not depend on cache state. Usually the right pick for CI.

**Compiled dependencies** (`CompiledDependencies`) — the standard loader taking
dependency types from the compiler's export data instead of reading their source. It
produces the same document either way — whatever the spec needs out of a dependency
is read at the moment it is needed — and on a warm build cache it is the fastest by
a wide margin, and several times smaller.

It must *compile* the dependency closure rather than type-check it, so on a cold
cache it is an order of magnitude slower and writes a large build cache. Reach for
it where the cache is warm by construction — your own machine, a watch loop, a
pipeline that restores its cache — and leave it off where a clean checkout is the
norm, which is what a CI runner usually is. Code that does not compile is **not** a
reason to avoid it: such a load is retried from source automatically.

Two further options drop the `GOROOT` requirement altogether, for environments with
no Go installation at all — a WASI guest, a browser. `StubStdlib` synthesizes the
standard library, and pays for the reach in fidelity: a fabricated type has the
right name and no structure. `ExportData` serves dependencies from a blob you
prepare in advance, and pays in preparation instead — the types are the compiler's
own, but the blob is only valid for the toolchain that produced it, and a package
it does not cover falls back to source and then to synthesis.

{{% notice style="note" %}}
The percentages are indicative, not a promise: the balance moves with the size of
the tree being scanned, and on a small one the pure-Go loader is *slower* warm than
the standard loader. The figures, the corpora they were taken on and the method are in
[`internal/benchmarks`](https://github.com/go-openapi/codescan/tree/master/internal/benchmarks),
whose harness also takes an extra corpus of your own to measure alongside them.
{{% /notice %}}

| Option | Type | Default | Flag | Section | Effect |
|--------|------|---------|------|---------|--------|
| `GOOS` / `GOARCH` | `string` | `""` (this machine) | `-goos` / `-goarch` | `go` | The platform the scanned code is built for. `//go:build` lines and `_linux.go` / `_amd64.go` filename suffixes resolve against them, so they select which files a package is made of. |
| `GOFLAGS` | `string` | `""` (process env) | `-goflags` | `go` | Default go command flags, e.g. `-tags=integration`. Flags given through `BuildTags` win, as they do for the go command. |
| `GOWORK` | `string` | `""` (search upwards) | `-gowork` | `go` | Workspace selection: `off` disables it, a path names a `go.work`. Inside a workspace a sibling module resolves to the copy being worked on rather than to the module cache — miss that and its types are read stale, or synthesized empty. |
| `GOEXPERIMENT` | `string` | `""` (process env) | `-goexperiment` | `go` | Toolchain experiments, e.g. `jsonv2`; each contributes a `goexperiment.<name>` build tag. |
| `ToolchainFreeLoader` | `bool` | `false` | `-loader=own` | `load` | Resolve the package graph with codescan's own loader instead of `golang.org/x/tools/go/packages`. Same job and, across the fixture corpus, the same spec; it differs in needing no installed toolchain and no subprocess, since it never runs `go list`. **Experimental.** |
| `FS` | `fs.FS` | `nil` | — | — | Read source through a virtual filesystem — an in-memory tree, an uploaded archive, an `embed.FS` — instead of the real one. Implies `ToolchainFreeLoader`, since `go list` can only read the real filesystem. **`FS` is the whole world the scan can read**: dependencies and GOROOT come through it too, absolute paths map by dropping the leading separator, and anything unreachable is synthesized — a valid but quietly thinner spec, announced by `scan.synthesized-import` and `scan.degraded-load`. **Experimental.** |
| `StubStdlib` | `bool` | `false` | `-stub-stdlib` | `load` | Synthesize the standard library from the names the code selects, rather than reading GOROOT. Toolchain-free loader only. Identity recognition still works (`time.Time`, `json.RawMessage` are matched on package and name), but a synthesized type has no fields and no method set — so `json.RawMessage` stops rendering as a byte array and nothing is seen to implement `encoding.TextMarshaler`. Trades fidelity for reach, quietly; prefer a full graph where GOROOT is available. **Experimental.** |
| `ExportData` | `fs.FS` | `nil` | `-export-data` *(genspec-wasi)* | — | Serve dependencies from pre-computed export data (one `<import path>.export` file per package) under the toolchain-free loader. Unlike `StubStdlib` this costs no fidelity, the types being the ones the compiler computed — but it is valid only for the toolchain that produced it, and an uncovered package falls back to source, then to synthesis. The module under scan is never read this way, and neither is a dependency whose source carries annotations or one the spec later needs a declaration from. **Experimental.** |
| `CompiledDependencies` | `bool` | `false` | `-compiled-dependencies` | `load` | Take dependency types from the compiler's export data instead of reading every dependency from source, under the go/packages loader. It costs no meaning: a dependency whose source carries annotations is read back after the load, and one that merely *declares* a type the spec carries is read at the lookup that wants it — so a `swagger:strfmt` written in a library still counts, and a model declared in an unannotated dependency keeps its doc comment and its fields. Set it for cost alone, and only where the build cache is warm: it is markedly faster warm and markedly slower cold, since `go list -export` compiles the closure before it can read it. A closure that does not compile is handled either way — the load falls back to source and raises `scan.compiled-dependencies`. |

{{% notice style="note" %}}
The virtual-filesystem and export-data options exist to make a scan possible
where no Go toolchain is: they are what lets codescan run compiled to
WebAssembly. See the [Playground]({{% relref "playground" %}}).
{{% /notice %}}

## Names & references

How definitions are named and how `$ref`s render. See
[Names & `$ref`s]({{% relref "/shaping-the-output/names-and-refs" %}}).

| Option | Type | Default | Flag | Section | Effect |
|--------|------|---------|------|---------|--------|
| `NameFromTags` | `[]string` | `nil` (⇒ `["json"]`) | `-name-from-tags` | `emit` | Ordered struct-tag types a property/parameter/header name is derived from; first that supplies a name wins. Explicit empty slice ⇒ Go field name. Only the name — `json` encoding directives (`-`, `,omitempty`, `,string`) always come from `json`. See [Naming from struct tags]({{% relref "naming-from-tags" %}}). |
| `SkipJSONifyInterfaceMethods` | `bool` | `false` | `-skip-jsonify-interface-methods` | `emit` | Emit interface-method property names verbatim (`ID`, `CreatedAt`) instead of auto-jsonifying them (`id`, `createdAt`). Only affects interface methods; struct fields and `swagger:name` overrides are unchanged. See [Interface-method property names]({{% relref "interface-method-names" %}}). |
| `RefAliases` | `bool` | `false` | `-ref-aliases` | `emit` | Render Go type aliases as a first-class `$ref` (via `swagger:model`) instead of expanding them inline. See [Alias rendering]({{% relref "alias-rendering" %}}). |
| `TransparentAliases` | `bool` | `false` | `-transparent-aliases` | `emit` | Make aliases fully transparent — never creating a definition. See [Alias rendering]({{% relref "alias-rendering" %}}). |
| `DefaultAllOfForEmbeds` | `bool` | `false` | `-default-all-of-for-embeds` | `emit` | Render a plain (untagged, unnamed) struct embed as an `allOf` member — a `$ref` for a model embed, an inline member otherwise — with the embedding struct's own fields in a sibling member, instead of inlining promoted properties. json-named embeds, `swagger:allOf` embeds, and interface embeds are unaffected. See [Composing embeds with allOf]({{% relref "composing-embeds-with-allof" %}}). |
| `NameConcatBudget` | `float64` | `0` (⇒ `0.65`) | `-name-concat-budget` | `emit` | Readability cutoff `[0,1]` for the package-segment concatenation that deconflicts colliding definition names; lower scores are more readable. A group whose best concat scores above the budget is a candidate for the hierarchical fallback. See [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}). |
| `EmitHierarchicalNames` | `bool` | `false` | `-emit-hierarchical-names` | `emit` | For the rare collision group whose best flat concat exceeds `NameConcatBudget`, emit nested container definitions (`#/definitions/<pkg>/<Name>`) instead of a long flat concat, with an explanatory diagnostic. The always-correct flat concat is the default. See [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}). |
| `EmitRefSiblings` | `bool` | `false` | `-emit-ref-siblings` | `emit` | Emit a `$ref`'d field's description and vendor extensions as direct `$ref` siblings (`{$ref, description, x-*}`) instead of an `allOf` wrap. Validations/externalDocs still force a compound. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |
| `SkipAllOfCompounding` | `bool` | `false` | `-skip-all-of-compounding` | `emit` | Never emit an `allOf` compound for a `$ref`'d field. Validations/externalDocs are dropped (description/extensions too, unless `EmitRefSiblings` keeps them as siblings); each drop raises a diagnostic. `required` is unaffected. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |
| `DescWithRef` | `bool` | `false` | — | — | **Deprecated** — prefer `EmitRefSiblings`. In the description-only case, wrap the `$ref` in a single-arm `allOf` to preserve the description (strict draft-4 shape). No-op when `EmitRefSiblings` is set. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |

## Titles & descriptions

The human-readable text the spec carries. See
[Titles & descriptions]({{% relref "/shaping-the-output/titles-and-descriptions" %}}).

| Option | Type | Default | Flag | Section | Effect |
|--------|------|---------|------|---------|--------|
| `SingleLineCommentAsDescription` | `bool` | `false` | `-single-line-comment-as-description` | `emit` | Route every single-line doc comment to `description`, never to `title`/`summary` (the first-sentence convention otherwise applies). Multi-line comments keep the title/description split. See [Single-line comments as descriptions]({{% relref "single-line-comments" %}}). |
| `AfterDeclComments` | `bool` | `false` | `-after-decl-comments` | `emit` | Let swagger annotations live inside a struct body or as a trailing comment, in addition to the doc comment above the declaration, so the godoc stays clean. v0.36 scope: type declarations (struct inside-body + alias trailing comment). See [Keeping annotations out of the godoc]({{% relref "keeping-annotations-out-of-the-godoc" %}}). |
| `CleanGoDoc` | `bool` | `false` | `-clean-go-doc` | `emit` | Strip godoc doc-link brackets from generated `title`/`description` (humanizing unresolved ones, dropping reference-definition lines, recomposing resolved links to each schema's exposed name). Applies only to godoc-derived prose; overrides are untouched. See [Cleaning godoc doc-links]({{% relref "cleaning-godoc-doc-links" %}}). |

## Field types, formats & extensions

How an individual property renders. See
[Field types & formats]({{% relref "/shaping-the-output/field-types-and-formats" %}}).

| Option | Type | Default | Flag | Section | Effect |
|--------|------|---------|------|---------|--------|
| `SetXNullableForPointers` | `bool` | `false` | `-set-x-nullable-for-pointers` | `emit` | Emit `x-nullable: true` on pointer-typed fields. See [Nullable pointers]({{% relref "nullable-pointers" %}}). |
| `SkipExtensions` | `bool` | `false` | `-skip-extensions` | `emit` | Suppress all `x-go-*` vendor extensions in the output. See [Vendor extensions]({{% relref "vendor-extensions" %}}). |
| `EmitXGoType` | `bool` | `false` | `-emit-x-go-type` | `emit` | Stamp an `x-go-type` extension (fully-qualified originating Go type) on every emitted definition, for round-tripping a spec back to its Go types. Suppressed under `SkipExtensions`. See [Vendor extensions]({{% relref "vendor-extensions" %}}). |
| `SkipEnumDescriptions` | `bool` | `false` | `-skip-enum-descriptions` | `emit` | Keep the per-enum-value const-name mapping (from `swagger:enum`) out of the `description`, exposing it only via the `x-go-enum-desc` extension. Suppressed entirely under `SkipExtensions`. |

## Diagnostics & observability

Channels for what the scan observed; these do not change the output spec.

| Option | Type | Default | Flag | Section | Effect |
|--------|------|---------|------|---------|--------|
| `OnDiagnostic` | `func(Diagnostic)` | `nil` | — | — | Invoked once per diagnostic in source order (parser warnings, validation failures, prunes, renames). Diagnostics never block the build — invalid constructs are dropped from the spec while their explanation flows here. The only output channel. **Experimental** while LSP integration matures. |
| `OnProvenance` | `func(Provenance)` | `nil` | — | — | Invoked once per anchor node in the produced spec, carrying its JSON pointer and the source position of the Go construct that produced it. Never blocks the build. **Experimental** while LSP/TUI integration matures. |
| `Debug` | `bool` | `false` | — | — | **Deprecated, ignored.** The legacy stderr debug logger was retired; wire `OnDiagnostic` instead. Retained for API compatibility. |

The two callbacks are how the commands report: `genspec` renders diagnostics on
standard error, and `genspec-wasi -format=json` carries both in its envelope for
a program to read.

## See also

- [Setting an option]({{% relref "setting-options" %}}) — the flag and key rules
  the two middle columns follow, and which spelling wins.
- [Annotations]({{% relref "annotations" %}}) — the `swagger:*` vocabulary the
  scanner reads from comments.
- [Keyword reference]({{% relref "keywords" %}}) — the `keyword: value` forms
  inside annotation bodies.
- [Shaping the output]({{% relref "/shaping-the-output" %}}) — task-oriented
  how-tos that put these options to work on real input.
