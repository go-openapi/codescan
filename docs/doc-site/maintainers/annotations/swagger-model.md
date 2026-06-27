---
title: "swagger:model"
weight: 90
description: "Publishes a Go type as a definitions entry."
---


## What it does

Declares a Go type as a published model. The scanner walks the type,
emits a schema into the spec's `definitions` map, and resolves
cross-references between models.

The title/description split follows a heuristic: a single-line comment
**ending in a period** becomes the `title`; one **without** a trailing
period becomes the `description`; a **multi-line** comment uses the
first line as `title` and the rest as `description`. The descriptive
prose must come **before** the `swagger:model` line — an
annotation-first block still publishes the model but drops its title
and description.

## Where it goes

On a type declaration (`type T struct { … }`, `type T int`,
`type T = Other`, …).

## Syntax

```ebnf
ModelAnnotation = ANN_MODEL , [ IDENT_NAME ] ;
```

The optional `IDENT_NAME` is the name the model takes in `definitions`
(default: the Go type's name). It must be a plain identifier (a JSON
label), not a Go-qualified name — a dotted name such as `utils.Error`
is rejected with a warning and dropped. Cross-package types resolve
automatically, so reference a model by its bare name.

The annotation opens a [`SchemaBlock`]({{% relref "grammar#schema-family" %}})
body — its fields and their doc comments carry the schema validations.

## Supported keywords

All [schema decorators]({{% relref "keywords#schema-decorators" %}}) plus the
[length / array / numeric validations]({{% relref "keywords#numeric-validations" %}})
on field doc comments.

## Example

```go
// Pet is the petstore's primary entity.
//
// swagger:model
type Pet struct {
	// ID is the unique identifier.
	ID int64 `json:"id"`

	// Name is the pet's display name.
	Name string `json:"name"`

	// Tags categorise the pet.
	Tags []string `json:"tags,omitempty"`
}
```

With a name override, the type is published as `#/definitions/PetWithExtras`:

```go
// swagger:model PetWithExtras
type DetailedPet struct { … }
```

A field group declaring several names (`R, G, B, A uint8`) emits **one
property per name**; a `json:` tag on the group cannot rename the
individual fields, though tag options still apply.

**Full example.** `fixtures/enhancements/named-struct-tags-ref/types.go`.
