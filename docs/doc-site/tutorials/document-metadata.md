---
title: Document metadata
weight: 50
description: |
  Set the top-level spec fields — title, version, host, basePath, schemes,
  consumes/produces, license and contact — from the package doc comment.
---

A single `swagger:meta` block on a package doc comment carries the document's
top-level metadata: its `info` (title, description, version, license, contact),
the `host` and `basePath`, the default `schemes`, and `consumes`/`produces`. The
pane pairs the annotated package with the document it produces, from the
test-covered [`docs/examples/concepts/meta`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/meta)
package.

## swagger:meta

The block lives in the package doc comment. The **title** comes from the first
line with the `Package <name>` prefix stripped; the following paragraph becomes
the `description`. The indented `Key: value` lines and list blocks populate the
rest — `License:` and `Contact:` parse into structured objects.

{{< example go="concepts/meta/doc.go" goregion="meta" golabel="Package doc comment"
            json="concepts/meta/testdata/meta.json" jsonlabel="the document" >}}

For the full meta keyword surface (security definitions, external docs,
extensions, terms of service), see the
[`swagger:meta` reference]({{% relref "/maintainers/annotations#swaggermeta" %}})
and the [meta keywords]({{% relref "/maintainers/keywords#meta-single-line-keywords" %}}).

## What's next

- [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}) — add
  the paths this document describes.
- [Putting it together]({{% relref "/tutorials/putting-it-together" %}}) — a
  complete scan from meta to definitions.
