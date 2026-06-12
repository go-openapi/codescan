---
title: Getting started
weight: 1
description: |
  Install codescan and choose how to drive it. Today the scanner is consumed
  as a Go library; more usage modes will be added here as siblings.
---

## Install

```cmd
go get github.com/go-openapi/codescan
```

codescan exposes a deliberately small surface: a single `Run` function and an
`Options` struct.

```go
func Run(opts *Options) (*spec.Swagger, error)
```

## Ways to use codescan

{{< children type="card" description="true" >}}

> Today, codescan is used as a Go library (below). Additional usage modes will
> appear here as siblings as the toolkit grows.
