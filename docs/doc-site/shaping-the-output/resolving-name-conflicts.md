---
title: Resolving $ref name conflicts
weight: 18
draft: true
description: |
  When two types want the same definition name, codescan detects the conflict
  and auto-resolves it — and you stay in control of the exposed $ref names.
---

<!--
DRAFT — scaffold for the feat/name-identity-cyclic-ref feature (#2637, #2783),
landing soon. Flip `draft: false` and add the test-backed example panes once the
feature is on the base branch. This page is the author-facing "how to tune it"
counterpart to the Same-name collisions note in
[Model definitions]({{% relref "/tutorials/model-definitions" %}}).

Outline of what this section explains (the WHAT, not the HOW):
-->

## When names collide

What counts as a name conflict — two distinct Go types resolving to the same
`#/definitions/<Name>` (e.g. the same short name in different packages), and the
self-referential / cyclic case where a type's `$ref` points back at itself.
Cross-link to [type discovery]({{% relref "/shaping-the-output/type-discovery" %}}).

## How codescan resolves them automatically

What auto-resolution guarantees: every colliding definition gets a distinct,
deterministic name so the spec is valid — no silent overwrite, no lost
definition. (Describe the guarantees the author can rely on, not the internals.)

## Keeping the exposed names under your control

The point of the section: the generated `$ref` names are part of your published
contract, so the author — not the resolver — should decide the ones that matter.
Cover the knobs for pinning / steering names (e.g. an explicit
`swagger:model <Name>`, plus the feature's tuning surface — fill in once landed)
and when to reach for each.

## When to tune vs. let it auto-resolve

Guidance: pin the names that appear in your public API contract; let
auto-resolution handle incidental / internal collisions.
