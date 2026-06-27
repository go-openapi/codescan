---
title: "swagger:alias"
weight: 20
description: "Deprecated no-op — alias rendering is controlled by Go aliases + options."
---


{{% notice style="warning" %}}
**Deprecated.** `swagger:alias` no longer affects the emitted spec. It is an
empty sink that only raises a `validate.deprecated` diagnostic.
{{% /notice %}}

## What it does

Nothing, today. Earlier documentation claimed it published a `$ref` to
the alias target; that was never accurate. Its only real effect was to
force a named **primitive** type to inline its scalar (e.g.
`{type: string}`) instead of producing the `$ref` a named type
otherwise gets — and that force-inline behaviour has been removed.

## Where it went

On a type alias / named-type declaration.

## Syntax

```ebnf
AliasBlock = ANN_ALIAS , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
```

The optional `IDENT_NAME` is ignored — the annotation has no effect.

## Migration

- To **inline** a type at a use site, use `swagger:type inline` on the
  field (see [`swagger:type`]({{% relref "swagger-type" %}})).
- To publish a type as a **first-class definition** that fields `$ref`,
  use [`swagger:model`]({{% relref "swagger-model" %}}).
- To control alias rendering **globally**, use the `RefAliases` /
  `TransparentAliases` options. A plain (unannotated) Go alias
  `type T = Other` dissolves to its target by default. See
  [Alias rendering]({{% relref "/shaping-the-output/names-and-refs/alias-rendering" %}}).

## Supported keywords

None — the annotation is inert.
