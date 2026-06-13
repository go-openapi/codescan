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
`deprecated` is an **operation-level** flag in OpenAPI 2.0. The Schema object has
no native `deprecated`, so `deprecated:` on a model field is not emitted on the
property (and produces no `x-deprecated`).
{{% /notice %}}

## What's next

- [Validations]({{% relref "/tutorials/validations" %}}) — the value-constraint
  keywords.
- [Document metadata]({{% relref "/tutorials/document-metadata" %}}) — the
  top-level spec fields.
