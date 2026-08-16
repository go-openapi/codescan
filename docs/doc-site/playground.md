---
title: "Playground"
weight: 15
description: |
  Scan Go source in your browser.

  Edit annotations, watch the specification change, and follow a spec node back to the code that produced it.
---

Everything below runs in this tab. There is no server: the scanner is `codescan` itself, compiled to WebAssembly.

The source you open never leaves your browser.

{{% notice style="note" title="Experimental" %}}
Offered for demonstration.
It follows [`genspec-tui`](https://github.com/go-openapi/codescan/tree/master/cmd/genspec-tui) closely,
and its interface is checked by hand rather than by tests.
{{% /notice %}}

{{< playground >}}

## What to try

The **Examples** menu carries five modules, each one whole and each scanning as it stands:

|              | shows                                                                      |
|--------------|----------------------------------------------------------------------------|
| Models       | a struct becoming a definition: validations, an enum, an example, a `$ref` |
| Routes       | `swagger:route` with its parameters and responses                          |
| Operation    | `swagger:operation`, where you write the OpenAPI directly in YAML          |
| Enums        | a Go constant set becoming an enum, typed from the declaration             |
| Polymorphism | a discriminated base and its subtypes under `swagger:allOf`                |

Edit anything on the left and it rescans after a pause.

**Track** joins the two panes. Put the cursor on a Go line and the specification
highlights what that line produced; put it on a spec line and the source
highlights what produced it; click a diagnostic and both light up. It answers by
position rather than by matching names, which is why it survives a rename.

One direction is exact and the other is not. The scanner records where a field *starts* and not where it ends,
so a cursor sitting in a doc comment is attributed to the nearest anchor,
ties resolving downwards because Go documentation sits above what it documents.

Press <kbd>/</kbd> in the specification to search it, <kbd>n</kbd> and <kbd>N</kbd> to step through matches.
The **Swagger UI** tab renders the document as a reader of your API would see it.

## Scanning your own code

Use **Open module…**. Three things are worth knowing first, and the second is
the one nobody guesses:

1. **It has to be a module.** `go mod init example.com/api` if there is no
   `go.mod`. Import paths resolve against the module, and without one there is
   nothing to resolve them against.

2. **Vendor the dependencies** — `go mod vendor`. There is no module cache in a
   browser and nothing is downloaded, so a dependency's types can only arrive as
   source. That matters more than it looks: a library that declares things in
   its *comments*, as `strfmt` does with `swagger:strfmt`, cannot be understood
   any other way. Open a module without vendoring and the playground says so
   before it scans.

3. **Pick the folder, not the files.** The whole directory goes in at once. Test
   files are skipped, `vendor/` is kept, and the tree is re-rooted on the
   outermost `go.mod` — so it does not matter whether you pick the module, its
   parent, or a directory inside it.

The status line reports what each scan cost, in time and in memory. Hovering it
breaks the time into fetching the scanner, compiling it, preparing the
filesystem and scanning, which is how to tell a slow first load from a slow
scan. The scanner is fetched once and then cached, so later scans start
immediately.

## The same thing without a browser

[`genspec-wasi`](https://github.com/go-openapi/codescan/tree/master/cmd/genspec-wasi)
is the command this page is built around.
It writes a specification to standard output,
and `-format=json` wraps it with the diagnostics and cross-references that drive the two panes above.

```sh
go install github.com/go-openapi/codescan/cmd/genspec-wasi@latest
genspec-wasi -workdir ./my-api ./...
```

For an interactive scan in a terminal, with the same tracking and the same diagnostics,
use [`genspec-tui`](https://github.com/go-openapi/codescan/tree/master/cmd/genspec-tui).
