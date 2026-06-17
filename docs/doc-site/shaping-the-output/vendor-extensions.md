---
title: Vendor extensions
weight: 40
description: |
  Control the x-go-* vendor extensions codescan emits, or suppress them with
  SkipExtensions.
---

By default codescan records where each spec object came from in Go via
`x-go-name` (and `x-go-package` on definitions) — useful for round-tripping and
code generation. `Options.SkipExtensions` removes them for a leaner spec.

{{< code file="shaping/extensions/extensions.go" lang="go" region="model" >}}

{{< compare left="shaping/extensions/testdata/off.json" leftlabel="Default"
            right="shaping/extensions/testdata/on.json" rightlabel="SkipExtensions: true" >}}

```go
codescan.Run(&codescan.Options{
    Packages:       []string{"./..."},
    ScanModels:     true,
    SkipExtensions: true,
})
```

`SkipExtensions` removes the scanner-derived `x-go-*` extensions. Extensions you
author yourself (via the `Extensions:` keyword) are not affected, and neither is
`x-deprecated` (it carries semantic intent — see
[Other type decorators]({{% relref "/tutorials/other-type-decorators" %}})).

## Stamping `x-go-type`

`x-go-name` and `x-go-package` record *where* a definition came from, but not the
originating type's own name. `Options.EmitXGoType` adds an `x-go-type` extension
carrying the fully-qualified Go type (`<package path>.<type name>`) — useful for
round-tripping a generated spec back to its source types:

```go
codescan.Run(&codescan.Options{
    Packages:    []string{"./..."},
    ScanModels:  true,
    EmitXGoType: true,
})
```

{{< compare left="shaping/extensions/testdata/off.json" leftlabel="Default — no x-go-type"
            right="shaping/extensions/testdata/xgotype.json" rightlabel="EmitXGoType: true" >}}

The stamp lands on the **definition**, beside `x-go-package`. It is opt-in and
default-off, so existing specs are unchanged; it is presence-guarded, so it never
overwrites the deliberate `x-go-type` the special-type recognizers already set
(`error`, the unmodellable generic-type fallback). Like the other `x-go-*`
extensions it rides the `SkipExtensions` umbrella — set `SkipExtensions` and no
`x-go-type` is emitted either.

## Enum descriptions

A `swagger:enum` type backed by Go `const` declarations folds the const→value
mapping into the field's `description` **and** duplicates it in the
`x-go-enum-desc` extension. When the prose already says everything you want, the
folded mapping is noise. `Options.SkipEnumDescriptions` keeps the authored prose
as the description; the mapping then rides `x-go-enum-desc` only:

```go
codescan.Run(&codescan.Options{
    Packages:             []string{"./..."},
    ScanModels:           true,
    SkipEnumDescriptions: true,
})
```

This knob is independent of `SkipExtensions`: set both to drop the mapping
everywhere (no description folding, no `x-go-enum-desc`).

## Authoring `x-*` on parameters and headers

The `x-go-*` extensions above are scanner-derived. To attach your **own** vendor
extension — say `x-example` for a tool like Dredd — use an `Extensions:` block in
the doc comment. It works on a **model** (the `x-*` lands on the definition), a
**model field**, a **parameter**, and a **response header** alike. (A bare
`// x-example: 2` line would be read as the description; the `Extensions:` block
is the supported form.)

{{< code file="shaping/extensions/extensions.go" lang="go" region="paramext" >}}

{{< code file="shaping/extensions/testdata/paramext.json" lang="json" >}}

Author-supplied extensions are not stripped by `SkipExtensions` — the fragment
above is produced *with* `SkipExtensions: true`, yet `x-example` and `x-units`
survive, because the flag only removes the scanner-derived `x-go-*` set.
