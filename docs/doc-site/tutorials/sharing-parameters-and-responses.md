---
title: Sharing parameters & responses
weight: 25
description: |
  Declare a parameter or response once and reuse it across operations through
  the spec-level shared namespace, with the wildcard swagger:parameters and
  swagger:response forms.
---

When the same header, query parameter or error response appears on many
operations, you don't have to repeat it. codescan can publish a parameter or
response **once** into the spec's top-level `parameters` / `responses` maps, then
reference it from each operation as a `$ref`. This is the OpenAPI 2.0 shared
namespace (`#/parameters/{name}`, `#/responses/{name}`).

Each pane below pairs the annotated Go (left) with the exact fragment the scanner
emits (right), from the test-covered
[`docs/examples/concepts/sharedparams`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/sharedparams)
package. For the per-operation basics this builds on — the plain
`swagger:parameters <operationID>` and `swagger:response <name>` forms — see
[Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}).

## Declaring a shared parameter

`swagger:parameters *` declares a struct whose fields are registered at the spec
top level, `#/parameters/{name}`, keyed by each parameter's resolved name. The
bare `*` is **register-only**: it publishes the parameter but does not, by
itself, attach it to any operation.

The convenience form `swagger:parameters * <operationID>…` does both at once — it
registers the parameter **and** `$ref`s it into the listed operations, which is
handy for a small spec.

{{< code file="concepts/sharedparams/sharedparams.go" lang="go" region="shared" >}}

Both structs land in the top-level `parameters` map, each keyed by its parameter
name (the `json:` tag, or a `name:` / `swagger:name` override):

{{< code file="concepts/sharedparams/testdata/parameters.json" lang="json" >}}

## Referencing a shared parameter

Once a parameter is registered, an operation can pull it in by name. There are
two reference channels:

- **`swagger:parameters * <operationID>`** on the declaring struct — the
  convenience form above. `AuthHeader` uses it to inject `X-API-Key` into
  `createPet`.
- **`swagger:parameters <operationID> <name>…`** as a standalone marker on the
  operation's function — the *scaling* channel: the shared struct need not
  enumerate every operation that wants the parameter; instead each operation
  opts in next to its own `swagger:route`.

{{< code file="concepts/sharedparams/sharedparams.go" lang="go" region="routes" >}}

`listPets` opts in through the standalone marker, so its only parameter is a
`$ref` to the shared `X-Request-ID`:

{{< code file="concepts/sharedparams/testdata/listpets.json" lang="json" >}}

`createPet` receives the `$ref`'d `X-API-Key` (from `* createPet`) **alongside**
its own inlined body parameter — references and inline parameters coexist:

{{< code file="concepts/sharedparams/testdata/createpet.json" lang="json" >}}

## Path-item parameters

A parameter can also attach to a whole **path** rather than a single operation.
`swagger:parameters /path` inlines a struct's fields into the path-item's
`parameters` array, so every operation under that path inherits them.

{{< code file="concepts/sharedparams/sharedparams.go" lang="go" region="pathitem" >}}

`X-Tenant` now rides the `/pets/{id}` path-item; `getPet` inherits it without
declaring a header of its own:

{{< code file="concepts/sharedparams/testdata/pathitem.json" lang="json" >}}

{{% notice style="warning" %}}
**Exact path, no hierarchy.** OpenAPI 2.0 has no path nesting, so the target is
matched literally: `swagger:parameters /pets/{id}` applies to `/pets/{id}` only —
**not** to `/pets`. Path-item parameters also *co-exist* with operation-level
ones rather than replacing them; if an operation declares a parameter with the
same `(name, in)`, the operation's wins at resolution time per the OAS2 rule.
{{% /notice %}}

To `$ref` an already-registered shared parameter into a path-item (instead of
inlining a new one), use the reference form with a path target:
`swagger:parameters /path <name>…` — the path-item analogue of the per-operation
marker above.

## Shared responses

Responses share the same way. `swagger:response *` registers a struct at
`#/responses/{name}` (keyed by the Go type name). The `*` is a synonym for the
bare/named `swagger:response` form — its job is to mark the response as a
*shared* one. Operations then name it in their `Responses:` block and it is
emitted as a `$ref`.

{{< code file="concepts/sharedparams/sharedparams.go" lang="go" region="sharedresponse" >}}

The shared `ErrorResponse` lands in the top-level `responses` map:

{{< code file="concepts/sharedparams/testdata/responses.json" lang="json" >}}

Both routes write `default: ErrorResponse`, which resolves to a single shared
`$ref` (visible as `responses.default.$ref` in the operation panes above) —
one error envelope, defined once, referenced everywhere.

## Conflicts, duplicates & dangling references

The shared namespace is referenced **only by short name**, so codescan cannot
silently rename a collision the way it
[deconflicts model definitions]({{% relref "/shaping-the-output/resolving-name-conflicts" %}}).
Instead it applies a deterministic, observable policy and reports every
adjustment through `Options.OnDiagnostic` (the scan never fails on these — it
keeps a valid spec and warns):

| Situation | Policy | Diagnostic |
|---|---|---|
| Two `swagger:parameters *` register the same name | **keep-first** (sorted by package path then position; never renamed) — later one dropped | `scan.shared-parameter-conflict` |
| Two `swagger:response *` register the same name | keep-first; later one dropped | `scan.shared-response-conflict` |
| A reference names a parameter no `*` registered | reference dropped (no dangling `$ref` emitted) | `scan.dangling-parameter-ref` |
| An operation names an unregistered shared response | reference dropped | `scan.dangling-response-ref` |
| A `* <opid>…` marker repeats an operation id | duplicate dropped | `scan.duplicate-target` |
| A reference repeats a parameter name | collapses to a single `$ref` | `scan.duplicate-ref` |

{{% notice style="note" %}}
The shared **parameters**, **responses** and **definitions** namespaces are
independent: `#/parameters/Status`, `#/responses/Status` and
`#/definitions/Status` can all coexist. The resolved (`name:`-overridden) name is
the key, so references must use that name — not the Go field name. An `InputSpec`
overlay entry seeds the namespace and wins any keep-first conflict.
{{% /notice %}}

## What's next

- [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}) — the
  per-operation `swagger:parameters` / `swagger:response` basics.
- [Pruning unused models]({{% relref "/shaping-the-output/pruning-unused-models" %}})
  — shared parameters and responses count as reachability roots when pruning.
- [Keyword reference]({{% relref "/maintainers/keywords" %}}) — the exhaustive
  `parameters` / `responses` body grammars.
