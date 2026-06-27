---
title: "Keyword reference"
weight: 20
description: "The keyword: value forms recognised inside annotation blocks — grouped by class, with the annotation contexts that accept each one and its value shape."
---

Keywords are the `keyword: value` lines that decorate an
[annotation]({{% relref "annotations" %}}) block. They come in two flavours:
**inline** (one line, `keyword: value`, the value classified by a
[value shape]({{% relref "appendix#value-shapes" %}})) and **body** (a header line
plus indented continuation lines — a flat token list, a YAML map, or a per-line
sub-language). Three things matter about any keyword: the **class** it belongs to,
the **annotation contexts** that accept it, and its **value shape**.

This section groups the surface by class — pick the page that matches what you're
decorating. For the formal productions see [grammar.md]({{% relref "grammar" %}});
for the value-shape and context-token reference tables see the
[Appendix]({{% relref "appendix" %}}).

## Keyword classes

| Class | Covers | Keywords |
|-------|--------|----------|
| [Parameters & responses]({{% relref "parameters-and-responses" %}}) | request parameters and response headers (the reduced SimpleSchema surface) | `in`, `name`, `collectionFormat`, `examples`, + the shared validations |
| [Schema validations & decorators]({{% relref "schema-validations-and-decorators" %}}) | model schemas and struct fields | `maximum`/`minimum`/`multipleOf`, `maxLength`/`minLength`, `maxItems`/`minItems`, `maxProperties`/`minProperties`, `pattern`, `patternProperties`, `additionalProperties`, `unique`, `default`, `example`, `enum`, `required`, `readOnly`, `discriminator`, `deprecated` |
| [Routes & operations]({{% relref "/maintainers/keywords/routes-and-operations" %}}) | `swagger:route` / `swagger:operation` metadata | `schemes`, `consumes`, `produces`, `responses`, `parameters`, `tags` |
| [Security]({{% relref "/maintainers/keywords/security" %}}) | authentication requirements & scheme definitions | `security`, `securityDefinitions` |
| [Spec metadata]({{% relref "spec-metadata" %}}) | `swagger:meta` top-of-document fields | `version`, `host`, `basePath`, `license`, `contact`, `tos`, `infoExtensions`, `externalDocs`, `extensions`, `tags` |

{{< children type="card" description="true" >}}

## Context matrix

Which annotation family accepts a given keyword — the transpose of the
[annotation × keyword matrix]({{% relref "annotations#annotation--keyword-compatibility-matrix" %}}).
A ✅ means the keyword is legal on that annotation (on the annotation's own block or
on one of its fields); a blank means it is rejected there with a
`CodeContextInvalid` diagnostic. The detailed entry for each keyword lives on its
class page (linked above).

| Keyword | `meta` | `model` | `parameters` | `response` | `route` | `operation` |
|---------|:------:|:-------:|:------------:|:----------:|:-------:|:-----------:|
| `maximum` `minimum` `multipleOf` | | ✅ | ✅ | ✅ | | |
| `maxLength` `minLength` | | ✅ | ✅ | ✅ | | |
| `maxItems` `minItems` `unique` | | ✅ | ✅ | ✅ | | |
| `pattern` | | ✅ | ✅ | ✅ | | |
| `collectionFormat` | | | ✅ | ✅ | | |
| `maxProperties` `minProperties` | | ✅ | | | | |
| `patternProperties` `additionalProperties` | | ✅ | | | | |
| `default` `example` `enum` | | ✅ | ✅ | ✅ | | |
| `required` | | ✅ | ✅ | | | |
| `readOnly` `discriminator` | | ✅ | | | | |
| `deprecated` | | ✅ | | | ✅ | ✅ |
| `in` | | | ✅ | | | |
| `name` | | ✅ | ✅ | ✅ | | |
| `examples` | | | | ✅ | | |
| `schemes` `consumes` `produces` | ✅ | | | | ✅ | ✅ |
| `security` | ✅ | | | | ✅ | ✅ |
| `securityDefinitions` | ✅ | | | | | |
| `responses` `parameters` | | | | | ✅ | ✅ |
| `tags` | ✅ | | | | ✅ | ✅ |
| `version` `host` `basePath` `license` `contact` `tos` | ✅ | | | | | |
| `infoExtensions` | ✅ | | | | | |
| `externalDocs` | ✅ | ✅ | | | ✅ | ✅ |
| `extensions` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

The `parameters` / `response` columns also cover the **items** sub-context (array
elements): the array-element validations ride there too. `model` covers
`swagger:allOf` member fields. See the
[Appendix]({{% relref "appendix#annotation-contexts" %}}) for the precise meaning of
each context token (`param`, `header`, `schema`, `items`, …).
