---
title: Nullable pointers
weight: 20
description: |
  Mark pointer-typed fields as nullable with x-nullable, via
  SetXNullableForPointers.
---

Swagger 2.0 has no native nullable flag; the go-openapi toolchain uses the
`x-nullable` vendor extension. `Options.SetXNullableForPointers` decides whether
pointer-typed struct fields acquire it automatically. The model below has two
pointer fields:

{{< code file="shaping/nullable/nullable.go" lang="go" region="model" >}}

Scanned with the option off (default) and on, the pointer fields differ:

{{< compare left="shaping/nullable/testdata/off.json" leftlabel="Default"
            right="shaping/nullable/testdata/on.json" rightlabel="SetXNullableForPointers: true" >}}

```go
codescan.Run(&codescan.Options{
    Packages:                []string{"./..."},
    ScanModels:              true,
    SetXNullableForPointers: true,
})
```

{{% notice style="info" %}}
**`omitempty` changes the meaning.** A pointer field tagged
`json:"…,omitempty"` is treated as *optional* (may be absent) rather than
*nullable* (may be `null`), so it does **not** receive `x-nullable` even with the
option on. Drop `omitempty` when you mean the value can be present-but-null.
{{% /notice %}}
