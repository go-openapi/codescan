---
title: Overlaying a spec
weight: 60
description: |
  Merge scanned discoveries on top of an existing Swagger document with
  InputSpec.
---

`Options.InputSpec` seeds the scan with an existing `*spec.Swagger`: codescan
merges what it discovers on top of it rather than starting from a blank
document. Use it to keep hand-authored top-level metadata or a hand-written
definition, or to compose a spec across several scans.

The scanned package contributes one model:

{{< code file="shaping/overlay/overlay.go" lang="go" region="model" >}}

Given a base document with metadata and a hand-authored `Health` definition,
the scan preserves all of it and adds the discovered `Widget`:

{{< compare left="shaping/overlay/testdata/base.json" leftlabel="InputSpec (base)"
            right="shaping/overlay/testdata/merged.json" rightlabel="After the scan" >}}

```go
var base spec.Swagger
_ = json.Unmarshal(baseSpecJSON, &base)

doc, _ := codescan.Run(&codescan.Options{
    Packages:   []string{"./..."},
    ScanModels: true,
    InputSpec:  &base,
})
```

The document's `info`, `host`, `basePath` and the hand-authored `Health`
definition survive untouched; only the discovered definitions are added.
