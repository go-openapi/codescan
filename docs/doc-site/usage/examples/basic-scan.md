---
title: Scan a package
weight: 1
description: |
  The smallest end-to-end use of codescan: annotate a package, scan it, and
  get back a Swagger 2.0 document.
---

This example scans a tiny annotated "petstore" package and produces a Swagger
2.0 spec. It is the worked version of
[usage as a library](../../getting-started/usage-as-a-library/).

## The annotated API

A package-level `swagger:meta` block sets the top-level metadata:

{{< code file="petstore/doc.go" lang="go" lines="3-19" >}}

A `swagger:route` registers a path and ties it to a response:

{{< code file="petstore/pet.go" lang="go" region="route" >}}

A `swagger:model` struct becomes a definition, with field comments driving
validations:

{{< code file="petstore/pet.go" lang="go" region="model" >}}

## Running the scan

`ScanPetstore` builds the `Options` and calls `codescan.Run`:

{{< code file="basic/scan.go" lang="go" region="runScan" >}}

## The generated spec

Marshalling the returned `*spec.Swagger` to JSON yields the document below —
the meta block became the top-level `info` / `basePath`, the `swagger:route`
became the `/pets` path, and the `swagger:model` became the `Pet` definition:

{{< code file="basic/testdata/swagger.json" lang="json" >}}

This JSON is not hand-written: it is a golden file the example's test
regenerates and compares on every run (`UPDATE_GOLDEN=1 go test ./...`). Because
the example is ordinary, test-covered Go, `go test ./docs/examples/...` keeps
the page honest — if the scanner's output changes, CI fails before the
documentation can go stale.
