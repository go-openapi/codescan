---
title: Usage as a headless CLI
weight: 3
description: |
  Point genspec at annotated Go source and get a Swagger 2.0 document — on
  standard output, in a file, checked against the schema, or as a
  machine-readable envelope from a sandbox with no Go toolchain in it.
---

`genspec` is the ordinary way to run codescan without writing a program: point it at annotated Go source,
get a Swagger 2.0 document. Everything the library can be told is a flag, the document goes to standard output or to `-output`,
and what the scan observed goes to standard error.

## Install

```cmd
go install github.com/go-openapi/codescan/cmd/genspec@latest
```

```cmd
# scan the module in the current directory, document to standard output
genspec

# somewhere else, narrowed, to a file, and checked
genspec -workdir ../my-api -output swagger.yaml -validate ./internal/api/...
```

The packages are Go patterns, resolved against `-workdir`. Naming none scans
`./...`.

{{% notice style="note" %}}
`genspec` does the job of go-swagger's `swagger generate spec`, but is released
independently of it — so fixes and new options reach it at codescan's pace
rather than go-swagger's. go-swagger has a much larger scope, and the
dependencies that go with it.
{{% /notice %}}

## Two streams, never mixed

The document goes to standard output; diagnostics go to standard error, colored
when that is a terminal. Nothing is written to standard error when there is
nothing to say. So the obvious pipeline is safe:

```cmd
genspec > swagger.json
genspec | jq '.definitions | keys'
```

This is the command's doing, not the library's: codescan itself never writes to
either stream. Every observation it makes — a dropped construct, a rename, a
prune — is passed to the command through a callback, and `genspec` renders those
callbacks as lines you can read.

## Choosing the output

| Flag | Effect |
|------|--------|
| `-output` | the file to write, or `-` for standard output (the default) |
| `-format` | `json`, `yaml`, or `auto` — which reads the extension of `-output`, and writes JSON when that says nothing |
| `-compact` | JSON with no indentation |
| `-input` | merge the scan's discoveries into an existing document |

`auto` is why the common cases need no `-format` at all:

```cmd
genspec -output swagger.yaml     # YAML, because of the name
genspec -output swagger.json     # JSON
genspec > swagger.json           # JSON
genspec -compact                 # JSON with no indentation
```

YAML is derived from the JSON rendering, which costs key order: the document
comes out alphabetical rather than in the order the spec types declare. Same
information, a different diff against a hand-written file.

`-input` is the place for everything a scanner cannot know — the host, the
security definitions, a hand-written path the annotations do not describe. The
scan is merged on top of it; see
[Overlaying a spec]({{% relref "overlaying-a-spec" %}}).

## Diagnostics

| Flag | Effect |
|------|--------|
| `-quiet` | say nothing at all |
| `-verbose` | also report hints, which are muted by default — and say which configuration file was read |
| `-color` | `auto` (a terminal), `always`, `never`. `auto` honours `NO_COLOR` and `TERM=dumb` |
| `-validate` | check the document against the Swagger 2.0 schema and report what is wrong with it |
| `-fail-on` | exit non-zero when something reaches this severity: `error`, `warning`, or `never` |

`-fail-on` covers what `-validate` found as well as what the scan observed: they reach the reader as one stream,
so a threshold that saw only half of it would be a trap rather than a policy.
It defaults to `never`, because a scan that emits warnings is the ordinary case,
and a command that failed the build over one would mostly teach people to stop reading them.

## Validating: whether the document is legal Swagger 2.0

A scan that reports nothing is not a promise that the document it produced is
valid. The scanner diagnoses what is wrong with your *annotations*; whether the
result is a legal Swagger 2.0 document is a separate question. Pass `-validate`
to answer it:

```cmd
genspec -validate -output swagger.yaml ./internal/api/...
```

It checks the rendered JSON rather than the document in memory, so what is
validated is exactly what was written — including whatever the round trip
through JSON did to it. The check is [go-openapi/validate][validate], the same
one behind `swagger validate`.

