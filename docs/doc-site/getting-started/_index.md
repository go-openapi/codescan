---
title: Getting started
weight: 10
description: |
  Install codescan and choose how to drive it — as a Go library from your own
  program, or interactively from the terminal UI.
---

## Install

As a library, to call from your own program:

```cmd
go get github.com/go-openapi/codescan
```

codescan exposes a deliberately small surface: a single `Run` function and an
`Options` struct.

```go
func Run(opts *Options) (*spec.Swagger, error)
```

Or as a terminal front-end, to watch a spec take shape as you annotate:

```cmd
go install github.com/go-openapi/codescan/cmd/genspec-tui@latest
```

## Ways to use codescan

{{< children type="card" description="true" >}}

> Both drive the same scanner over the same annotations, so what you see in the
> terminal UI is what your generator will produce.
