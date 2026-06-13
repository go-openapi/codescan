---
title: About
weight: 5
description: |
  What codescan is, why you would scan source to produce a spec, and how it
  relates to the go-swagger toolkit.
---

`codescan` is a **code-first** OpenAPI engine: it reads specially formatted
comments (annotations) in your Go source and produces a
[Swagger 2.0][swagger2] specification. It works entirely at the AST /
`go/types` level — it never compiles or runs the code it scans.

## Two ways to build an API

APIs and their documentation tend to evolve along one of two paths. The
go-openapi / go-swagger toolkit supports both.

- **Design-first** (contract-first) — you write the OpenAPI document first and
  treat it as the contract, then generate servers and clients from it. If this
  is your workflow, reach for [go-swagger][go-swagger] (`swagger generate
  server` / `swagger generate client`).
- **Code-first** — you write annotated Go and scan it to produce the spec. This
  keeps the document in sync with the code as it changes, and lets you produce a
  valid specification for a service that already exists.

**codescan is the engine for the code-first path.**

## Relationship to go-swagger

codescan began life as a single package inside [go-swagger][go-swagger] and was
spun out into its own [go-openapi][go-openapi] repository. It is the scanner
**behind** the go-swagger command:

```sh
swagger generate spec ./...
```

go-swagger remains the main command-line consumer of this library. This site
documents the **scanner library itself** — the layer beneath `swagger generate
spec` — so it sits upstream of go-swagger's "generate spec" documentation. If
you arrived here from go-swagger: the annotations are exactly the same, and you
can either keep using the `swagger` CLI or call `codescan.Run` directly from
your own program (see [Getting started]({{% relref "/getting-started" %}})).

## Why scan from source

- **One source of truth.** The spec is derived from the code, so it cannot
  silently drift from what the service actually exposes.
- **Fast iteration.** Add a field, add its annotation, regenerate — no separate
  document to keep in step by hand.
- **Document what exists.** Produce a standards-compliant spec for an API server
  that is already deployed, so it becomes interoperable with new clients and
  tooling.

When document-level metadata (info, security, servers) is more naturally
hand-authored, you do not have to push it into the code: scan the code for the
operations and models, and **overlay** the result onto a hand-written base
document (see [Shaping the output → Overlaying a spec]({{% relref "/shaping-the-output/overlaying-a-spec" %}})).

## A community toolkit

go-openapi and go-swagger are community-driven, open-source **building blocks**
meant to be assembled and customized — there are too many ways to approach APIs
to cover them all. Fork, reuse, and adapt what you find useful. See the
[go-swagger project's "About" page][go-swagger-about] for the wider toolkit
story.

## Where to go next

{{< cards >}}
{{% card title="Getting started" %}}
Install codescan and produce your first spec.

→ [getting-started]({{% relref "/getting-started" %}})
{{% /card %}}

{{% card title="Tutorials" %}}
Learn by spec concept, with annotated Go beside the spec it produces.

→ [tutorials]({{% relref "/tutorials" %}})
{{% /card %}}
{{< /cards >}}

[swagger2]: https://swagger.io/specification/v2/
[go-swagger]: https://github.com/go-swagger/go-swagger
[go-swagger-about]: https://goswagger.io/
[go-openapi]: https://github.com/go-openapi
