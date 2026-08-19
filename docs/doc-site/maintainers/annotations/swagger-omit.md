---
title: "swagger:omit"
weight: 105
description: "Drops named fields from what an embed promotes into the enclosing schema."
---

## Usage

```goish
// swagger:omit <field>[,<field>…]
```

## What it does

Stops the named fields being **promoted** out of an embedded type, so they never
reach the enclosing schema.

Embedding a shared type is how Go reuses a struct, but the reused type often
carries more than one particular endpoint should: server-assigned fields on a
create request, or a field the enclosing struct re-declares for itself.
`swagger:omit` is how the **author** resolves that — codescan does not guess
which fields were meant, it documents the type as written unless told otherwise.

Names are **Go field names**, never JSON aliases: the annotation acts before
names are computed, so it is indifferent to `json` tags and to
[`NameFromTags`]({{% relref "naming-from-tags" %}}).

It is a *pre-filter*, not an edit of the finished schema, so it reads the same
whether the embed is inlined or composed into an `allOf` member
(see [Composing embeds with allOf]({{% relref "composing-embeds-with-allof" %}})):
the field is simply never written.

## Where it goes

Two placements:

- **on the embed** — targets are plain field names of *that* embedded type. This
  is the ergonomic form and needs no qualification;
- **on the type declaration** — for an embed you cannot annotate (a type you do
  not own, or one nested deeper). A bare name resolves against the promoted set;
  a dotted path names the embed chain, `Base.ID` or `Outer.Inner.Deep`.

```go
// swagger:omit Base.ID,Created
type Decorated struct {
	Base
	// …
}
```

**Embeds only.** Every path segment but the last must name an embedded field:
`swagger:omit` removes *promoted* content, which is the only thing the enclosing
schema owns. To exclude a struct's own field, use
[`swagger:ignore`]({{% relref "swagger-ignore" %}}) on the field itself.

## Grammar (EBNF)

```ebnf
OmitBlock  = ANN_OMIT , OmitTarget , { "," , OmitTarget } ;
OmitTarget = GoIdent , { "." , GoIdent } ;
```

The whole remainder of the line is the argument list; spaces after commas are
allowed (`swagger:omit ID, Created`).

## Supported keywords

None. `swagger:omit` is a classifier: it takes arguments and opens no keyword
block.

## Example

The go-swagger#1992 shape: a request body embeds the shared domain type, and the
server-assigned fields are dropped from *this* body only.

{{< example
    go="concepts/omit/omit.go" goregion="omit"
    json="concepts/omit/testdata/body.json" jsonlabel="The request body" >}}

The shared type is never touched, so its own definition — which the response
`$ref`s — still documents every field:

{{< code file="concepts/omit/testdata/user.json" lang="json" >}}

## Diagnostics

All three are **Hints** — informational, never blocking:

| code | fires when |
|---|---|
| `scan.omit-unresolved` | the target names no field of the embedded type: a typo, or a field renamed upstream |
| `scan.omit-behind-ref` | the embed is composed as a `$ref` (an annotated `swagger:model`); Swagger 2.0 cannot subtract a property from a reference, so the omission is dropped rather than silently forking the definition |
| `scan.shadowed-embed-field` | a field re-declared with `json:"-"` carries the Go name of a promoted one — see below |

`swagger:omit` is the only annotation whose output depends on a name the Go
compiler never checks; everything else codescan emits is derived from types.
`scan.omit-unresolved` reports it when a field is renamed upstream, instead of
letting the annotation rot silently, so wire
[`OnDiagnostic`](https://pkg.go.dev/github.com/go-openapi/codescan#Options) if
you rely on the annotation.

{{% notice style="warning" title="`json:\"-\"` does not hide a promoted field" %}}
Re-declaring a promoted field with `json:"-"` looks like it should hide it. It
does not: `encoding/json` ignores a `-` field **entirely**, so it never enters
the name set, never shadows the promoted one, and Go keeps marshalling the
embedded field. `swagger:omit` is the annotation that removes it for real.
{{% /notice %}}

## Deprecated

No. Added in v0.37.
