---
title: Documenting generic responses
weight: 22
description: |
  Your handlers return one generic envelope with an interface{} payload, but you
  want the spec to describe a concrete type per operation. Doc-only structs that
  embed the envelope and shadow the payload field close the gap.
---

A common Go pattern is a single response *envelope* — a JSend-style wrapper that
every handler returns, with an open `any` (`interface{}`) field for the payload:

{{< example go="shaping/genericenvelopes/genericenvelopes.go" goregion="envelope" golabel="The generic envelope"
            json="shaping/genericenvelopes/testdata/apiresponse.json" jsonlabel="definitions[APIResponse]" >}}

codescan reads this faithfully: `Data` is `any`, so in the spec it becomes an
**open schema** — `{"x-go-name":"Data"}`, no `type`, no `$ref`. That is correct
(the field genuinely accepts anything), but it is not what you want in an API
contract, where each operation returns a *specific* payload.

codescan will not grow a per-route override syntax like swaggo's
`body:APIResponse{Data: StatusReport}` — that asks the scanner to invent a type
the code never declares. Instead, declare the type: a **doc-only struct** that
mirrors the envelope but pins the payload.

## Embed the envelope, shadow the payload

The DRY way is to embed the generic envelope — promoting its `Status` and
`Message` — and re-declare only the one opaque field with a concrete type:

{{< example go="shaping/genericenvelopes/genericenvelopes.go" goregion="docstruct" golabel="Doc-only envelope"
            json="shaping/genericenvelopes/testdata/statusenvelope.json" jsonlabel="definitions[StatusEnvelope]" >}}

The locally-declared `Data` is shallower than the embedded one, so it **wins** —
both in codescan's view and in `encoding/json` at runtime. The result is a clean
flat object whose `data` is a concrete `$ref`, with `status` and `message`
carried over from the embed:

{{< compare left="shaping/genericenvelopes/testdata/apiresponse.json"  leftlabel="Generic — data is open"
            right="shaping/genericenvelopes/testdata/statusenvelope.json" rightlabel="Doc-only — data is concrete" >}}

You restate **one** field, not the whole envelope. Point the route's response at
the doc-only struct with the [`body:` sub-language]({{% relref "/shaping-the-output/inline-response-bodies" %}}),
and specialise the same envelope per operation with a different payload each
time:

{{< example go="shaping/genericenvelopes/genericenvelopes.go" goregion="routes" golabel="swagger:route"
            json="shaping/genericenvelopes/testdata/paths.json" jsonlabel="paths" >}}

{{% notice style="info" %}}
**The handler never changes.** Your code keeps returning the generic
`APIResponse`; the doc-only structs exist only to give the scanner a concrete
shape. They are valid Go, though — because the shadowing `Data` wins,
`StatusEnvelope{}` marshals to exactly the same JSON the generic envelope would,
so you may return one directly if you prefer a typed handler.
{{% /notice %}}

## When embedding doesn't fit

If your envelope has fields you would rather not promote — or you want the
documented type to live behind a reusable `swagger:response` — restate the
fields explicitly instead of embedding:

```go
// swagger:model
type StatusEnvelope struct {
    Status  string       `json:"status"`
    Data    StatusReport `json:"data"`
    Message string       `json:"message,omitempty"`
}
```

This produces the same concrete `data`, at the cost of repeating every field. The
embed-and-shadow form above is preferred whenever the envelope's other fields map
through unchanged.
