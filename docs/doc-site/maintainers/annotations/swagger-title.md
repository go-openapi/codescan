---
title: "swagger:title"
weight: 165
description: "Overrides the godoc-derived title on a model or field."
---

## Usage

```goish
// swagger:title <text>   (where <text> is the rest of the line)
```

## What it does

Replaces the godoc-derived `title` on a schema with explicit text.

By default a model's title comes from the first line of its doc comment;
`swagger:title` overrides that when the prose isn't the title you want to
publish. It is a schema-family **override** — a sibling of
[`swagger:description`]({{% relref "swagger-description" %}}).

## Where it goes

On a type (model) doc comment or a struct-field doc comment. It is
schema-only: a response has no `title` (the annotation is ignored there), and on
a non-body parameter or response header it is rejected with a
`CodeContextInvalid` diagnostic.

## Grammar (EBNF)

```ebnf
TitleAnnotation = ANN_TITLE , RAW_VALUE ;
```

`RAW_VALUE` is the rest of the head line, captured verbatim. The annotation
dispatches through the [schema parser]({{% relref "grammar#schema-family" %}})
(not the classifier parser), so validation keywords co-located on the same
comment group still surface as schema validations.

## Supported keywords

None of its own — the text is the entire argument. A blank `swagger:title`
emits a `CodeEmptyOverride` diagnostic.

## Example

{{< example
    go="shaping/overrides/overrides.go" goregion="model"
    json="shaping/overrides/testdata/widget.json" >}}

See [Overriding titles & descriptions]({{% relref "overriding-titles-and-descriptions" %}})
for the full walkthrough.
