---
title: Keeping annotations out of the godoc
weight: 41
description: |
  Let swagger annotations live inside a struct body or as trailing comments so
  the godoc above each declaration stays clean — the AfterDeclComments opt-in.
---

A godoc comment and an API description pursue different goals. The godoc is for
the Go developers reading the package; the API text is for the consumers of the
generated spec. By default codescan reads its annotations from the doc comment
**above** a declaration, which mixes the two concerns — a `swagger:model`,
`maxProperties:` or `swagger:strfmt` line sits right in the middle of the prose a
Go reader sees.

`AfterDeclComments` separates them. With the option on, codescan also reads
annotations placed **inside a struct body** (its leading comment) or **inlined as
a trailing comment** on the same line as the declaration. The godoc above stays
concise and human-facing while the swagger machinery lives out of it — same
annotation grammar, no new syntax. It is the placement counterpart to
[overriding titles & descriptions]({{% relref "/shaping-the-output/overriding-titles-and-descriptions" %}}),
which separates the same two concerns at the *text* level.

Each pane below pairs the annotated Go (left) with the exact fragment the scanner
emits (right), from the test-covered
[`docs/examples/shaping/afterdecl`](https://github.com/go-openapi/codescan/tree/master/docs/examples/shaping/afterdecl)
package.

## Inside a struct body, or trailing on a field

The `swagger:model` annotation (and any decl-level keyword such as
`maxProperties:`) can live as the **leading comment inside the struct body**,
above the first field. A field-level annotation like `swagger:strfmt` can ride a
**trailing comment** on the field line. The godoc above `Widget` says nothing
about swagger:

{{< code file="shaping/afterdecl/afterdecl.go" lang="go" region="struct" >}}

## Inlined on a defined type or alias

For a non-struct type — a defined type or a type alias — the annotation rides a
trailing comment after the declaration:

{{< code file="shaping/afterdecl/afterdecl.go" lang="go" region="aliases" >}}

## Turning it on

`AfterDeclComments` is opt-in and defaults to off — so existing code, where a
clean comment that happens to look like an annotation is just prose, is never
reinterpreted:

```go
codescan.Run(&codescan.Options{
    Packages:          []string{"./..."},
    ScanModels:        true,
    AfterDeclComments: true,
})
```

With the option **off**, the inside-body and trailing annotations above are inert
— the clean godoc carries no annotation, so nothing is discovered. With it
**on**, the same source yields the three definitions, each with its keywords
applied (and the clean godoc still supplies the human-facing `title`):

{{< compare left="shaping/afterdecl/testdata/off.json" leftlabel="Default — annotations inert"
            right="shaping/afterdecl/testdata/on.json" rightlabel="AfterDeclComments — discovered" >}}

{{% notice style="info" %}}
**Scope (v0.36).** The opt-in covers **type declarations** — a struct's
inside-body leading comment, a struct field's trailing comment, and the trailing
comment of a defined type or alias. **Routes and operations are already
position-agnostic**: a `swagger:route` / `swagger:operation` block inside a
function body is discovered with or without this option. Const-based enums are a
planned follow-up.
{{% /notice %}}

## What's next

- [Overriding titles & descriptions]({{% relref "/shaping-the-output/overriding-titles-and-descriptions" %}})
  — separate the godoc and the API text at the content level.
- [Single-line comments]({{% relref "/shaping-the-output/single-line-comments" %}})
  — how a plain comment is routed to `title` vs `description`.