Findings arrive through the same stream as the scan's own, because to a reader
they are the same kind of news about the same document. Each is located by the
JSON pointer the validator recorded as it walked the spec, so it names the node
rather than a line:

```text
ERROR | paths./orders.post.parameters.1.in in body is required at=/paths/~1orders/post/parameters/1
```

A finding about something the document *lacks* — no `info` block, no `paths` —
has an empty pointer, which is a location in RFC 6901 and not the absence of
one. It is reported as the whole document rather than printed as nothing:

```text
ERROR | info in body is required at=(the whole document)
genspec: the specification is not valid: 1 finding(s)
```

Warnings and errors both come out. Only errors make the document invalid: a
warning — an unused definition, say — is worth saying and is not a verdict, and
whether it fails the command is `-fail-on`'s business rather than the
validator's.

An invalid document exits **4**, and that outranks `-fail-on`'s **3**: it is the
more specific answer. The document is still written either way, so you can look
at what was rejected.

[validate]: https://github.com/go-openapi/validate

## Exit status

A specification is written whenever one could be produced, so a non-zero status
says what was wrong with the document rather than meaning nothing came out.

| Status | Meaning |
|--------|---------|
| 0 | the scan produced a document, and nothing asked for more |
| 1 | the scan failed |
| 2 | the command line does not make sense |
| 3 | what was reported reached the severity `-fail-on` names |
| 4 | `-validate` found the document invalid |

An invalid document outranks `-fail-on`: it is the more specific answer.

## Configuring it once

Anything that can be a flag can be preset in a `.codescan.yaml`, found by
searching upwards from wherever you are — so a project configures itself once
and the command is run bare:

```yaml
scan:
  exclude-tags: [internal]

document:
  format: yaml
  compact: true

diagnostics:
  validate: true
  fail-on: warning
```

The options naming a **path** are not among them — `-workdir`, `-output`, `-input` — because a file
found by searching upwards belongs to the tree being scanned, and that tree must not choose where the
command reads or writes. They are typed:

```cmd
genspec -workdir ./api -output swagger.yaml ./...
```

Anything typed on the command line wins over it. The file's full contract —
where it is looked for, what sections exist, `-config` and `--no-config` — is in
[Setting an option]({{% relref "setting-options" %}}).

## The scan itself

`genspec -h` lists every flag. They fall into the four families a configuration
file uses as its sections:

- **which code** (`scan`) — `-workdir`, the positional patterns, `-build-tags`,
  `-include` / `-exclude`, `-include-tags` / `-exclude-tags`, `-exclude-deps`
- **what it is built as** (`go`) — `-goos`, `-goarch`, `-goflags`, `-gowork`,
  `-goexperiment`. Each decides what compiles, and so what the document says;
  each is a flag rather than inherited state, so a scan is reproducible
- **how it is read** (`load`) — `-loader`, `-stub-stdlib`,
  `-compiled-dependencies`
- **what is emitted** (`emit`) — `-scan-models`, `-prune-unused-models`, the
  alias and `allOf` knobs, `-skip-extensions`, the naming and doc-comment knobs

They are the library's own options under their own names: a flag is the
kebab-case of the field it writes, without exception. The
[Options reference]({{% relref "options-reference" %}}) gives each one with its
field, its default and what it does.

## Without a Go toolchain

`genspec-wasi` is the same scan with nothing behind it: it depends on nothing
beyond the library, runs no subprocess, and cross-compiles to `wasip1/wasm`.
Use it where `genspec` cannot go — a sandbox, a CI image with no Go in it, a
WebAssembly runtime — and it is the engine behind the
[Playground]({{% relref "/playground" %}}).

```cmd
go install github.com/go-openapi/codescan/cmd/genspec-wasi@latest
genspec-wasi -workdir ../my-api ./internal/api/...
```

It carries the same scan flags, but drops `genspec`'s document and diagnostics
surface: no `-validate`, no `-fail-on`, no `-color`. In their place it offers an
envelope.

### An envelope for a program

