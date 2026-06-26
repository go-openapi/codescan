---
title: Cleaning godoc doc-links
weight: 40
description: |
  Strip godoc doc-link brackets from generated descriptions and recompose
  resolvable links to each schema's exposed name — the CleanGoDoc opt-in.
---

A Go doc comment can use godoc's [doc-link](https://go.dev/doc/comment#doclinks)
syntax — `[Gadget]`, `[Order.CustName]`, reference-style `[text]: url` lines.
Those render as live links in `pkg.go.dev`, but carried verbatim into a spec
`title` / `description` they read as bracket noise, and the bracketed Go
identifier is rarely the name the schema is actually exposed under.

`CleanGoDoc` tidies that up. With the option on, godoc doc-link brackets are
removed and — when a link resolves to a scanned schema — the span is recomposed
to the name that schema is **exposed under**, so the prose stays true to the
generated definitions. It applies **only to godoc-derived prose**; an
author-written
[`swagger:title` / `swagger:description`]({{% relref "overriding-titles-and-descriptions" %}})
override is deliberate text and is never touched.

Each pane below pairs the annotated Go (left) with the exact fragment the scanner
emits (right), from the test-covered
[`docs/examples/shaping/godoclinks`](https://github.com/go-openapi/codescan/tree/master/docs/examples/shaping/godoclinks)
package.

## What gets cleaned

This model — its doc comment and its fields — is dense with doc-link syntax: a
self-reference, links to other models, a pointer, a cross-package link, an
unknown identifier, ordinary brackets, and a reference-definition line:

{{< code file="shaping/godoclinks/godoclinks.go" lang="go" region="widget" >}}

Scanned with `CleanGoDoc` off the godoc is emitted verbatim; on, every doc-link
is resolved or humanized and the reference-definition line is dropped:

{{< compare left="shaping/godoclinks/testdata/gizmo_off.json" leftlabel="Default — verbatim"
            right="shaping/godoclinks/testdata/gizmo_on.json" rightlabel="CleanGoDoc — cleaned" >}}

Reading the cleaned pane:

- **Links recompose to the exposed name.** `[Gadget]` → `Gadget` (no override, so
  its Go name); the `[Order.CustName]` member link → `Order.customer_name` (the
  model name plus the field's `json` name); and the leading self-name `Widget`
  → `Gizmo`, because the model is published as `swagger:model gizmo` (restored to
  sentence case). A cross-package `[inventory.Ledger]` resolves through the
  file's imports to `Ledger`.
- **Unresolved links are humanized.** `[Sprocket]` names no scanned model, so it
  becomes the plain word `sprocket` rather than a dangling bracket.
- **Reference-definition lines are dropped.** The `[the spec]: https://…` line on
  the `spec` field is link plumbing carrying no prose, so the whole line is
  removed.

{{% notice style="info" %}}
**It recomposes to the *final* exposed name.** The substitution runs after
codescan resolves definition names, so a link to a model that gets
[renamed to deconflict a collision]({{% relref "resolving-name-conflicts" %}})
points at the renamed definition, not the original Go identifier.
{{% /notice %}}

## Conservative by design

Only a genuine doc-link is rewritten — a dotted chain (`[pkg.Type]`) or an
uppercase-led identifier (`[Widget]`). Ordinary prose brackets are left exactly
as written, as the `index` field above shows: `[0]`, `[see notes]` and the
bare-lowercase `[id]` all survive untouched.

`CleanGoDoc` is opt-in and defaults to off — with it off, output is
byte-identical to before, so existing specs never shift under you.

## What's next

- [Overriding titles & descriptions]({{% relref "overriding-titles-and-descriptions" %}})
  — replace the godoc text outright (overrides are never cleaned).
- [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}})
  — the exposed-name resolution that doc-links recompose to.
