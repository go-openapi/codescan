---
title: "go-openapi codescan"
type: home
description: Generate Swagger 2.0 specifications from annotated Go source code.
weight: 1
---

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

### Where to go next

{{< cards >}}
{{% card title="Getting started" %}}
Install the scanner, annotate a package, and produce your first spec.

→ [getting-started]({{% relref "/getting-started" %}})
{{% /card %}}

{{% card title="Tutorials" %}}
Learn by spec concept — model definitions, routes, validations — annotated Go
next to the spec it produces.

→ [tutorials]({{% relref "/tutorials" %}})
{{% /card %}}

{{% card title="Shaping the output" %}}
How-to guides for the rendering knobs: $ref vs inline, aliases, nullable
pointers, extensions.

→ [shaping-the-output]({{% relref "/shaping-the-output" %}})
{{% /card %}}

{{% card title="Annotation index" %}}
Every annotation at a glance, linked to its worked example and its full
reference.

→ [annotation-index]({{% relref "/annotation-index" %}})
{{% /card %}}

{{% card title="Reference (maintainers)" %}}
The complete compendium — annotations, keywords, sub-languages and the formal
grammar.

→ [maintainers]({{% relref "/maintainers" %}})
{{% /card %}}

{{% card title="Project" %}}
Repo overview, license and links to the shared go-openapi guides.

→ [project]({{% relref "/project" %}})
{{% /card %}}
{{< /cards >}}

## Licensing

`SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers`

This library ships under the [Apache-2.0 license](./project/license/).

## Contributing

Issues and pull requests welcome.

See the shared [go-openapi contributing guidelines][contributing-doc-site] and
the per-repo notes in [project/](./project/).

---

{{< children type="card" description="true" >}}

[swagger2]: https://swagger.io/specification/v2/
[contributing-doc-site]: https://go-openapi.github.io/doc-site/contributing/contributing/index.html
[maintainers-doc-site]: https://go-openapi.github.io/doc-site/maintainers/index.html
