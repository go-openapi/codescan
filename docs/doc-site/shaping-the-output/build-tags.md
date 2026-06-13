---
title: Build tags
weight: 70
description: |
  Scan source guarded by Go build constraints by passing build tags to the
  scanner.
---

Go files can be guarded by `//go:build` constraints. By default codescan loads a
package under the default build configuration, so tag-gated files are skipped.
`Options.BuildTags` passes the tags through to the package loader, so the
annotations in those files are scanned too.

The package has an always-present model plus this one, in a file that opens with
the constraint `//go:build experimental`:

{{< code file="shaping/buildtags/experimental.go" lang="go" region="experimental" >}}

Scanned with no tags and with `experimental`, the gated `Experimental` model
appears only in the second:

{{< compare left="shaping/buildtags/testdata/off.json" leftlabel="Default"
            right="shaping/buildtags/testdata/on.json" rightlabel="BuildTags: experimental" >}}

```go
codescan.Run(&codescan.Options{
    Packages:   []string{"./..."},
    ScanModels: true,
    BuildTags:  "experimental",
})
```

`BuildTags` accepts the same comma-separated form as `go build -tags`.
