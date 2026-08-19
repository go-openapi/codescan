<!--
SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
SPDX-License-Identifier: Apache-2.0
-->

# genspec-wasi

The codescan playground: a Svelte front-end around the `wasip1` artifact, so a page can scan Go
source without a server, a toolchain, or an upload. The files never leave the browser.

Destined for the doc site as a static bundle; [`../hugo`](../hugo) will embed it through a shortcode.
For the conformance probe that established this works at all, see
[`hack/browser`](../../browser/README.md).

**Experimental, and offered for demonstration.** It works end to end and follows
[`cmd/genspec-tui`](../../../cmd/genspec-tui/README.md) closely enough to be judged against it:
syntax highlighting on both sides, a diagnostics gutter in both margins, cross-references in three
directions, `/` search, and a Swagger UI tab. It is not a supported product, and the section below
says what that means for how far to trust a green build.

```sh
npm install
npm run wasm      # cross-compiles cmd/genspec-wasi into public/
npm run dev       # http://localhost:5174
npm run build     # dist/
npm run check     # svelte-check
npm run dist      # wasm + build + pack into ../hugo, which is what the doc site serves
```

`public/genspec-wasi.wasm`, `node_modules/` and `dist/` are generated; none is committed.

> **Node 22 or 20.19+.** `package.json` says so under `engines`, and it is worth reading as a
> requirement rather than a preference: rolldown ships its native binding as an *optional*
> dependency gated on that same range, so an older node installs everything except the binding —
> with no error — and the first `vite`/`vitest`/`svelte-check` run dies with **"Cannot find native
> binding … npm has a bug related to optional dependencies"**. It is not an npm bug and reinstalling
> will not fix it. Switch node, `rm -rf node_modules`, and install again: `npm ci` will not notice
> that the tree it is reusing was built by the wrong one.

## Using it

Both panes are editors: the left is Go, the right is the document the scan produced, read-only but
navigable. Editing rescans after a pause, and the compiled artifact is kept, so a re-scan costs an
instantiation plus the scan.

**Tracking** joins them, and is what `genspec-tui` exists for. With **Track** on:

| From | To |
|---|---|
| a source line | the spec node it produced |
| a spec line | the Go source that produced it |
| a diagnostic | the line that raised it |

The spec direction is exact — the renderer records which lines each RFC 6901 pointer occupies, and a
node with no provenance of its own resolves to its nearest anchored ancestor. The source direction is
not: the scanner anchors a *point*, so a cursor that is not on one takes the nearest anchor, tie
going downwards because Go documentation sits above what it documents.

| Key | Where | Action |
|---|---|---|
| arrows, `Home`/`End`, `PgUp`/`PgDn` | any pane | move; tracking follows |
| `/` | spec | search — case-insensitive, by line |
| `n` / `N` | spec | next / previous match, wrapping |
| `Enter` / `Shift+Enter` | find bar | same |
| `Esc` | find bar | close |
| `Enter` | diagnostics | go to the line, whether or not tracking is on |

Search puts the **cursor** on the match rather than only scrolling to it, so tracking follows a
search: look for `maxLength`, press `n`, and the source pane walks the fields that have one.

**Swagger UI** is the other tab, fetched on first activation and not before — 1.4 MB of JavaScript
and 177 KB of CSS, against a 20 MB artifact. Two of its defaults reach the network and both are
turned off: `validatorUrl` otherwise POSTs the whole document to `validator.swagger.io` to draw a
badge, and "Try it out" fires real requests at whatever host the document names. Neither is
compatible with the line in the status bar.

Swagger UI is a light interface with no dark mode, so in a dark theme it is pulled towards ours by
inverting and rotating the hue back — the pair is chosen because `hue-rotate(180deg)` undoes what
`invert()` does to hue, which keeps the method badges the colours that carry their meaning. It is a
heuristic, so the strip above the preview offers **show as published**: the untouched rendering, as a
reader of the document sees it. A light theme is left alone; there is nothing to
reconcile.

A document with no paths gets an explanation rather than an empty page: a package of annotated
models produces exactly that, and Swagger UI cannot tell it apart from a failure.

The status line reports what the last scan spent. Hovering it splits the number four ways — compile,
prepare, instantiate, scan — which is how to tell a slow first load from a slow scan. Compile is zero
on every run after the first; the scan is not, because nothing below it is incremental.

## How this is tested

**The user interface is verified by hand.** There are 104 unit tests, and they cover the pure
modules: the renderer and its pointer spans, the tracking rules, search, the file tree, the option
flags, the formatters. Not one of the seventeen components is rendered by any test.

So a green build says the logic holds. It does not say the page works — and the difference is not
theoretical. A shell that collapsed to nothing, arrow keys that did not move, an active line that was
themed but never applied, a Swagger UI that stayed white under a dark filter: every one of those
passed `svelte-check`, the unit tests and the build, and was found by looking at the page.

Automation is **deferred rather than rejected**. CI already carries codescan's golden suite, and a
chromium download does not belong on it for a demonstration. If this tier keeps growing it likely
moves to a repository of its own, dedicated to exactly this — the interface and the WASI integration
— which is where paying for a browser in CI would make sense. Whoever picks that up: the store is
drivable under vitest today and is the cheaper half to start with.

```sh
npm test           # the pure modules
npm run check      # svelte-check
```

