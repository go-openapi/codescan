---
title: Pruning unused models
weight: 30
description: |
  Scan a shared library with swagger:model discovery, then keep only the
  definitions actually reachable from your API — the middle ground between
  "only what routes use" and "every model, used or not".
---

`Options.ScanModels` (the `-m` flag) publishes **every** `swagger:model` type it
finds, whether or not anything references it — see
[When the scanner emits a type]({{% relref "type-discovery" %}}).
That is exactly what you want when the annotated package *is* the contract. It is
the wrong default when you point codescan at a large **shared model library** and
only care about the slice your API actually exposes: the spec fills up with
definitions no operation, parameter or response ever references.

`Options.PruneUnusedModels` is the middle ground. It runs `swagger:model`
discovery as usual, then drops every discovered definition that is not
reachable from your API surface.

## Three emission modes

The same source renders three ways, depending on two options:

| Mode | Options | What is emitted |
|---|---|---|
| **Reachable only** | *(default)* | Only models reachable from an operation, parameter or response — discovery-driven. A `swagger:model` that nothing references is **not** emitted. |
| **Every model** | `ScanModels` | Every `swagger:model` type, reachable or not. The library's whole annotated surface lands in `definitions`. |
| **Models, then pruned** | `ScanModels` + `PruneUnusedModels` | Discovery runs as in *Every model*, then the unreachable definitions are pruned away — you keep the reachable subset, including models discovered only because `swagger:model` published them. |

`PruneUnusedModels` is a **modifier on `ScanModels`**. Without `ScanModels` the
emitted set is already reachable-only, so the flag has nothing to do: it is a
no-op and says so with a single informational diagnostic.

## What counts as reachable

A definition survives the prune when it is reachable — directly or transitively
through any `$ref` — from one of these **roots**:

- an operation's body parameters and response schemas;
- a top-level shared `response` or `parameter`;
- a definition supplied via [`InputSpec`]({{% relref "overlaying-a-spec" %}}).

The walk follows references through every schema shape — properties, `allOf` /
`anyOf` / `oneOf`, array items, `additionalProperties`, and so on — and
terminates cleanly on recursive or cyclic models. A model referenced **only by
another unreferenced model** is itself unreachable, so the whole dead subtree is
removed, not just its entry point.

Those shared `response` / `parameter` roots are pruned too. A
[shared parameter or response]({{% relref "/tutorials/sharing-parameters-and-responses" %}})
that **no operation and no path-item references** is itself dropped (with a
`scan.pruned-unused` Hint) — and because that happens *before* the definition
walk reads its roots from the same `#/parameters` / `#/responses` maps, a model
kept alive only by a now-pruned shared object becomes prunable in turn.
Shared objects supplied through `InputSpec` are pinned, exactly like
definitions.

{{% notice style="info" %}}
Definitions you supply through `InputSpec` are **pinned**: they are never pruned,
and they seed the reachability roots, so anything they `$ref` survives too. The
prune only ever removes definitions codescan *discovered*, never ones you handed
it.
{{% /notice %}}

## Pruning happens before name resolution

This is the part that makes pruning more than a convenience. codescan keys every
definition by a compiler-unique identity while it builds, then a final stage
projects each one back to the shortest unique name — deconflicting cross-package
collisions along the way (`billing.Account` / `identity.Account` →
`BillingAccount` / `IdentityAccount`; see
[Resolving $ref name conflicts]({{% relref "resolving-name-conflicts" %}})).

`PruneUnusedModels` runs **before** that name-resolution stage. So when one half
of a colliding pair is unused, it is pruned *first* — and the collision never
happens. The surviving model keeps its clean, unqualified name instead of being
pushed to a package-qualified one to avoid a twin that is not even in your spec.
Pruning a shared library this way removes a whole class of surprising
`#/definitions/<Pkg><Name>` renames that only existed because of models you were
not using.

## Diagnostics

The prune is never silent. Through the
[`OnDiagnostic`](https://pkg.go.dev/github.com/go-openapi/codescan#Options) sink
codescan reports:

- `scan.pruned-unused` — one informational diagnostic per pruned definition,
  located at the originating Go type, so you can see exactly what was dropped and
  why; and the single no-op notice when the flag is set without `ScanModels`.
- `scan.renamed-definition` — one per collision the name stage *did* resolve,
  located at the Go type, recording the final name it landed under. With pruning
  on, collisions that vanish produce no such diagnostic at all.

## When to use it

- **Reach for `PruneUnusedModels`** when you scan a shared or third-party model
  package with `-m` and want only the definitions your API actually exposes —
  the reachable subset, with the noise dropped and the collision churn gone.
- **Stay on plain `ScanModels`** when the annotated package is itself the
  published contract and every `swagger:model` is meant to appear.
- **Stay on the default** (neither flag) when you only ever want what your routes
  reference; there is nothing extra to discover or prune.

## What's next

- [When the scanner emits a type]({{% relref "type-discovery" %}}) —
  reachability and `swagger:model`, the rules pruning builds on.
- [Resolving $ref name conflicts]({{% relref "resolving-name-conflicts" %}}) —
  the name-resolution stage pruning runs ahead of.
- [Overlaying a spec]({{% relref "overlaying-a-spec" %}}) —
  `InputSpec`, whose definitions are pinned against the prune.
