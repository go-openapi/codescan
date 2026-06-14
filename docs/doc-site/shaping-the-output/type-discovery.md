---
title: When the scanner emits a type
weight: 15
description: |
  codescan never invents definitions — a type appears only when it is reachable
  or registered. Understand reachability and swagger:model so nothing goes
  missing or appears unexpectedly.
---

codescan does not emit a definition for every type it can see. A named type
reaches the spec when **either** of these holds:

- it is **reachable** — referenced (directly or transitively) from an operation,
  parameter, response, or another emitted model; or
- it is **registered** — annotated `swagger:model`, which (with
  `Options.ScanModels`) publishes it even when nothing references it.

A type that is neither reachable nor registered is simply absent — the scanner
never invents it. The package below has one of each case:

{{< code file="shaping/discovery/discovery.go" lang="go" region="types" >}}

Scanned with `ScanModels: true`, the definitions are:

{{< code file="shaping/discovery/testdata/definitions.json" lang="json" >}}

- **`Cart`** — a `swagger:model` root.
- **`Order`** — has **no** `swagger:model`, yet it is emitted (as a `$ref`
  target) because `Cart` references it. You do not need to annotate every nested
  type.
- **`Standalone`** — a `swagger:model` that nothing references; `ScanModels`
  publishes it anyway.
- **`Orphan`** — neither referenced nor annotated, so it never appears.

{{% notice style="info" %}}
If a model is missing from your spec, it is almost always **unreachable**: no
operation/parameter/response/model leads to it. Either reference it, or annotate
it `swagger:model` and scan with `ScanModels`.
{{% /notice %}}

## Generic and embedded types

codescan resolves types through `go/packages` type information, so two forms
that look tricky still work:

- **Generics.** An instantiated generic — `WrappedRequest[Order]`, whether
  annotated `swagger:parameters` or `swagger:model` — emits the concrete type:
  the type argument is substituted, so a `T`-typed field becomes a `$ref` to the
  argument's definition. The generic's declaration may live in a different file
  from its instantiation. A free (un-instantiated) type parameter is skipped
  with a warning.
- **Embedded fields**, including those from an **external package**, are
  promoted into the embedding type, and a custom field type resolves to its
  underlying type.
