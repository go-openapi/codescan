---
title: README
description: Repo overview and announcements.
weight: 1
---

# codescan

A Go source code scanner that produces Swagger 2.0 (OpenAPI 2.0) specifications
from annotated Go source files.

Supports Go modules (since go1.11).

## Announcements

* **2025-04-19**: large package layout reshuffle
  * the entire project is being refactored to restore a reasonable level of
    maintainability
  * the only exposed API is `Run()` and `Options`.

## Status

API is stable.

## Import this library in your project

```cmd
go get github.com/go-openapi/codescan
```

## Basic usage

```go
import (
  "github.com/go-openapi/codescan"
)

swaggerSpec, err := codescan.Run(&codescan.Options{
  Packages: []string{"./..."},
})
```

See [getting started](../../getting-started/) for a worked example.

## Change log

See <https://github.com/go-openapi/codescan/releases>.

## Licensing

This library ships under the [Apache-2.0](../license/) license.

See the license [NOTICE](https://github.com/go-openapi/codescan/blob/master/NOTICE),
which recalls the licensing terms of all the pieces of software on top of which
it has been built.

## Other documentation

* [All-time contributors](https://github.com/go-openapi/codescan/blob/master/CONTRIBUTORS.md)
* [Contributing guidelines][contributing-doc-site]
* [Maintainers documentation][maintainers-doc-site]
* [Code style][style-doc-site]

[contributing-doc-site]: https://go-openapi.github.io/doc-site/contributing/contributing/index.html
[maintainers-doc-site]: https://go-openapi.github.io/doc-site/maintainers/index.html
[style-doc-site]: https://go-openapi.github.io/doc-site/contributing/style/index.html
