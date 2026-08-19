<!--
SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
SPDX-License-Identifier: Apache-2.0
-->

# genspec-wasi

A headless spec generator: point it at annotated Go source, get a Swagger 2.0
document on standard output. It is the non-interactive counterpart to
[`genspec-tui`](../genspec-tui/README.md), and the form codescan takes when it
is built for WebAssembly.

It depends on nothing beyond the library itself, which is the point: it
cross-compiles to `wasip1/wasm` and runs under any WASI runtime, with no Go
toolchain present and no subprocess.

Audience: codescan/go-swagger maintainers and contributors.

## Install and run

`genspec-wasi` lives in the **main module**, so it carries no dependencies the
library does not already have.

```sh
go install github.com/go-openapi/codescan/cmd/genspec-wasi@latest

# scan the module in the current directory
genspec-wasi

# or point it somewhere, and narrow the scope
genspec-wasi -workdir ../my-api ./internal/models/... ./internal/api/...
```

From a checkout:

```sh
go run ./cmd/genspec-wasi -workdir ./testdata ./goparsing/petstore/...
```

| Flag | Default | Meaning |
|---|---|---|
| `-workdir` | `.` | directory the scan runs from; patterns are relative to it |
| `-scan-models` | `true` | also emit definitions for `swagger:model` types |
| *(one flag per boolean option)* | | every boolean knob on `codescan.Options` has a flag, named after the field in kebab-case — `-prune-unused-models`, `-ref-aliases`, `-clean-go-doc` … Run `genspec-wasi -h` for the list. The one exception is `ToolchainFreeLoader`, which `-loader` states more fully. |
| `-build-tags` | | comma-separated build tags to apply while loading |
| `-goos` / `-goarch` | this machine's | the platform the **scanned code** is built for |
| `-loader` | `auto` | here `auto` and `own` are the same thing: a WebAssembly guest can start no subprocess, so `go list` can never run. `go` is refused up front for that reason, rather than failing deep inside the go command |
| `-export-data` | | directory or `.zip` of precomputed dependency types (see below) |
| `-stub-stdlib` | `false` | synthesize standard-library types instead of reading GOROOT |
| `-compiled-dependencies` | `false` | take dependency types from the compiler's export data rather than reading every dependency from source. The spec is the same either way; this is much faster on a warm build cache and much slower on a cold one, since export data has to be compiled before it exists. **Native builds only** — it needs `-loader=go`, which WebAssembly cannot run |
| `-format` | `spec` | `spec` writes the document alone; `json` wraps it with diagnostics and provenance (see below) |
| `-output` | `-` | where to write the specification |
| `-indent` | `true` | indent the emitted JSON |
| `-quiet` | `false` | suppress scan diagnostics on standard error (moot under `-format=json`) |

`-loader=auto` is why the same source builds for both worlds: WebAssembly has
no process model, so `go list` can never run there and the choice makes itself.

## Writing for a program: `-format=json`

The default shape — document on stdout, diagnostics as prose on stderr — is what
a person or a shell pipeline wants. It is not enough for a program. Prose carries
a position only in the sense that one is printed in it, and provenance, which
says *which Go construct produced this spec node*, has nowhere to go at all.

`-format=json` writes one object instead:

```json
{
  "spec": { "swagger": "2.0", "definitions": { "doc": {} } },
  "diagnostics": [
    {"severity": "hint", "code": "validate.dropped-ref-sibling",
     "message": "field \"for\": description dropped …",
     "file": "models/m.go", "line": 12, "col": 5}
  ],
  "provenance": [
    {"pointer": "/definitions/doc", "file": "models/m.go", "line": 8, "col": 6},
    {"pointer": "/definitions/doc/properties/at", "file": "models/m.go", "line": 10, "col": 2}
  ],
  "runtime": {
    "sys": 390070272, "heapAlloc": 369098752,
    "totalAlloc": 372244480, "collections": 2
  }
}
```

`runtime` is the Go runtime's own account of the scan, read just after it
finished and without forcing a collection first — collecting would report a
tidier heap than the one the scan actually ran with. It is carried because it
cannot be recovered afterwards: once the process has exited, nobody can ask.
Everything else here can be. Definition counts come from the document and
elapsed time is the caller's own clock, which is why neither is included.

`sys` is memory obtained from the host; under `wasip1` the linear memory never
shrinks, so it is a high-water mark rather than a reading at one moment.
`heapAlloc` against `totalAlloc` says which kind of heavy a scan is: close
together means it is **holding** memory, far apart with many collections means
it is churning through it.

**Paths are relative to `-workdir`.** The scan reports where it read a file from,
which under a WASI guest is whichever mount point the host chose. A caller holds
`models/pet.go` and should not have to know that the guest called it
`/src/models/pet.go`. A position *outside* the module — a dependency in the module
cache, a type in GOROOT — stays absolute rather than becoming a chain of `..`,
and that is how a consumer tells the two apart.

**Provenance covers anchors only**: type declarations, fields, values, and route
or meta blocks. A finer node resolves to its nearest anchored ancestor at the
consumer, which is the contract `genspec-tui`'s cross-references already work to.
Anchors are sorted by pointer, so the same scan produces the same bytes.

Diagnostics keep the order they were reported in, and do **not** also go to
stderr: under `-format=json` stderr stays clean, which is what makes the envelope
safe to read from a pipe.

