---
title: Getting started
weight: 10
description: |
  Install codescan and choose how to drive it.

  As a Go library from your own program, as a command in a build, or interactively from the terminal UI.
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

---

Or as a command, to run in a build or a pipeline:

```cmd
go install github.com/go-openapi/codescan/cmd/genspec@latest
```

---

Or as a terminal front-end, to watch a spec take shape as you annotate:

```cmd
go install github.com/go-openapi/codescan/cmd/genspec-tui@latest
```

---

If you just want to experiment, learn or reproduce an issue you're currently having,
the easiest way is to try our [Playground]({{% relref "/playground" %}}) in your browser.

## Ways to use codescan

{{< children type="card" description="true" >}}

> All three drive the same scanner over the same annotations, and every knob is
> spelled the same way in each, so the terminal UI shows exactly the document
> your build will produce.
>
> See [Setting an option]({{% relref "setting-options" %}}).
