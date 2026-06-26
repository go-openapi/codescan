---
title: Interface-method property names
weight: 25
description: |
  Emit interface-method property names verbatim (ID, CreatedAt) instead of the
  auto-jsonified spelling (id, createdAt), with SkipJSONifyInterfaceMethods.
---

When a model's shape is described by an **interface**, its methods have no
natural JSON serialization — Go's `encoding/json` can't marshal interface
methods, so there's no struct tag to read a name from. codescan invents one by
running its jsonify transform on the Go method name: `ID` → `id`, `CreatedAt`
→ `createdAt`. That "one size fits all" convention isn't always what you want —
an interface already named for its JSON shape, or a codebase with its own
canonical-name discipline, wants the Go name kept as-is.

`SkipJSONifyInterfaceMethods` opts out of the mangler. With it set, an
interface-method property is emitted under the Go method name verbatim. It is an
opt-out and defaults to off; with it off, output is unchanged.

## What changes

This model is an interface with two default-path methods and one carrying a
`swagger:name` override:

{{< code file="shaping/interfacenames/interfacenames.go" lang="go" region="account" >}}

Scanned with the flag off the method names auto-jsonify; on, they ride through
verbatim:

{{< compare left="shaping/interfacenames/testdata/account_off.json" leftlabel="Default — jsonified"
            right="shaping/interfacenames/testdata/account_on.json" rightlabel="SkipJSONifyInterfaceMethods — verbatim" >}}

Reading the two panes:

- **Default-path methods are jsonified.** `ID()` → `id`, `CreatedAt()` →
  `createdAt`; the original Go name is preserved as the `x-go-name` extension.
- **With the opt-out, the Go name is the property name.** `ID` and `CreatedAt`
  appear verbatim — and `x-go-name` drops, since it would now just repeat the
  property name.
- **A `swagger:name` override is verbatim either way.** `OverriddenField` is
  published as `explicit_name` in both panes — the override already bypasses the
  mangler, so the flag never touches it (and never re-mangles it to
  `explicitName`).

{{% notice style="note" %}}
This flag only affects **interface methods**, which have no JSON serialization to
mirror. Struct-field property names are untouched — they always reflect what
`encoding/json` actually produces (see
[Naming from struct tags]({{% relref "naming-from-tags" %}}) to source those from
a different tag).
{{% /notice %}}

## What's next

- [Naming from struct tags]({{% relref "naming-from-tags" %}}) — choose which
  struct tag a *field* name comes from (the struct-field analogue of this knob).
- [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}) —
  how the definition names themselves are kept distinct.
