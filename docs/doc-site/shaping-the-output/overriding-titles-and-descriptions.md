---
title: Overriding titles & descriptions
weight: 43
description: |
  Replace the godoc-derived title and description with API-facing text using
  swagger:title and swagger:description — on models, fields, $ref'd fields and
  responses.
---

A Go doc comment is written for Go readers. The same prose is not always what
you want in the published API — a comment may explain internal usage, reference
Go types, or simply read awkwardly to an API consumer. `swagger:title` and
`swagger:description` let the spec text **diverge** from the godoc: the
annotation replaces the prose-derived value, leaving the Go comment free to say
whatever Go developers need.

This is the *explicit* counterpart to
[Single-line comments]({{% relref "/shaping-the-output/single-line-comments" %}}),
which controls how a plain comment is *implicitly* routed to `title` vs
`description`. Each pane below pairs the annotated Go (left) with the exact
fragment the scanner emits (right), from the test-covered
[`docs/examples/shaping/overrides`](https://github.com/go-openapi/codescan/tree/master/docs/examples/shaping/overrides)
package.

## Overriding a model and its fields

`swagger:title <text>` sets the `title`; `swagger:description <text>` sets the
`description`. Both sit in the comment block beside `swagger:model` (on a type)
or beside a field's other keywords. The model's Go-facing godoc here is replaced
wholesale by the two overrides:

{{< code file="shaping/overrides/overrides.go" lang="go" region="model" >}}

{{< code file="shaping/overrides/testdata/widget.json" lang="json" >}}

A few things to read out of that pane:

- **`title` on a property comes *only* from an override.** A field's godoc
  becomes its `description`; codescan never derives a property `title` from
  prose, so `swagger:title` (as on `label`) is the only way to set one.
- **`plain` keeps its godoc** — no override means no change. Overrides are
  strictly opt-in; un-annotated declarations behave exactly as before.

### Multi-line descriptions

`swagger:description` may span several lines. The lines immediately following
the annotation fold into one description (joined with newlines) and the body
**terminates at the first blank line, keyword, annotation, or end of comment**.
The `notes` field above shows this: its two prose lines fold together, and the
ordinary godoc paragraph after the blank line is discarded.

### Keeping a co-located validation keyword

Because the override annotations dispatch through the schema family, a
validation keyword on the *same* field still applies — they co-exist rather
than one shadowing the other. The `capacity` field carries both
`swagger:description` and `maximum: 1000`, and the output keeps both.

### Suppressing a godoc comment

A **bare** `swagger:description` (no text, empty body) applies the *empty* value
— a deliberate way to drop a godoc comment from the spec without deleting it
from the source. Because a stray bare marker could also be an accident,
codescan raises a `scan.empty-override` warning through `OnDiagnostic`. The
`suppressed` field above emits no `description` at all.

## Overrides beside a `$ref`

`title` and `description` are symmetric `$ref` siblings: on a field whose Go
type is a referenced model, they follow the **same preservation rule a prose
description does**. Under the default flags they drop to a bare `$ref`; with
[`EmitRefSiblings`]({{% relref "/shaping-the-output/descriptions-beside-a-ref" %}})
they ride alongside the `$ref` as direct siblings.

{{< compare left="shaping/overrides/testdata/gadget_bare.json" leftlabel="Default — dropped to a bare $ref"
            right="shaping/overrides/testdata/gadget_siblings.json" rightlabel="EmitRefSiblings — kept as siblings" >}}

## Responses and headers

`swagger:description` also overrides the description of a `swagger:response` and
of its response headers. OpenAPI 2.0 Response and Header objects have **no
`title` field**, so a `swagger:title` on a response or header is rejected with a
`parse.context-invalid` diagnostic — the description override still applies.

{{< code file="shaping/overrides/overrides.go" lang="go" region="response" >}}

{{< code file="shaping/overrides/testdata/errorresponse.json" lang="json" >}}

{{% notice style="info" %}}
**Precedence.** An override always wins over the godoc-derived value. Absent →
the godoc is used unchanged. Empty (bare marker) → the empty value is applied
*and* `scan.empty-override` is raised. `swagger:title` is schema-only; on a
response/header it is dropped with `parse.context-invalid`.
{{% /notice %}}

## What's next

- [Single-line comments]({{% relref "/shaping-the-output/single-line-comments" %}})
  — the implicit `title` vs `description` routing this overrides.
- [Descriptions beside a `$ref`]({{% relref "/shaping-the-output/descriptions-beside-a-ref" %}})
  — the `EmitRefSiblings` rule that title/description ride.
