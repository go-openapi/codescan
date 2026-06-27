---
title: "swagger:description"
weight: 45
description: "Overrides the godoc-derived description on a model, field, response, or header."
---

## Usage

```goish
// swagger:description <text>   (single line, or a blank-terminated body)
// swagger:description |        (opens a verbatim literal markdown block)
```

## What it does

Replaces the godoc-derived `description` on a schema with explicit text.

By default a description comes from a declaration's doc comment;
`swagger:description` overrides it when the godoc prose isn't what you want to
publish. It is a schema-family **override** — a sibling of
[`swagger:title`]({{% relref "swagger-title" %}}).

A trailing `|` opens a **verbatim literal markdown block**: the body is captured
exactly — blank lines, indentation, and table pipes preserved — until the next
line-leading annotation or end of comment. See
[Markdown descriptions]({{% relref "markdown-descriptions" %}}).

## Where it goes

On a type (model) doc comment, a struct-field doc comment, a `swagger:response`
struct, or a response header field.

## Grammar (EBNF)

```ebnf
DescriptionAnnotation = ANN_DESCRIPTION , RAW_VALUE ;
```

`RAW_VALUE` is the rest of the head line; under Option B a blank-terminated body
extends it, and a trailing `|` switches the body to verbatim literal capture.
The annotation dispatches through the
[schema parser]({{% relref "grammar#schema-family" %}}) (not the classifier
parser), so validation keywords co-located on the same comment group still
surface.

## Supported keywords

None of its own — the text (plus any folded body) is the entire argument. A bare
`swagger:description` with no text **suppresses** the godoc-derived description
and emits a `CodeEmptyOverride` diagnostic.

## Example

A plain override on a model and its fields:

{{< example
    go="shaping/overrides/overrides.go" goregion="model"
    json="shaping/overrides/testdata/widget.json" >}}

The `|` literal block captures a verbatim markdown body (table and list preserved):

{{< example
    go="shaping/markdowndesc/markdowndesc.go" goregion="markdown"
    json="shaping/markdowndesc/testdata/markdown.json" >}}
