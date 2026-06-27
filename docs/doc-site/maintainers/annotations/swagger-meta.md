---
title: "swagger:meta"
weight: 80
description: "Declares the package as the top-level OpenAPI spec container."
---

## Usage

```goish
// swagger:meta
```

## What it does

Declares the package as the OpenAPI spec container.

The scanner reads the package doc comment for the top-level spec fields:
title (via [stripPackagePrefix]({{% relref "grammar#prose" %}}) of the
doc's first line), description, license, contact, host, basePath, version,
schemes, consumes, produces, securityDefinitions, extensions, and the rest
of the meta keyword surface.

## Where it goes

On the package doc comment. No arguments — a bare annotation.

## Grammar (EBNF)

```ebnf
MetaBlock = ANN_META , [ Title ] , [ Description ] , MetaBody ;
```

The body is a `MetaBody` of single-line `MetaKeyword`s
(`version`, `host`, `basePath`, `license`, `contact`, `schemes`) and
`MetaRawBlock`s (`consumes`, `produces`, `security`,
`securityDefinitions`, `tos`). See [grammar §meta-family]({{% relref "grammar#meta-family" %}}).

## Supported keywords

All [meta single-line keywords]({{% relref "keywords#meta-single-line-keywords" %}})
(`schemes`, `version`, `host`, `basePath`, `license`, `contact`) plus the
meta-scope [body keywords]({{% relref "keywords#body-keywords" %}})
(`consumes`, `produces`, `security`, `securityDefinitions`, `extensions`,
`infoExtensions`, `tos`, `externalDocs`, `tags`). A `Tags:` block declares the
spec's top-level `tags` (name, description, nested `externalDocs`, `x-*`
extensions per tag).

## Example

{{< example
    go="concepts/meta/doc.go" goregion="meta"
    json="concepts/meta/testdata/meta.json" >}}

**Full example.** `fixtures/goparsing/spec/api.go`.
