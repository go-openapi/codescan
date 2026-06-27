---
title: "swagger:default"
weight: 40
description: "Classifier hint marking a value declaration as a spec default anchor."
---

## Usage

```goish
// swagger:default
```

## What it does

Marks the surrounding declaration as the spec's default value for the
corresponding shape.

Used in narrow contexts where the scanner expects an explicit anchor for a
default. This annotation is **value-only** — there's no exported entity it
publishes; it's a classifier hint the scanner consumes during discovery.

## Where it goes

On a value declaration (`var`, `const`) or a struct field.

## Grammar (EBNF)

```ebnf
DefaultClassifierBlock = ANN_DEFAULT , [ Title ] , [ Description ] ;
```

Takes no argument — an optional title/description may follow on the
doc comment.

## Supported keywords

None of its own. Most spec defaults are instead carried by the
[`default:` keyword]({{% relref "keywords#default" %}}) on the relevant
field; this annotation has a narrow surface and is not commonly authored
directly.

## Example

`swagger:default` is value-only: it produces no definition, so there is no
emitted spec to render. The source below shows the narrow classifier-hint
form — in practice most defaults come from the
[`default:` keyword]({{% relref "keywords#default" %}}) on a field.

{{< code file="concepts/examples/examples.go" region="swaggerdefault" lang="go" >}}
