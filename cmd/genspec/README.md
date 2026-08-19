<!--
SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
SPDX-License-Identifier: Apache-2.0
-->

# genspec

Point it at annotated Go source, get a Swagger 2.0 document.

```sh
go install github.com/go-openapi/codescan/cmd/genspec@latest

# scan the module in the current directory, write the document to standard output
genspec

# somewhere else, narrowed, to a file, and checked
genspec -workdir ../my-api -output swagger.yaml -validate ./internal/api/...
```

This is the ordinary native command. Everything the library can be told is a
flag, the document goes to standard output or to `-output`, and what the scan
observed goes to standard error as colored diagnostics.

> **NOTE** `genspec` is a standalone CLI that works like `go-swagger` "swagger generate spec" command,
> but is released independently and will benefit fixes and enhancements at a faster pace than `go-swagger` releases.
> `go-swagger` has a larger scope and more dependencies.

## Which one to reach for

Three commands run the same scan. The question is about the machine, not about the specification:

| Command | Use it when |
|---------|-------------|
| **`genspec`** | you are on a normal machine and want a specification |
| [`genspec-wasi`](../genspec-wasi/README.md) | there is no Go toolchain, no subprocess, or you are running under WebAssembly — it takes no dependency beyond the library. It also speaks a machine-readable envelope (`-format=json`) carrying diagnostics and cross-references |
| [`genspec-tui`](../genspec-tui/README.md) | you are working *on* the annotations and want the source and the document side by side, live |

They share their flag surface: `cmd/internal/cliopts` declares every knob the
library takes, once, so `-name-from-tags` means the same thing whichever one you
reach for. A guard there fails the build when an option lands with no flag.

## Configuration file

Anything that can be a flag can be preset in a `.codescan.yaml`, found by
searching upwards from wherever you are — so a project configures itself once
and the command is run bare:

```yaml
scan:
  exclude-tags: [internal]

emit:
  scan-models: true
  name-from-tags: [form, json]

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

Keys are grouped into sections, and inside a section a key **is** the flag it
sets, spelled exactly as on the command line — so `genspec -h` doubles as the
reference for the file. The sections are the families below, plus `document`
and `diagnostics` for this command's own flags.

**Anything typed wins.** That holds for a flag typed with the value it already
had: `-scan-models=false` means false even where the file says true.

`-config <path>` — or `-c <path>` — reads a particular file, which must exist.
`--no-config` reads none, whatever is lying around, for a run that has to be
reproducible. Asking for both at once is an error rather than a coin toss.
`.codescan.yml` and `.codescan.json` are searched for too — JSON is a subset of
YAML, so it needs no parser of its own.

One file serves the whole family: a section a command does not recognize is skipped
rather than refused, so `genspec-tui`'s settings can sit beside `genspec`'s. A
key inside a section it *does* know must name one of its flags — which is what
makes a typo an error rather than a setting that quietly never applied. Run
with `-verbose` to see which file was read and which keys it skipped.

## Output

`-output` names the file, or `-` for standard output (the default). `-format`
is `json`, `yaml`, or `auto` — which reads the extension of `-output` and writes
JSON when that says nothing. So the common cases need no `-format` at all:

```sh
genspec -output swagger.yaml     # YAML, because of the name
genspec -output swagger.json     # JSON
genspec > swagger.json           # JSON
genspec -compact                 # JSON with no indentation
```

YAML is derived from the JSON rendering, which costs key order: the document
comes out alphabetical rather than in the order the spec types declare. Same
information, different diff against a hand-written file.

`-input` merges the scan's discoveries into an existing document — the place for
everything a scanner cannot know, such as the host, the security definitions, or
a hand-written path the annotations do not describe.

## Diagnostics

Everything the scan observed is reported on standard error, colored when that is
a terminal. Nothing is written there when there is nothing to say.

| Flag | Meaning |
|------|---------|
| `-quiet` | say nothing at all |
| `-verbose` | also report hints, which are muted by default |
| `-color` | `auto` (a terminal), `always`, `never`. `auto` honours `NO_COLOR` and `TERM=dumb` |
| `-validate` | check the document against the Swagger 2.0 schema and report what is wrong with it |
| `-fail-on` | exit non-zero when something reaches this severity: `error`, `warning`, or `never` |

`-fail-on` covers what `-validate` found as well as what the scan observed: they
reach the reader as one stream, so a threshold that saw only half of it would be
a trap rather than a policy. It defaults to `never`, because a scan that emits
warnings is the ordinary case, and a command that failed the build over one
would mostly teach people to stop reading them.

## Exit status

A specification is written whenever one could be produced, so a non-zero status
describes what was wrong with it rather than meaning nothing came out.

| Status | Meaning |
|--------|---------|
| 0 | the scan produced a document, and nothing asked for more |
| 1 | the scan failed |
| 2 | the command line does not make sense |
| 3 | what was reported reached the severity `-fail-on` names |
| 4 | `-validate` found the document invalid |

`-validate` finding the document invalid outranks `-fail-on`: it is the more
specific answer.

## Scan options

`genspec -h` lists them all. They fall into families:

- **what to scan** — `-workdir`, the positional package patterns, `-build-tags`,
  `-include` / `-exclude`, `-include-tags` / `-exclude-tags`, `-exclude-deps`
- **what to build it as** — `-goos`, `-goarch`, `-goflags`, `-gowork`,
  `-goexperiment`. Each changes what gets compiled, and so what the document
  says; each is a flag rather than inherited state, so a scan is reproducible
- **how to load it** — `-loader`, `-stub-stdlib`, `-compiled-dependencies`
- **what to emit** — `-scan-models`, `-prune-unused-models`, the alias and
  `allOf` knobs, `-skip-extensions`, the naming knobs, the doc-comment knobs

They are the library's own options under their own names: a flag is the kebab-case of the field it writes,
without exception.
See the [package documentation](https://pkg.go.dev/github.com/go-openapi/codescan#Options) for what each one means.
