---
title: Descriptions beside a $ref
weight: 50
description: |
  Keep a field's description when its type resolves to a $ref, by wrapping the
  reference in a single-arm allOf (DescWithRef).
---

When a struct field's only decoration is a description and its Go type resolves
to a named model (a `$ref`), JSON Schema draft 4 cannot carry a sibling
`description` next to a `$ref`. `Options.DescWithRef` decides what happens to
that description.

{{< code file="shaping/descref/descref.go" lang="go" region="model" >}}

By default the description is dropped (a bare `$ref`); with `DescWithRef` it is
preserved by wrapping the `$ref` in a single-arm `allOf`:

{{< compare left="shaping/descref/testdata/off.json" leftlabel="Default — description dropped"
            right="shaping/descref/testdata/on.json" rightlabel="DescWithRef: true" >}}

```go
codescan.Run(&codescan.Options{
    Packages:    []string{"./..."},
    ScanModels:  true,
    DescWithRef: true,
})
```

{{% notice style="info" %}}
When the field carries **more** than a description — a validation override or a
user-authored extension — the `allOf` wrapper is emitted **regardless** of this
flag, because the override would otherwise be lost. `DescWithRef` only governs
the description-only case.
{{% /notice %}}