Document on stdout and prose on stderr suits a person or a pipeline. It is not
enough for a program: prose carries a position only in the sense that one is
printed in it, and provenance — which Go construct produced this spec node — has
nowhere to go at all. `-format=json` writes one object instead:

```json
{
  "spec": { "swagger": "2.0", "definitions": { "doc": {} } },
  "diagnostics": [
    {"severity": "hint", "code": "validate.dropped-ref-sibling",
     "message": "field \"for\": description dropped …",
     "file": "models/m.go", "line": 12, "col": 5}
  ],
  "provenance": [
    {"pointer": "/definitions/doc", "file": "models/m.go", "line": 8, "col": 6}
  ],
  "runtime": { "sys": 390070272, "heapAlloc": 369098752, "collections": 2 }
}
```

Positions are relative to `-workdir`, so a caller holds `models/pet.go` and need
not know what the guest called it; a position *outside* the module stays
absolute, which is how a consumer tells the two apart. Provenance covers anchors
— type declarations, fields, values, route and meta blocks — sorted by pointer,
so the same scan produces the same bytes. Under `-format=json` standard error
stays clean, so the envelope is safe to read from a pipe.

Experimental, inheriting the status of `Options.OnProvenance`.

### Under a WASI runtime

```cmd
GOOS=wasip1 GOARCH=wasm go build -o genspec-wasi.wasm ./cmd/genspec-wasi

# wasmtime — <host>::<guest>
wasmtime run --dir "$PWD::$PWD" genspec-wasi.wasm -workdir "$PWD" ./...

# wazero — <host>:<guest>, no separator
wazero run -mount="$PWD:$PWD" genspec-wasi.wasm -workdir "$PWD" ./...
```

Two things a guest cannot work out for itself:

- **`-goos` / `-goarch` must be passed.** Left alone they default to the
  platform the scanner is *running* on, which inside a guest is `wasip1` — which
  silently drops every `_linux.go` file and produces a different document than
  the same scan run natively.
- **GOROOT and the module cache are found by path.** Nothing in a WASI
  environment can ask the go command where they live, so if the scan needs them
  they must be mounted *and* named through the environment.

### How much of the host to expose

The real choice is how much of the host the guest may see. Measured on the
petstore fixture under wasmtime:

| mounted | mode | time | peak RSS | result |
|---------|------|------|----------|--------|
| GOROOT + module cache | default | 7.3 s | 681 MB | identical to a `go list` scan |
| module cache | `-export-data` | 1.0 s | 138 MB | identical |
| module cache | `-stub-stdlib` | 1.0 s | 147 MB | degraded |
| project tree only | `-stub-stdlib` | 0.1 s | 123 MB | degraded |

Memory is usually the binding constraint rather than time — 681 MB for a fixture
this small is more than a browser tab can host.

`-export-data` reads dependencies from the export data the compiler already
produced instead of parsing and type-checking them. It costs a fraction of the
time with **no** loss of fidelity, the types being the compiler's own; it is
valid only for the toolchain that produced it. `-stub-stdlib` fabricates
standard-library types from the names the code selects, needs no GOROOT at all,
and is the one option here that is **not** failsafe: recognition by identity
survives (`time.Time` is still a `date-time`) but structure does not, so
`json.RawMessage` stops rendering as a byte array and nothing is seen to
implement `encoding.TextMarshaler`. Every synthesized import raises a
`scan.synthesized-import` diagnostic, so the loss is never silent.

Full detail, including a build that embeds its own export data and needs nothing
mounted but the project, is in the
[`genspec-wasi` README](https://github.com/go-openapi/codescan/blob/master/cmd/genspec-wasi/README.md).

## Next

- [Setting an option]({{% relref "setting-options" %}}) — the configuration file
  in full, and which spelling wins.
- [Options reference]({{% relref "options-reference" %}}) — every flag with the
  library field it writes.
- [Usage as a terminal UI]({{% relref "usage-as-a-tui" %}}) — the same scan with
  the source and the document side by side, while you write the annotations.
