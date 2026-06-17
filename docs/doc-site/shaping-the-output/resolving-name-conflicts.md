---
title: Resolving $ref name conflicts
weight: 18
description: |
  When two Go types want the same definition name, codescan keeps them distinct
  with deterministic, package-qualified names — and you stay in control of the
  $ref names that form your published contract.
---

A Swagger definition is keyed by a single short name (`#/definitions/Account`),
but a Go program routinely has *several* types that would map to that name —
the same leaf declared in different packages, or a `swagger:model Account`
override applied twice. codescan keys every definition by a compiler-unique
identity (`<package-path>/<name>`) while it builds, then a final reduce stage
projects each identity back to the shortest name that is still unique. The
result is deterministic regardless of discovery or map-iteration order: no
silent overwrite, no lost definition.

The panes below are backed by the test-covered
[`docs/examples/shaping/nameconflicts`](https://github.com/go-openapi/codescan/tree/master/docs/examples/shaping/nameconflicts)
package tree.

## When names collide

Two packages each declare an `Account`, with entirely different fields:

{{< code file="shaping/nameconflicts/billing/account.go" lang="go" region="billing" >}}

{{< code file="shaping/nameconflicts/identity/account.go" lang="go" region="identity" >}}

A `Dashboard` model references both, so they are discovered together:

{{< code file="shaping/nameconflicts/doc.go" lang="go" region="dashboard" >}}

Before name-identity, the two would have merged onto a single
`#/definitions/Account` — a union of fields, last package wins,
non-deterministically. Now each keeps its own definition and the references
resolve to the deconflicted names:

{{< code file="shaping/nameconflicts/testdata/dashboard.json" lang="json" >}}

## How codescan resolves them automatically

The reduce stage gives every reachable identity the shortest acceptable name:

- A **globally unique** leaf is lifted to its bare name — byte-identical to the
  pre-feature output, so the common case sees zero churn.
- A **colliding** leaf is qualified with the minimal-depth PascalCase concat of
  its nearest package segments (`billing.Account` / `identity.Account` →
  `BillingAccount` / `IdentityAccount`), deepening one segment at a time until
  the whole group is unique. A `validate.colliding-model-name` diagnostic
  records each rename.

Every emitted definition also carries an `x-go-package` extension recording the
source package, so even identically-shaped collisions stay traceable:

{{< code file="shaping/nameconflicts/testdata/billingaccount.json" lang="json" >}}

{{% notice style="info" %}}
The whole pass is a pure function of the reachable identity set, so the names
are **stable across runs** — but they are *derived from your package paths*.
Renaming or moving a package changes the segment used in a qualified name. Pin
the names that matter (see [below](#keeping-the-exposed-names-under-your-control)).
{{% /notice %}}

## Same-package duplicates

A single package cannot own a definition name twice. If two Go types in the
same package both claim `swagger:model Entry`, codescan keeps one
(deterministically) and reverts the other to its Go type name, with a
`validate.duplicate-model-name` diagnostic:

{{< code file="shaping/nameconflicts/ledger/ledger.go" lang="go" region="dup" >}}

Here `Entry` keeps the contested name and `Reversal` falls back to its Go name —
the `Dashboard` refs above point at `#/definitions/Entry` and
`#/definitions/Reversal`, never a merged `Entry`. This is a genuine authoring
error (one package, one name); the fallback keeps the spec valid rather than
silently dropping a model.

## Referencing a model by leaf across packages

The type-name keywords — `swagger:type`, `swagger:additionalProperties`, and
`swagger:patternProperties` — accept a **bare leaf** as their argument. codescan
resolves it the same way the reduce stage does: the annotating type's own
package first, then uniquely across the scanned model set. A leaf unique in
another package resolves to a `$ref`:

{{< example go="shaping/nameconflicts/doc.go" goregion="bag" golabel="swagger:additionalProperties Widget"
            json="shaping/nameconflicts/testdata/bag.json" jsonlabel="#/definitions/Bag" >}}

If the leaf matches a model in **several** packages it is ambiguous: the
reference is dropped (never guessed) and a `validate.ambiguous-type-name`
diagnostic is raised. Disambiguate with a same-package type or pin the target
with a `swagger:model <Name>` override. The same leaf rule applies to the
`additionalProperties:` / `swagger:patternProperties` value forms covered in
[Maps & free-form objects]({{% relref "/tutorials/maps-and-free-form-objects" %}}).

## Keeping the exposed names under your control

The generated `$ref` names are part of your published contract, so the author —
not the resolver — should decide the ones that matter:

- **Pin a public name** with an explicit `swagger:model <Name>`. A pinned name
  is the identity's leaf, so two pinned names that still collide are deconflicted
  by package segment exactly like inferred ones — pin *distinct* names for the
  types in your public surface.
- **Let auto-resolution handle the rest.** Incidental or internal collisions get
  a valid, stable, package-qualified name with no action from you.

### Tuning the qualified names

Two scanner options steer the rare, deep collisions:

| Option | Default | Effect |
|---|---|---|
| `NameConcatBudget` | `0.65` | Readability cutoff in `[0,1]` (lower is more readable). A collision group whose best flat concat scores *above* the budget becomes a candidate for the hierarchical fallback. Raise toward `1.0` to accept longer concats; lower to fall back sooner. |
| `EmitHierarchicalNames` | `false` | Opt into the fallback: over-budget groups are emitted as nested container definitions (`#/definitions/<pkg>/<Name>`, each tagged with `x-go-package`) instead of a long flat concat. |

{{% notice style="warning" %}}
`EmitHierarchicalNames` is off by default on purpose. A nested definition is a
deep JSON pointer that only `ExpandSpec` resolves, and a definitions-enumerating
consumer (e.g. go-swagger codegen, one model per entry) sees the container nodes
rather than the models. The always-correct flat concat stays the default; enable
the nested shape only when you prefer it for the over-budget tail.
{{% /notice %}}

## When to tune vs. let it auto-resolve

- **Pin** the names that appear in your public API contract — clients,
  generated SDKs, and hand-written `$ref`s depend on them.
- **Let auto-resolution handle** incidental collisions between internal types;
  the qualified names are valid and stable.
- **Reach for `EmitHierarchicalNames`** only when a few collision groups have
  package names long enough to make the flat concat unwieldy, *and* your
  consumers resolve `$ref` pointers (rather than enumerating definitions).

## What's next

- [Type discovery]({{% relref "/shaping-the-output/type-discovery" %}}) — which
  types become definitions in the first place.
- [Maps & free-form objects]({{% relref "/tutorials/maps-and-free-form-objects" %}}) —
  the typed `additionalProperties` / `patternProperties` value forms.
- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) —
  `swagger:model` and the per-type annotations.
