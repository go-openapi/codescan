---
title: Alias rendering
weight: 60
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

This is an **advanced, rarely-needed** case. To keep the alias name in the spec
(its own definition that other schemas `$ref`), annotate the alias with
`swagger:model`. Two top-level options then govern how that first-class alias
*definition* is shaped:

- **default (expand)** — the alias definition is a structural copy of the
  target.
- **`RefAliases: true`** — the alias definition is a `$ref` chain to the target
  (`{"$ref": "#/definitions/Money"}`), preserving the alias name at use sites.
- **`TransparentAliases: true`** — aliases dissolve to their target everywhere,
  overriding the per-declaration annotation (use sites become `$ref` to the
  target, as in the default-dissolve example above).

The canonical witnesses are the `fixtures/enhancements/alias-calibration-embed`
golden trio.

{{% notice style="note" %}}
Most APIs never need first-class aliases — prefer naming a real `swagger:model`
type over aliasing one. Reach for `RefAliases` / `TransparentAliases` only when
you specifically need to control whether an alias name survives in the output.

The `swagger:alias` *annotation* is
[deprecated]({{% relref "/maintainers/annotations#swaggeralias--deprecated" %}})
and has no effect — alias rendering is governed by the plain Go alias plus these
options, or by `swagger:model` for a first-class definition.
{{% /notice %}}