Errors are sentinels, so a caller can ask which refusal it met rather than
matching on prose: `errBadFlag` for a flag value the command does not accept,
`errBadExportData` for a `-export-data` path that is neither a directory nor a
`.zip`.

Experimental, inheriting the status of `Options.OnProvenance`.

## Running it under a WASI runtime

Build the artifact, then hand it to a runtime along with the directories it is
allowed to read:

```sh
GOOS=wasip1 GOARCH=wasm go build -o genspec-wasi.wasm ./cmd/genspec-wasi
```

Verified against **wasmtime 41** and **wazero 1.11**. Their mount syntax
differs, which is the first thing to get wrong:

```sh
# wasmtime — <host>::<guest>
wasmtime run --dir "$PWD::$PWD" genspec-wasi.wasm -workdir "$PWD" ./...

# wazero — <host>:<guest>, no separator
wazero run -mount="$PWD:$PWD" genspec-wasi.wasm -workdir "$PWD" ./...
```

Two things a guest cannot work out for itself:

- **`-goos` / `-goarch` must be passed explicitly.** Left alone they default to
  the platform the scanner is *running* on, which inside a guest is `wasip1`.
  That silently drops every `_linux.go` file and produces a different
  specification than the same scan run natively. Pass the platform of the code
  under scan.
- **GOROOT and the module cache are found by path.** Nothing in a WASI
  environment can ask the go command where they live, so if the scan needs
  them they have to be mounted *and* named through the environment
  (`--env GOROOT=…`, `--env GOMODCACHE=…`).

wazero is a pure-Go runtime and convenient to embed in tests; wasmtime is
several times faster on this workload. Both produce identical output.

## How much of the host to expose

The real choice is how much of the host the guest may see. Measured on the
petstore fixture under wasmtime:

| mounted | mode | time | peak RSS | result |
|---|---|---|---|---|
| GOROOT + module cache | default | 7.3 s | 681 MB | identical to a `go list` scan |
| module cache | `-export-data` | 1.0 s | 138 MB | identical |
| module cache | `-stub-stdlib` | 1.0 s | 147 MB | degraded — see below |
| project tree only | `-stub-stdlib` | 0.1 s | 123 MB | degraded |

Memory is usually the binding constraint rather than time: 681 MB for a fixture
this small is more than a browser tab can host.

### Precomputed dependency types

`-export-data` reads a scan's **dependencies** from the export data the
compiler already produced, instead of parsing and type-checking them. That is
where nearly all the time goes, so it costs a fraction — with no loss of
fidelity, because the types are the compiler's own. The module being scanned
is always read from source: its comments are the annotations.

It takes a directory or a `.zip`, so a host with somewhere to put a file but
no tree to build hands over one blob.

```sh
go run ./hack/genexportdata -out /tmp/exportdata std

wasmtime run --dir "$PWD::$PWD" --dir /tmp/exportdata::/tmp/exportdata \
  genspec-wasi.wasm -export-data /tmp/exportdata -workdir "$PWD" ./...
```

The data is valid only for the toolchain that generated it, since the export
format is tied to the Go release. Regenerate it when the toolchain moves.

**A package whose meaning lives in comments cannot go in.** `strfmt` declares
its formats with `swagger:strfmt`, and export data holds types, not comments —
such a package comes back structurally intact and semantically empty, with
nothing erroring. `genexportdata` detects and skips them, saying which; they
have to be read from source.

### A build that needs nothing mounted but the project

The `exportdata` tag embeds that same data in the binary, so the artifact is
self-contained:

```sh
go run ./hack/genexportdata -out internal/exportdata/exportdata.zip std
GOOS=wasip1 GOARCH=wasm go build -tags exportdata -o genspec-wasi.wasm ./cmd/genspec-wasi

# no GOROOT, no toolchain, nothing but the sources being scanned
wasmtime run --dir "$PWD::$PWD" genspec-wasi.wasm -workdir "$PWD" ./...
```

That costs about 5 MB of artifact (20 MB, 8.5 MB compressed, against 15 MB) and
runs the petstore in 1.1 s. The archive is generated rather than committed.

### Synthesizing the standard library instead

`-stub-stdlib` fabricates standard-library types from the names the scanned code
selects through them. It needs no GOROOT and no module cache at all, and it is
the smallest footprint on offer — but it is **not failsafe**, and its failure
mode is quiet: the specification comes out slightly thinner rather than
erroring.

Recognition by type identity survives, so `time.Time` is still a `date-time`.
Structure does not: a synthesized type has no fields and no method set, so
`json.RawMessage` stops rendering as a byte array, `time.Duration` as an
integer, and a type is no longer seen to implement `encoding.TextMarshaler`.
Across codescan's fixture corpus 138 of 143 scans stay byte-identical.

Prefer a full graph, or the export data above, wherever GOROOT is available.

Whatever the mode, every import that had to be synthesized raises a
`scan.synthesized-import` diagnostic on standard error naming the import and
where it came from, so the loss is never silent. Drop `-quiet` to see them.

## Tests

The integration tests build the artifact and run it under whichever runtime is
on `PATH`, comparing the result against an in-process scan:

```sh
go test ./internal/integration/ -run TestWASIArtifact -v
```

They skip when no runtime is installed, when there is no go command to build
with, and under `-short`. The self-contained case additionally skips unless
`internal/exportdata/exportdata.zip` has been generated.
