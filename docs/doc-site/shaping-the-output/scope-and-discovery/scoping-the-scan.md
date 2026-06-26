---
title: Scoping the scan
weight: 10
description: |
  Limit what gets scanned — package patterns, working directory, include/exclude
  filters, tag filters, and dependency handling.
---

Several options narrow *what* codescan looks at, independent of *how* individual
types render. They decide which packages are loaded and which discovered
operations survive into the spec.

## Package patterns and WorkDir

`Options.Packages` takes relative `go list`-style patterns — `./petstore`,
`./...` for a whole tree — resolved against `Options.WorkDir` (the module root).
This is the worked form in the [Getting started]({{% relref "/getting-started/usage-as-a-library" %}})
guide:

```go
codescan.Run(&codescan.Options{
    WorkDir:    "/path/to/module",
    Packages:   []string{"./..."},
    ScanModels: true,
})
```

To produce **several specs** from one module — e.g. one per API version — run a
scan per package tree (`./v1/...`, then `./v2/...`) and write each result
separately. There is no single-run "split by version"; the unit of a scan is the
set of packages you pass.

## Include / Exclude

`Options.Include` and `Options.Exclude` are lists of regular expressions matched
against **package import paths**. Include acts as an allow-list (when non-empty,
only matching packages are scanned); Exclude removes matches. Use them to keep
internal or generated packages out of the spec:

```go
codescan.Run(&codescan.Options{
    Packages: []string{"./..."},
    Exclude:  []string{"/internal/", "/testdata/"},
})
```

## Tag filters

`Options.IncludeTags` / `Options.ExcludeTags` filter **operations** by their
Swagger tags after discovery — handy for publishing a public subset of an API
while keeping the admin routes in the source:

```go
codescan.Run(&codescan.Options{
    Packages:    []string{"./..."},
    ExcludeTags: []string{"admin", "internal"},
})
```

## ExcludeDeps

By default codescan may follow types into dependency packages to resolve
referenced models. `Options.ExcludeDeps` keeps the scan within your own module,
leaving out types pulled in from dependencies.

Build constraints get their own guide — see
[Build tags]({{% relref "build-tags" %}}).
