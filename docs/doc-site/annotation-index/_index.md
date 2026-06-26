---
title: Annotation index
weight: 40
description: |
  Every swagger:* annotation at a glance — what it produces and where it
  attaches — linked to both its worked example and its full reference.
---

The complete `swagger:*` vocabulary, one row each. **By example** jumps to the
tutorial that shows the annotation as runnable Go next to the spec it produces;
**Reference** jumps to the exhaustive rule in the Maintainers compendium.

| Annotation | Attaches to | Produces | By example | Reference |
|---|---|---|---|---|
| `swagger:meta` | package doc | top-level `info`, `host`, `basePath`, `schemes`, … | [example]({{% relref "/tutorials/document-metadata#swaggermeta" %}}) | [reference]({{% relref "/maintainers/annotations#swaggermeta" %}}) |
| `swagger:model` | type declaration | a `definitions` entry | [example]({{% relref "/tutorials/model-definitions#swaggermodel" %}}) | [reference]({{% relref "/maintainers/annotations#swaggermodel" %}}) |
| `swagger:strfmt` | type declaration | `{type: string, format: …}` at every use | [example]({{% relref "/tutorials/model-definitions#swaggerstrfmt" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerstrfmt" %}}) |
| `swagger:enum` | named type | an `enum` array (+ `x-go-enum-desc`) | [example]({{% relref "/tutorials/model-definitions#swaggerenum" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerenum" %}}) |
| `swagger:allOf` | embedded field / struct | an `allOf` composition | [example]({{% relref "/tutorials/model-definitions#swaggerallof" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerallof" %}}) |
| `swagger:alias` *(deprecated)* | type alias | **no effect** — alias rendering is controlled by Go aliases + options | [how-to]({{% relref "alias-rendering" %}}) | [reference]({{% relref "/maintainers/annotations#swaggeralias--deprecated" %}}) |
| `swagger:route` | func / var doc | a `paths` entry + operation | [example]({{% relref "/tutorials/routes-and-operations#swaggerroute" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerroute" %}}) |
| `swagger:operation` | func / var doc | a `paths` entry (YAML body) | [example]({{% relref "/tutorials/routes-and-operations#swaggeroperation" %}}) | [reference]({{% relref "/maintainers/annotations#swaggeroperation" %}}) |
| `swagger:parameters` | struct declaration | parameters on the named operation(s) | [example]({{% relref "/tutorials/routes-and-operations#swaggerparameters" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerparameters" %}}) |
| `swagger:response` | struct declaration | a `responses` entry | [example]({{% relref "/tutorials/routes-and-operations#swaggerresponse" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerresponse" %}}) |
| `swagger:ignore` | type / field doc | excludes the declaration | [example]({{% relref "/tutorials/model-definitions#swaggerignore" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerignore" %}}) |
| `swagger:name` | field / method doc | renames a JSON property | [example]({{% relref "/tutorials/model-definitions#swaggername" %}}) | [reference]({{% relref "/maintainers/annotations#swaggername" %}}) |
| `swagger:type` | type / field doc | overrides the inferred Swagger type | [example]({{% relref "/tutorials/model-definitions#swaggertype" %}}) | [reference]({{% relref "/maintainers/annotations#swaggertype" %}}) |
| `swagger:additionalProperties` | type doc | object `additionalProperties` (open / closed / typed) | [example]({{% relref "/tutorials/maps-and-free-form-objects#open--closed-objects" %}}) | [reference]({{% relref "/maintainers/annotations#swaggeradditionalproperties" %}}) |
| `swagger:patternProperties` | type doc | typed `patternProperties` (regex → value) | [example]({{% relref "/tutorials/maps-and-free-form-objects#pattern-properties" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerpatternproperties" %}}) |
| `swagger:file` | param / response field | `{type: file}` | [example]({{% relref "/tutorials/routes-and-operations#swaggerfile" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerfile" %}}) |
| `swagger:default` | value / field doc | a default-value anchor | [example]({{% relref "/tutorials/examples-and-defaults#swaggerdefault" %}}) | [reference]({{% relref "/maintainers/annotations#swaggerdefault" %}}) |

## Keywords, not annotations

Validations, examples and defaults inside a block are driven by **keywords**
(`minimum:`, `pattern:`, `enum:`, `example:`, `default:`, …), not annotations.
See the [Validations]({{% relref "/tutorials/validations" %}}) and
[Examples & defaults]({{% relref "/tutorials/examples-and-defaults" %}})
tutorials, and the [Keyword reference]({{% relref "/maintainers/keywords" %}}).
