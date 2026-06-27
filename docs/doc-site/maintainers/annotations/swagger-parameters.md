---
title: "swagger:parameters"
weight: 120
description: "Declares a Go struct as the parameter set for one or more operations."
---


## What it does

Declares a Go struct as the parameters set for one or more operations.
Each field becomes one parameter on the named operation(s), and the
field's doc comment carries its `in:`, `required:`, validations, and
description.

- A parameter's **name** comes from the field's `json:` tag, falling
  back to the Go field name when there is no tag (the `form:` tag is not
  consulted). A `name:` keyword in the field doc takes precedence over
  both and is the canonical, preferred way to set the name — the legacy
  `swagger:name` *annotation* is inert here and emits a `context-invalid`
  diagnostic pointing at `name:`. See the
  [universal `name` keyword]({{% relref "/maintainers/keywords#name" %}}).
- Operation IDs **accumulate**: the same ID may appear in several
  `swagger:parameters` lines to compose a set from multiple structs, and
  one struct may carry several lines splitting a long ID list.
- `swagger:parameters` declarations are collected across **all scanned
  packages** and matched to operations by ID, so a shared set can live in
  its own package.

## Where it goes

On a struct declaration. A bare slice variable (`var Filters []string`)
carries no per-field `in:`/`type:`/`required:`, so parameters must be a
struct.

## Syntax

```ebnf
ParametersAnnotation = ANN_PARAMETERS , IDENT_NAME , { IDENT_NAME } ;
```

The `IDENT_NAME` arguments are the operation IDs this set applies to (at
least one). The first argument may instead be a `*` wildcard (spec-level
shared `#/parameters/{name}`) or a `/path` (inlined into that exact
path-item) — see
[Sharing parameters & responses]({{% relref "/tutorials/sharing-parameters-and-responses" %}}).

The annotation opens a [`SchemaBlock`]({{% relref "grammar#schema-family" %}})
body — field doc comments carry the parameter validations.

## Supported keywords

[param-context keywords]({{% relref "keywords#parameter-location" %}}) on
fields: `in`, `required`, the numeric / length / format validations,
`default`, `example`, `enum`, `allowEmptyValue`, `collectionFormat`.

## Example

```go
// ListItemsParams declares pagination + filter parameters for the
// listItems operation.
//
// swagger:parameters listItems
type ListItemsParams struct {
	// Offset is the page offset.
	//
	// in: query
	// minimum: 0
	// default: 0
	Offset int `json:"offset"`

	// Limit is the page size.
	//
	// in: query
	// minimum: 1
	// maximum: 100
	// default: 20
	Limit int `json:"limit"`

	// Tag is the filter tag.
	//
	// in: query
	// required: false
	Tag string `json:"tag,omitempty"`
}
```

**Full example.** `fixtures/enhancements/simple-schema-violation/api.go`.
