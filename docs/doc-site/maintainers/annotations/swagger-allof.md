---
title: "swagger:allOf"
weight: 30
description: "Marks a struct as participating in an allOf composition."
---


## What it does

Marks a struct as participating in an `allOf` composition. The struct's
fields plus any embedded `swagger:model`-tagged base produce an
`allOf: [$ref base, {inline fields}]` schema. The companion convention is
to embed the base type as an anonymous field with this annotation on the
embedding's doc comment (or on the embedded type itself).

## Where it goes

On a struct field that embeds another type, or on a struct type that has
at least one embedded base.

## Syntax

```ebnf
AllOfBlock = ANN_ALLOF , [ Title ] , [ Description ] ;
```

The annotation takes no arguments; an optional title/description may
follow on the doc comment.

## Supported keywords

[Schema-context keywords]({{% relref "keywords#schema-decorators" %}}) on
the inline-object member (the second `allOf` element).

## Example

```go
// Animal is the abstract base.
//
// swagger:model
type Animal struct {
	Kind string `json:"kind"`
}

// Dog is an Animal with a breed.
//
// swagger:model
type Dog struct {
	// swagger:allOf
	Animal

	Breed string `json:"breed"`
}
```

Produces:

```json
"Dog": {
  "allOf": [
    {"$ref": "#/definitions/Animal"},
    {
      "type": "object",
      "properties": {
        "breed": {"type": "string", "x-go-name": "Breed"}
      }
    }
  ]
}
```

The same composition applies when the embedding struct is a
`swagger:response` body: the embedded base emits an `allOf: [{$ref}, …]`
arm only when it is a `swagger:model` (a definition exists to point at);
an embedded `swagger:response` has its fields inlined instead.

**Full example.** `fixtures/enhancements/allof-edges/types.go`.
