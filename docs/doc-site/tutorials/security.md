---
title: Security
weight: 55
description: |
  Declare security schemes in swagger:meta, require them per route, or keep
  security out of your code entirely and overlay it onto the spec.
---

OpenAPI 2.0 splits authentication into two parts: **security definitions** name
the schemes (an API key, OAuth2, HTTP Basic), and **security requirements**
reference those schemes — document-wide and/or per operation. The panes below
are backed by the test-covered
[`docs/examples/concepts/security`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/security)
package.

## Declare the schemes

A `SecurityDefinitions:` block in `swagger:meta` declares every scheme once; a
`Security:` block sets the **document-wide default** requirement that applies to
operations that do not state their own.

{{< example go="concepts/security/doc.go" goregion="meta" golabel="Package doc comment"
            json="concepts/security/testdata/schemes.json" jsonlabel="securityDefinitions + security" >}}

The scheme `type` drives the rest: `apiKey` needs `in` + `name`, `oauth2` needs a
`flow` (and the URLs/`scopes` it implies), `basic` needs nothing more. The full
scheme surface is in the
[`securityDefinitions` reference]({{% relref "/maintainers/keywords#securitydefinitions" %}}).

## Require a scheme on a route

A route with no `Security:` keyword inherits the document-wide default
(`api_key`, above). A route that needs something different states its own
`Security:` requirement — here `createReport` requires `oauth2` with the `read`
and `write` scopes, overriding the default:

{{< example go="concepts/security/routes.go" goregion="routes" golabel="swagger:route"
            json="concepts/security/testdata/route.json" jsonlabel="security on createReport" >}}

A `Security:` block is plain **YAML** — a sequence of requirement objects.
Scopes are a flow list (`[read, write]`) or a block list; an empty list
(`api_key: []`) is the scheme with no scopes. The combining rule follows
OpenAPI 2.0:

- **multiple schemes in one item are ANDed** — all are required;
- **separate items are ORed** — satisfying any one grants access.

So requiring *both* an API key **and** an OAuth2 scope is two keys under a single
sequence item:

{{< example go="concepts/security/routes.go" goregion="routes" golabel="swagger:route"
            json="concepts/security/testdata/and.json" jsonlabel="security on archiveReport (AND)" >}}

A route's requirements replace the document default for that operation. To make
one operation **public** — opting out of the document-wide default — give it an
empty `Security: []`. That emits an explicit empty requirement (distinct from
omitting the keyword, which *inherits* the default):

{{< example go="concepts/security/routes.go" goregion="routes" golabel="swagger:route"
            json="concepts/security/testdata/public.json" jsonlabel="security on publicReport" >}}

The same works from a `swagger:operation` YAML body — a `security:` key there
sets that operation's requirement. (The *schemes* themselves are always global
`swagger:meta` — OpenAPI 2.0 has no per-operation `securityDefinitions`.)

## Keep security out of your code

Authentication is often handled by a layer in front of the app — a gateway or
service mesh — and you may not want security details in the annotations at all.
In that case, leave the code free of security annotations and **overlay** the
schemes and requirements with `Options.InputSpec`:

```go
// base carries only the security scheme + default requirement.
var base spec.Swagger
_ = json.Unmarshal(baseSpecJSON, &base)

doc, _ := codescan.Run(&codescan.Options{
    Packages:   []string{"./..."},
    ScanModels: true,
    InputSpec:  &base, // securityDefinitions + security come from here
})
```

The app package above (`concepts/routes`) carries **no** security annotations,
yet the merged document is secured — the schemes and the default requirement
come entirely from the base:

{{< code file="concepts/security/testdata/overlay.json" lang="json" >}}

See [Overlaying a spec]({{% relref "/shaping-the-output/overlaying-a-spec" %}})
for the full `InputSpec` merge semantics.

## What's next

- [Document metadata]({{% relref "/tutorials/document-metadata" %}}) — the other
  top-level `swagger:meta` fields.
- [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}) — the
  operations these requirements protect.