## Shape

| | |
|---|---|
| `src/styles/tokens.css` | every value the app draws with, named by role |
| `src/styles/base.css` | the shared look, drawn only from those tokens |
| `src/worker/` | owns WebAssembly: compiles once, instantiates per scan, builds the guest filesystem |
| `src/lib/store.svelte.ts` | the state every panel reads, created per mount and passed down through context |
| `src/lib/theme.svelte.ts` | light / dark / follow-the-system, resolved to one of two |
| `src/lib/flags.ts` | renders scan options as the command line the guest sees |
| `src/lib/render.ts` | writes the document as JSON or YAML, recording the lines each pointer occupies |
| `src/lib/track.ts` | joins pointer to source position, in both directions |
| `src/lib/search.ts` | line matching, and stepping through matches |
| `src/lib/editor.ts` | the CodeMirror extensions: gutter, decorations, theme |
| `src/components/Editor.svelte` | the editor both panes are built from |
| `src/components/SwaggerPreview.svelte` | the Swagger UI tab, loaded on first use |
| `src/lib/tree.ts` | re-roots a picked directory, and decides what is worth carrying into the guest |
| `src/lib/sample.ts` | what the playground opens with; a shortcode will replace it |
| `src/components/SplitPane.svelte` | the resizable split, draggable and arrow-key adjustable |
| `src/components/Tabs.svelte` | the tablist, to the WAI-ARIA pattern |
| `src/components/` | toolbar, options, file picker, the two panes, diagnostics, status |

## Theming, and being somebody else's component

Every token is declared inside a `@layer`, scoped to `.cs-root` rather than `:root`.
That is the whole embedding contract: unlayered declarations beat layered ones
regardless of specificity, so a host page that sets `--cs-accent` on any ancestor
wins without `!important` — and without us having to guess what the doc site calls
its own variables. Anything it does not override keeps our value.

There is no `prefers-color-scheme` block. `theme.svelte.ts` resolves "follow the
system" in JavaScript and always stamps `data-theme`, so each theme is stated once
instead of the dark one twice — once for "the system says so" and once for "you
asked for it" — which is how the two drift apart.

The state is created per mount and passed down through context rather than being a
module singleton. One playground per page is the shape the doc site wants, but a
singleton makes a second mount silently share the first one's tree.

The scan runs in a **worker**. A large tree is seconds of solid CPU, and the WASI shim implements
`poll_oneoff` by spinning rather than yielding — on the main thread either would stall the page.

Each scan needs a **fresh instance**: the artifact is a WASI command that ends at `proc_exit`, and
Go's `wasip1` target emits no reactor form. Compiling is the expensive half and is kept between
runs, so a re-scan costs a few milliseconds of instantiation.

## Opening a module

**Open module…** replaces the tree; it does not merge into it. Merging looks harmless and is not: the
sample, or whatever was loaded before, stays behind, and the next scan sees two modules at once with
two `go.mod` files.

A directory pick reports paths relative to the chosen folder, which may sit above the module — or
below it. The store re-roots on the outermost `go.mod` and drops what falls outside, so it does not
matter which level was picked. Test files are skipped, and a selection over 8 MB of Go source is
refused rather than held in memory — which a vendored tree of any size will meet, see the note below.

**Vendor first.** `go mod vendor` before opening a real project. There is no module cache in the
guest, so a vendored tree is the only way a third-party import resolves at all — and the only way one
whose meaning lives in comments resolves *correctly*, since export data holds types and not comments.

`vendor/modules.txt` is carried along with the vendored source, and is the easiest file here to lose:
it is the only one that is not Go, and the loader treats a vendor directory as authoritative only when
it can read it. Dropping it does not fail the scan — the whole vendored tree is ignored and every
dependency is synthesized instead, which surfaces as a wall of `scan.synthesized-import` warnings
pointing nowhere near the mistake. `lib/tree.ts` keeps it deliberately, and a test says so.

## Weight

A visitor downloads about **24 KB** of application and **8.4 MB** of artifact, both compressed. The
artifact dominates by more than two orders of magnitude, which is worth remembering before optimising
anything on the JavaScript side.

Most of that is the standard library's types: the artifact alone is 3.7 MB, and carrying its own copy
of the export data takes it to 8.4 MB. Mounting the archive into the guest filesystem instead
would keep the two independently cacheable — `argvFor(..., 'mounted')` and the reserved
`public/exportdata.zip` are that variant, unused for now.

## The standard library

The artifact carries the standard library's **export data** inside it, put there by the `exportdata`
build tag — which is why `npm run wasm` regenerates the archive before building, and why the worker
names no standard-library flag at all: the command finds its own copy.

That costs no fidelity. The types are the compiler's own, so method sets and interface identity
survive, and a scan with nothing mounted but the module raises no `scan.synthesized-import`.

`-stub-stdlib`, which synthesizes the standard library from the names the code selects through it, is
still reachable (`argvFor(..., 'stub')`) and is the degraded mode rather than a choice: a synthesized
type has no fields and no method set.

Export data cannot cover a package whose meaning lives in **comments** — `strfmt` declares its
formats with `swagger:strfmt`, and export data holds types, not comments — or any third-party import
that is neither vendored nor cached. Both resolve by reaching the guest as source; closing that gap is
the loader's job, not the front-end's.
