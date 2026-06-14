---
title: Usage as a library
weight: 1
description: |
  Import codescan, annotate a package, and produce a Swagger 2.0 specification
  from your Go program.
---

The most direct way to use codescan is to import it and call `Run` from your
own Go program — a generator, a `go:generate` step, or a test that keeps your
spec in sync with the source.

## Annotate your source

Annotations are special comments following the [go-swagger][go-swagger]
convention (`swagger:meta`, `swagger:route`, `swagger:model`,
`swagger:parameters`, `swagger:response`, …).

A package-level `swagger:meta` block carries the top-level metadata of the
spec:

{{< code file="petstore/doc.go" lang="go" lines="3-19" >}}

A `swagger:model` annotation turns a Go struct into a definition; field-level
comments become validations and descriptions:

{{< code file="petstore/pet.go" lang="go" region="model" >}}

## Run the scanner

Point codescan at the package(s) to scan. Patterns are relative `go list`-style
patterns, resolved against `WorkDir`:

{{< code file="basic/scan.go" lang="go" region="runScan" >}}

The returned `*spec.Swagger` is the standard
[`github.com/go-openapi/spec`](https://pkg.go.dev/github.com/go-openapi/spec)
document — marshal it to JSON or YAML, feed it to a validator, or merge it onto
an existing spec via `Options.InputSpec`.

## Options worth knowing

| Field | Effect |
|-------|--------|
| `Packages` | Relative `go list` patterns to scan (e.g. `./...`). |
| `WorkDir` | Directory the patterns resolve against. |
| `ScanModels` | Also emit definitions for `swagger:model` types. |
| `InputSpec` | Overlay: merge discoveries on top of an existing spec. |
| `BuildTags`, `Include`/`Exclude` | Scope control over what gets scanned. |
| `RefAliases`, `TransparentAliases`, `DescWithRef` | Alias-handling knobs. |
| `SkipExtensions` | Suppress `x-go-*` vendor extensions. |
| `SkipEnumDescriptions` | Keep the `swagger:enum` const→value mapping out of property/parameter descriptions (it still rides `x-go-enum-desc`). |

See the [godoc][godoc] for the full list.

## Next

- [Tutorials]({{% relref "/tutorials" %}}) — the worked, by-concept version of
  the above, each with the spec it produces.
- [Annotation index]({{% relref "/annotation-index" %}}) — every annotation at a
  glance, linked to its example and its full reference.
- [Maintainers reference]({{% relref "/maintainers" %}}) — the complete
  annotation vocabulary, keywords, and grammar.

[go-swagger]: https://github.com/go-swagger/go-swagger
[godoc]: https://pkg.go.dev/github.com/go-openapi/codescan#Options
