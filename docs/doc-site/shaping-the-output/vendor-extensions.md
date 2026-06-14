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
