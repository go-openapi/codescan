---
title: "go-openapi codescan"
type: home
description: Generate Swagger 2.0 specifications from annotated Go source code.
weight: 1
---

{{< version >}}

`github.com/go-openapi/codescan` is a Go source code scanner that produces
[Swagger 2.0][swagger2] (OpenAPI 2.0) specifications.

It reads specially formatted comments (annotations) in Go source files and
extracts API metadata — routes, parameters, responses, schemas and more — to
build a complete `spec.Swagger` document. It supports Go modules (since
go1.11).

The scanner works entirely at the AST / `go/types` level: it **never compiles
or executes** the code it scans. It only reads the source and its annotation
comments.

### Status

{{% button href="https://github.com/go-openapi/codescan/fork" hint="fork me on github" style=primary icon=code-fork %}}Fork me{{% /button %}}
Stable API. Actively maintained.

The only exposed API is `Run()` and `Options`.

### Getting started

To use codescan in your go program:

```cmd
go get github.com/go-openapi/codescan
```

Point the scanner at one or more packages and get back a `*spec.Swagger`:

```go
import "github.com/go-openapi/codescan"

swaggerSpec, err := codescan.Run(&codescan.Options{
    Packages: []string{"./..."},
})
```

Or as a command, to run in a build or a pipeline:

```cmd
go install github.com/go-openapi/codescan/cmd/genspec@latest
```

Or as a terminal front-end, to watch a spec take shape as you annotate:

```cmd
go install github.com/go-openapi/codescan/cmd/genspec-tui@latest
```

Try it out now from your browser in our [Playground]({{% relref "/playground" %}}).

### Relationship to go-swagger

`go-swagger` is a CLI tool that consumes the codescan library.
It works exactly on the same set of annotations.

The main differences with the newer `genspec` CLI shipped by this project are:

* release cadence (expect slightly less frequent updates on go-swagger, which has more dependencies and constraints)
* package distribution: at this moment, the codescan CLI tools do not ship as distro packages or docker images
* exposed CLI knobs and default settings (defaults need to be backward-compatible for go-swagger users)

`genspec` and `genspec-tui` are intended for users who want tools leaner than go-swagger, or who want to
experiment with the latest features.

### Where to go next

{{< children type="card" description="true" >}}

[swagger2]: https://swagger.io/specification/v2/
