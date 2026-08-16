---
title: Alias rendering
weight: 30
description: |
  Choose how Go type aliases render — dissolved to their target, or exposed as a
  first-class $ref via swagger:model, with RefAliases / TransparentAliases.
---

A Go type alias (`type Price = Money`) is, to the Go type system, *literally* the
same type as its target. codescan's default is to treat it that way: at a use
site the alias **dissolves** to its target, producing no definition of its own.

{{< example go="shaping/aliases/aliases.go" goregion="alias"
            json="shaping/aliases/testdata/invoice.json" golabel="Annotated Go" jsonlabel="#/definitions/Invoice" >}}

`Invoice.total` is typed `Price`, but the field resolves straight to
`#/definitions/Money` — `Price` itself never appears.

## Exposing an alias as a first-class entity

This is an **advanced, rarely-needed** case. To keep the alias name in the spec —
its own definition that other schemas `$ref` — annotate the alias with
`swagger:model`:

{{< code file="shaping/aliases-firstclass/firstclass.go" lang="go" region="firstclass" >}}

Two top-level options then govern how that first-class alias *definition* is
shaped. The panes below are the same package scanned under each.

### Default — the alias definition is a copy

`Fee` is emitted as a structural duplicate of `Amount`, and `Receipt.charge`
points at the alias:

{{< compare
    left="shaping/aliases-firstclass/testdata/expand.json"  leftlabel="Default (expand)"
    right="shaping/aliases-firstclass/testdata/refaliases.json" rightlabel="RefAliases: true" >}}

### `RefAliases: true` — the alias definition is a `$ref` chain

The right pane above: `Fee` becomes `{"$ref": "#/definitions/Amount"}`. One
shape, two names — the alias survives at use sites without duplicating the
target's properties. Prefer this over the default whenever the alias is
genuinely a synonym: a copy drifts the moment the target changes.

### `TransparentAliases: true` — use sites dissolve

{{< code file="shaping/aliases-firstclass/testdata/transparent.json" lang="json" >}}

`Receipt.charge` now points straight at `#/definitions/Amount`: the alias is
gone from the reference graph.

{{% notice style="warning" %}}
Note what did **not** happen: `Fee` is still emitted. `TransparentAliases`
governs how an alias renders at its *use sites*, not whether an annotated
declaration produces a definition — so with `ScanModels` you get a `Fee`
definition that nothing references. Add
[`PruneUnusedModels`]({{% relref "pruning-unused-models" %}}) to drop it, or
simply do not annotate an alias you intend to dissolve.
{{% /notice %}}

The three modes at a glance:

| | `Fee` definition | `Receipt.charge` |
|---|---|---|
| default (expand) | copy of `Amount` | `$ref: Fee` |
| `RefAliases: true` | `$ref: Amount` | `$ref: Fee` |
| `TransparentAliases: true` | copy of `Amount`, unreferenced | `$ref: Amount` |

Wider calibration lives in the `testdata/enhancements/alias-calibration-embed`
golden trio.

{{% notice style="note" %}}
Most APIs never need first-class aliases — prefer naming a real `swagger:model`
type over aliasing one. Reach for `RefAliases` / `TransparentAliases` only when
you specifically need to control whether an alias name survives in the output.

The `swagger:alias` *annotation* is
[deprecated]({{% relref "/maintainers/annotations/swagger-alias" %}})
and has no effect — alias rendering is governed by the plain Go alias plus these
options, or by `swagger:model` for a first-class definition.
{{% /notice %}}
