---
title: Other type decorators
weight: 45
description: |
  Mark a property read-only, and an operation deprecated.
---

Beyond validations, a couple of keyword decorators annotate a property's or
operation's *role*. The panes below pair the annotated Go with the fragment the
scanner emits, from the test-covered
[`docs/examples/concepts/decorators`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/decorators)
package.

For the value shapes and legal contexts of each, see the
[Keyword reference]({{% relref "/maintainers/keywords" %}}).

## readOnly

`read only: true` on a model field marks the property `readOnly` — the server
sets it, clients must not.

{{< example go="concepts/decorators/decorators.go" goregion="readonly"
            json="concepts/decorators/testdata/readonly.json" jsonlabel="#/definitions/Token" >}}

## deprecated

`deprecated: true` in a `swagger:route` / `swagger:operation` body marks the
**operation** deprecated.

{{< example go="concepts/decorators/decorators.go" goregion="deprecated"
            json="concepts/decorators/testdata/deprecated.json" jsonlabel="paths[/legacy/ping]" >}}

{{% notice style="info" %}}
On an **operation**, `deprecated: true` sets the native OpenAPI 2.0
`deprecated` field. OpenAPI 2.0 has no native `deprecated` on the Schema object,
so on a **model or model field** codescan emits the `x-deprecated: true` vendor
extension instead.

A godoc-style `Deprecated:` paragraph (the pkgsite convention) is an exact
**synonym** for `deprecated: true`, recognised in any context. On a Go *doc
comment* it is the natural form — a bare `// deprecated: true` line there reads
as a malformed deprecation notice to Go linters, whereas the capitalised
`Deprecated:` paragraph is idiomatic. Use `deprecated: true` in the indented
route / operation bodies, and the `Deprecated:` paragraph on model and field doc
comments; either yields the same result. `x-deprecated` carries semantic intent
rather than reflection metadata, so it is emitted even when `SkipExtensions` is
set.
{{% /notice %}}

A godoc `Deprecated:` paragraph marks a model and its fields — codescan emits
`x-deprecated: true` on each (the explicit `deprecated: true` annotation has the
same effect):

{{< example go="concepts/decorators/decorators.go" goregion="deprecatedmodel"
            json="concepts/decorators/testdata/deprecated_model.json" jsonlabel="#/definitions/Gadget" >}}

## What's next

- [Validations]({{% relref "/tutorials/validations" %}}) — the value-constraint
  keywords.
- [Document metadata]({{% relref "/tutorials/document-metadata" %}}) — the
  top-level spec fields.
