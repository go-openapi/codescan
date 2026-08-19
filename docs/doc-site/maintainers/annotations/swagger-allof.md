---
title: "swagger:allOf"
weight: 30
description: "Marks a struct as participating in an allOf composition."
---

## Usage

```goish
// swagger:allOf
```

## What it does

Marks a struct as participating in an `allOf` composition.

The struct's fields plus any embedded `swagger:model`-tagged base produce
an `allOf: [$ref base, {inline fields}]` schema. The companion convention
is to embed the base type as an anonymous field with this annotation on the
embedding's doc comment (or on the embedded type itself).

## Where it goes

On a struct field that embeds another type, or on a struct type that has
at least one embedded base.

## Grammar (EBNF)

```ebnf
AllOfBlock = ANN_ALLOF , [ Title ] , [ Description ] ;
```

The annotation takes no arguments; an optional title/description may
follow on the doc comment.

## Supported keywords

[Schema-context keywords]({{% relref "/maintainers/keywords/schema-validations-and-decorators#schema-decorators" %}}) on
the inline-object member (the second `allOf` element).

## Do not put other annotations beside it

`swagger:allOf` takes no arguments, and no other classifier annotation belongs in
an embedded field's doc comment. `swagger:strfmt` and `swagger:type` written
there are **ignored**, and codescan reports them under
`scan.ineffective-annotation`:

```go
type Wrong struct {
	// swagger:allOf
	// swagger:strfmt uuid   ← ignored, and warned about
	Token
}
```

The reason is that an embed contributes the shape of the type it embeds, and
**that type's own declaration** fixes the shape — never the site that embeds
it. So the annotation belongs one level down:

```go
// Token is rendered as a formatted string wherever it appears.
//
// swagger:strfmt uuid
type Token [16]byte

type Right struct {
	// swagger:allOf
	Token
}
```

This is not specific to `allOf`: the same annotations are ignored on a plain
(uncomposed) embed too, and reported the same way. They are honoured on an
ordinary — non-embedded — field, which is what makes the mistake an easy one.

## Example

A struct embedding a `swagger:model` base with `swagger:allOf` on the embed
produces an `allOf` of the base `$ref` and an inline-object member carrying
the embedding struct's own fields:

{{< example
    go="concepts/models/models.go" goregion="allof"
    json="concepts/models/testdata/allof.json" >}}

The same composition applies when the embedding struct is a
`swagger:response` body: the embedded base emits an `allOf: [{$ref}, …]`
arm only when it is a `swagger:model` (a definition exists to point at);
an embedded `swagger:response` has its fields inlined instead.

**Full example.** `testdata/enhancements/allof-edges/types.go`.
