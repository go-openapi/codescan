---
title: Usage as a library
weight: 1
description: |
  Import codescan, annotate a package.

  Produce a Swagger 2.0 specification from your Go program.
---

The most direct way to use codescan from your own Go program is to import it and call `Run`.

This supports various use-cases such as a generator, a `go:generate` step, or a test that keeps your spec in sync with the source.

## Install

```cmd
go get github.com/go-openapi/codescan
```

codescan exposes a deliberately small surface: a single `Run` function and an `Options` struct.

```go
func Run(opts *Options) (*spec.Swagger, error)
```

## Annotate your source

Annotations are special comments following the [go-swagger][go-swagger]
convention (`swagger:meta`, `swagger:route`, `swagger:model`, `swagger:parameters`, `swagger:response`, …).

A package-level `swagger:meta` block carries the top-level metadata of the spec:

{{< code file="petstore/doc.go" lang="go" lines="3-19" >}}

A `swagger:model` annotation turns a Go struct into a definition; field-level
comments become validations and descriptions:

{{< code file="petstore/pet.go" lang="go" region="model" >}}

## Run the scanner

Point codescan at the package(s) to scan. Patterns are relative `go list`-style patterns, resolved against `WorkDir`:

{{< code file="basic/scan.go" lang="go" region="runScan" >}}

The returned `*spec.Swagger` is the standard
[`github.com/go-openapi/spec`](https://pkg.go.dev/github.com/go-openapi/spec)
document — marshal it to JSON or YAML, feed it to a validator, or merge it onto an existing spec via `Options.InputSpec`.

## Options worth knowing

| Field | Effect |
|-------|--------|
| `Packages` | Relative `go list` patterns to scan (e.g. `./...`). |
| `WorkDir` | Directory the patterns resolve against. |
| `ScanModels` | Also emit definitions for `swagger:model` types. |
| `PruneUnusedModels` | With `ScanModels`, drop what nothing references — see [Pruning unused models]({{% relref "pruning-unused-models" %}}). |
| `InputSpec` | Overlay: merge discoveries on top of an existing spec. |
| `BuildTags`, `Include`/`Exclude` | Scope control over what gets scanned. |
| `OnDiagnostic` | Where everything the scan observed goes. codescan writes to no stream of its own, so without this the observations are lost. |

Those are the ones a first call tends to need. Everything else — alias handling,
`$ref` siblings, naming, doc-comment cleanup, the loader, the go environment —
is in the [Options reference]({{% relref "options-reference" %}}), which gives
each field with its command-line and configuration-file spellings beside it. The
[godoc][godoc] is the normative source.

## Dependencies

Code loading relies by default on the go toolchain and this requires go to be installed.

To alleviate this constraint, you may want to use the pure-go `ToolchainFreeLoader` loader in your options,
so programs that build on top of codescan don't shell out a `go list` command.

## Next

- [Scan a package]({{% relref "scan-a-package" %}}) — the whole of the above as
  one runnable, test-covered example.
- [Usage as a headless CLI]({{% relref "usage-as-a-headless-cli" %}}) — the same
  scan without writing a program, for a build or a pipeline.
- [Tutorials]({{% relref "/tutorials" %}}) — the worked, by-concept version of
  the above, each with the spec it produces.
- [Annotation index]({{% relref "/annotation-index" %}}) — every annotation at a
  glance, linked to its example and its full reference.
- [Maintainers reference]({{% relref "/maintainers" %}}) — the complete
  annotation vocabulary, keywords, and grammar.

[go-swagger]: https://github.com/go-swagger/go-swagger
[godoc]: https://pkg.go.dev/github.com/go-openapi/codescan#Options
