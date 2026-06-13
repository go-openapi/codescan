---
title: Tutorials
weight: 20
description: |
  Learn codescan by spec concept — model definitions, routes and operations,
  validations, examples, and document metadata — each shown as annotated Go
  next to the Swagger it produces.
---

These tutorials teach codescan **by spec concept**, not annotation by
annotation. Each page takes one thing you want in your OpenAPI document — a
model definition, a route, a validated field — and shows the Go annotation that
produces it next to the resulting JSON, side by side.

Every Go snippet on these pages comes from the test-covered
[`docs/examples`](https://github.com/go-openapi/codescan/tree/master/docs/examples)
module, and every JSON pane is a golden file a test regenerates — so the
examples cannot drift from what the scanner actually emits.

## Reading the panes

The example panes put the **annotation in** on the left and the **spec concept
out** on the right:

{{< example go="petstore/pet.go" goregion="model" golabel="Annotated Go"
            json="basic/testdata/swagger.json" jsonlabel="Generated spec" >}}

## The concepts

{{< children type="card" description="true" >}}

When you want the exhaustive rule rather than an example, every page links into
the [Maintainers reference]({{% relref "/maintainers" %}}); the
[Annotation index]({{% relref "/annotation-index" %}}) maps every annotation to
both.
