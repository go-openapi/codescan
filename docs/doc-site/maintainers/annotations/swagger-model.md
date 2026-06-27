---
title: "swagger:model"
weight: 90
description: "Publishes a Go type as a definitions entry."
---

## Usage

```goish
// swagger:model [<name>]   (where <name> overrides the definition name; defaults to the Go type name)
```

## What it does

Declares a Go type as a published model.

The scanner walks the type, emits a schema into the spec's `definitions` map,
and resolves cross-references between models.

The title/description split follows a heuristic: a single-line comment
**ending in a period** becomes the `title`. One **without** a trailing period
becomes the `description`; a **multi-line** comment uses the first line as
`title` and the rest as `description`.

The descriptive prose must come **before** the `swagger:model` line — an
annotation-first block still publishes the model but drops its title and
description.

## Where it goes

On a type declaration (`type T struct { … }`, `type T int`,
`type T = Other`, …).

## Grammar (EBNF)

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

Every [schema decorator]({{% relref "/maintainers/keywords/schema-validations-and-decorators#schema-decorators" %}}) and
[validation keyword]({{% relref "/maintainers/keywords/schema-validations-and-decorators#numeric-validations" %}}) is accepted
on a field doc comment. A keyword that is **not compatible** with the field's
inferred schema type (e.g. `minLength` on an integer) is ignored and raises a
diagnostic.

## Example

The doc comment above the type drives the model's name, title and description:

```go
// Pet is the petstore's primary entity.            <- title (first line, ends with a period)
//
// A pet can be any little animal you care about.   <- description
// In this example the model name is inferred from the type name, here "Pet".
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

Pass an argument to override the name; the type is then published as
`#/definitions/PetWithExtras`:

```go
// swagger:model PetWithExtras
type DetailedPet struct { … }
```

A single field group declaring several names emits **one property per name**.
A `json:` tag on the group cannot rename the individual fields — each keeps its
own name — though tag options still apply:

{{< example
    go="concepts/models/models.go" goregion="multiname"
    json="concepts/models/testdata/multiname.json" >}}

**Full example.** `fixtures/enhancements/named-struct-tags-ref/types.go`.
