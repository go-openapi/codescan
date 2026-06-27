---
title: Routes & operations
weight: 20
description: |
  Publish paths and operations — swagger:route and swagger:operation — with
  their parameters and responses.
---

Routes and operations turn an annotation into an entry in the spec's `paths`
map, wired to the parameters it accepts and the responses it returns. This page
covers the two operation annotations and the companion structs they reference.
Each pane pairs the annotated Go (left) with the exact fragment the scanner
emits (right), from the test-covered
[`docs/examples/concepts/routes`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/routes)
package.

For the exhaustive rule on any annotation below, follow its link to the
[Maintainers reference]({{% relref "/maintainers/annotations" %}}); the
`Parameters:` / `Responses:` body grammars are covered in
[Sub-languages]({{% relref "/maintainers/sub-languages" %}}).

## swagger:route

`swagger:route <METHOD> <path> [tags] <operationID>` declares a path and its
operation in one annotation. The body's `responses:` block ties status codes to
named responses (`$ref` into the spec's `responses`). It lives in a plain
comment block — no Go declaration required.

{{< example go="concepts/routes/routes.go" goregion="route"
            json="concepts/routes/testdata/route.json"
            full="concepts/routes/testdata/full.json" >}}

The body can also carry an indented `Parameters:` block to declare simple
parameters (path / query / header) inline — no `swagger:parameters` struct
needed. For body parameters or parameter sets shared across operations, use
[`swagger:parameters`](#swaggerparameters) instead. The block syntax is covered
in [sub-languages]({{% relref "/maintainers/sub-languages#parameters" %}}).

## swagger:operation

`swagger:operation` carries the same header but spells the operation out as a
YAML document after a `---` fence — useful when you want to author the operation
object directly (here a path parameter and an inline `$ref` response schema).

{{< example go="concepts/routes/routes.go" goregion="operation"
            json="concepts/routes/testdata/operation.json"
            full="concepts/routes/testdata/full.json" >}}

## swagger:parameters

`swagger:parameters <operationID>…` declares a struct whose fields become the
parameters of the named operation(s). Field doc comments carry `in:`, the
validations, and the description; the parameters attach to every operation ID
listed.

{{< example go="concepts/routes/routes.go" goregion="parameters"
            json="concepts/routes/testdata/parameters.json" jsonlabel="parameters on listPets" >}}

A field marked `in: body` makes its Go type the **request body** schema — the
usual shape for a POST or PUT payload:

{{< example go="concepts/routes/routes.go" goregion="bodyparam"
            json="concepts/routes/testdata/bodyparam.json" jsonlabel="parameters on createPet" >}}

When a parameter field's Go type is a struct (or any type that has no simple
Swagger representation), it cannot be a query/path/header parameter on its own. A
[`swagger:type`]({{% relref "/maintainers/annotations/swagger-type" %}}) override
collapses it to a simple parameter — a scalar, or a `[]`-wrapped scalar for an
array parameter:

{{< example go="concepts/routes/routes.go" goregion="paramtype"
            json="concepts/routes/testdata/paramtype.json" jsonlabel="parameters on filterPets" >}}

## swagger:response

`swagger:response <name>` declares a struct as a named entry in the spec's
top-level `responses`. A `Body` field (or `in: body`) becomes the response
schema; routes reference it by name. Here the body is a `[]Pet`, so the schema
is an array of `$ref`s.

{{< example go="concepts/routes/routes.go" goregion="response"
            json="concepts/routes/testdata/response.json" jsonlabel="responses[petsResponse]" >}}

## swagger:file

`swagger:file` on a parameter field marks it as a binary upload — the parameter
emits as `{type: file}`. It belongs on a `formData` field of a
`swagger:parameters` struct.

{{< example go="concepts/routes/routes.go" goregion="file"
            json="concepts/routes/testdata/file.json" jsonlabel="parameters on uploadPetPhoto" >}}

## externalDocs

An `externalDocs:` block (description + url) links an object out to external
documentation. It rides an **operation** (in a `swagger:route` or
`swagger:operation` body) and a **full schema** (a `swagger:model`). It is a
full-Schema-only keyword: on a simple-schema parameter (anything but `in: body`)
it is dropped with a diagnostic. The same `ExternalDocs:` block on a
`swagger:meta` package populates the spec's top-level `externalDocs` (see
[Document metadata]({{% relref "/tutorials/document-metadata" %}})).

{{< example go="concepts/routes/routes.go" goregion="externaldocs"
            json="concepts/routes/testdata/externaldocs.json" jsonlabel="operation externalDocs" >}}

On a model the link rides the definition, and it also rides individual
**struct fields**: on a plain field it attaches to the property directly; on a
`$ref`'d field it is lifted onto the field's `allOf` compound (a bare `$ref`
cannot carry sibling keywords). The value can be written as the indented block
above or as an equivalent inline `{ description: …, url: … }` mapping — which
reads better on a single-line doc comment:

{{< example go="concepts/routes/routes.go" goregion="externaldocs"
            json="concepts/routes/testdata/externaldocs_schema.json" jsonlabel="#/definitions/CatalogEntry" >}}

## What's next

- [Sharing parameters & responses]({{% relref "/tutorials/sharing-parameters-and-responses" %}})
  — declare a header or error response once and `$ref` it from many operations.
- [Validations]({{% relref "/tutorials/validations" %}}) — constrain parameter
  and field values.
- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) — the
  schemas these operations reference.
- [Shaping the output]({{% relref "/shaping-the-output" %}}) — `$ref` vs inline,
  aliases, and the other rendering knobs.
