---
title: Inline response bodies
weight: 20
description: |
  Declare a route's responses inline with the body: sub-language — a primitive,
  an array, or a model $ref — without writing a swagger:response struct.
---

The [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}})
tutorial declares each response as a `swagger:response` struct with a `Body`
field. That is the right tool when a response is reused across operations or has
headers. But when a response body is just *"a string"*, *"an array of `Pet`"*, or
*"a `Pet`"*, the wrapper struct is pure boilerplate.

The `Responses:` block of a `swagger:route` accepts the **`body:` sub-language**,
which names the body shape directly:

- `body:string` (or `number` / `integer` / `boolean`) — a primitive body;
- `body:Pet` — a `$ref` to the `Pet` definition;
- `body:[]Pet` — an array of that `$ref` (repeat `[]` to nest deeper);
- any trailing words after the body token become the response **description**;
  omit them and codescan derives one — the referenced model's godoc (`default`
  above → the `Pet` doc comment), or the HTTP status reason for a numeric code
  (`400` above → "Bad Request").

{{< example go="shaping/inlineresponses/inlineresponses.go" goregion="inline" golabel="swagger:route"
            json="shaping/inlineresponses/testdata/pathitem.json" jsonlabel="paths[/pets]" >}}

No `swagger:response` struct is defined — the three responses are produced
entirely from the `body:` tokens, and the `Pet` model is pulled into
`definitions` because the body `$ref`s reach it.

{{% notice style="info" %}}
A bare untagged token is read as a **response name**, never a type: `200: string`
is a (dangling) `$ref` to a response called `string`, not a primitive body. Use
the explicit `body:string` form for a primitive. The full grammar — tags,
untagged-token rules, and the reserved `array`/`object`/`file` keywords — is in
[sub-languages → Responses]({{% relref "/maintainers/sub-languages#responses" %}}).
{{% /notice %}}
