---
title: Setting options
weight: 10
description: |
  Three ways to set the same knob
---

There are three ways to set the same knob — a Go field, a command-line flag, a key
in a .codescan.yaml — where the file is looked for, what may be in it, and
which spelling wins when they disagree.

{{<tabs groupid="options-types">}}
{{% tab title="Go field" %}}
## As a Go field

Calling the library, an option is a field on
[`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options), passed to `Run`.
**The zero value is a valid configuration** — every boolean defaults to `false`,
every slice to `nil`, every numeric tunable to its built-in default — so you set only what you need:

{{< code file="basic/scan.go" lang="go" region="runScan" >}}

This is the only spelling that reaches *everything*: a few options are not
values a command line can carry — a filesystem to read, a document to merge
into, a callback to receive diagnostics. See [What has no flag](#what-has-no-flag).

{{% /tab %}}

{{% tab title="CLI flag" %}}
## As a flag

This is available for every CLI tool shipped: `genspec`, `genspec-tui`, `genspec-wasi`.
The library itself is not bound to command line flags.

Every value-typed option is a flag, on every command:

```cmd
genspec -name-from-tags form,json -prune-unused-models ./internal/api/...
```

**A flag is the kebab-case of the field it writes, without exception.**
`SkipJSONifyInterfaceMethods` is `-skip-jsonify-interface-methods`; no shorter
spelling is offered, because guessing one is how a caller ends up guessing wrong.
`genspec -h` lists them all, and the [Options reference]({{% relref "options-reference" %}})
gives the flag beside the field for each.

Two shapes do not follow from a field name:

- **the packages to scan** are positional arguments — `genspec ./api/...` —
  so that the command reads like the `go` commands it resolves patterns for.
  Naming none scans `./...`;
- **`-loader`** takes `go`, `own` or `auto` where the field
  (`ToolchainFreeLoader`) is a boolean: the useful default is the third answer,
  "whichever one can run here". On a native build that is `go list`, so a stock
  run loads exactly as it always did; `auto` picks `own` only where no subprocess
  can be started. That is what lets the same source build for a WASI guest, which
  has no process model and so can never run `go list`.

**Commands may add some extra flags that do not affect the scanning itself**.

- `genspec` adds its own `-output`, `-validate` or `-fail-on` (these are not library options)
- `genspec-tui` adds a unique `-profile` flag and related flags to store profiling data
- `genspec-wasi` adds a unique `-export-data` flag to preload a compiled go standard library

Those are documented with the command that owns them.
{{% /tab %}}

{{% tab title="Config file key" %}}
## As a configuration key

Anything that can be a flag can be preset in a `.codescan.yaml`, so a project
configures itself once and the command is run bare.

This is available for `genspec` and `genspec-tui` — not, for now, for `genspec-wasi`,
which reads its whole configuration from the arguments its host hands it.
The library itself is not bound to config files.

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

Keys are grouped into sections, and inside a section **a key is the flag it
sets**, spelled exactly as on the command line. There is no second vocabulary to
learn and no mapping table to keep in step: `genspec -h` is the reference for the
file.

### What a file may not set

**The options naming a path are settable on the command line only**: `-workdir`,
`genspec`'s `-output` and `-input`, and `genspec-tui`'s `-profile-dir`.

A file is found by searching upwards, so running a command inside a repository
reads *that repository's* file — and a tool whose job is reading somebody else's
code must not let the code decide where it reads or writes. Everything else a
file sets shapes the document, which is what one is for. Naming a file with
`-config` does not lift the restriction: the rule belongs to the option, so there
is nothing to remember at the point of use.

### The sections

The library's four are the questions a scan answers, in the order it answers
them. Each command adds its own for the flags that are its business:

| Section | The question | Declared by |
|---------|--------------|-------------|
| `scan` | which code is looked at | the library — every command |
| `go` | what it is built as: the go environment that decides what compiles | the library — every command |
| `load` | how the packages are read | the library — every command |
| `emit` | what the specification ends up saying | the library — every command |
| `document` | how the specification is rendered: `format`, `compact` | `genspec` |
| `diagnostics` | how loud it is about what it saw: `color`, `quiet`, `verbose`, `validate`, `fail-on` | `genspec` |
| `profile` | whether a run is profiled: `profile`, `mem-profile-rate` | `genspec-tui` |

### Where the file is found

The search walks **upwards** from where the command was run, so running it from
anywhere inside a project finds the project's file. It stops at the first hit
rather than merging what it passed: a file half-overridden by another three
directories up is not something anyone can read off the page.

`.codescan.yaml`, `.codescan.yml` and `.codescan.json` are looked for in that
order. JSON is a subset of YAML, so it needs no parser of its own — and a
generated file is as likely to be JSON.

| Flag | Effect |
|------|--------|
| `-config <path>`, `-c <path>` | read this file, which **must** exist — a caller who named one meant that file |
| `--no-config` | read none, whatever is lying around, for a run that has to be reproducible |

Asking for both at once is an error rather than a coin toss. So is `-config` and
`-c` naming different files.

{{% notice style="note" %}}
The search starts from the current directory, **not** from `-workdir`. A file
found through the very directory it was meant to describe would be reasoning in a
circle — which is one of the reasons `-workdir` is not something a file sets.
{{% /notice %}}

{{% /tab %}}
{{< /tabs >}}

## Which spelling wins

**Flags win over config**. That holds for a flag typed with the value it already had:

```cmd
# false, even where the file says scan-models: true
genspec -scan-models=false
```

It is decided by asking the flag set which flags were actually *seen*, not by
comparing values against defaults — which is what makes the rule statable in one
sentence instead of one per flag.

Everything the file sets lands through the same path a command-line argument
takes, so a value is parsed and validated exactly once, and a file cannot
express anything an argument could not. A malformed value is refused the same
way, naming the file.

Between the two, the ordinary defaults apply — with one deliberate divergence:
**the commands default `-scan-models` to `true`** where the library field is
`false`. A command asked to produce a specification and handed a package of
annotated models should produce their definitions.

## One file, several commands

There is one file name for the whole family, not one per command: a project
configuring a scan has configured it for all of them. What tells them apart is
the sections.

- A section a command does not know is **skipped**, not refused — that is how
  `genspec-tui`'s `profile:` settings sit beside `genspec`'s `document:` in the
  same file.
- A key inside a section it *does* know must name one of its flags. That is what
  makes a typo an error rather than a setting that quietly never applied.

Run `genspec -verbose` to see which file was read and which keys were skipped.

## What has no flag

Four options are not values a command line can carry.

The command owns them instead — and each of the commands makes its own choice about how,
which is why they are not in the shared table:

| Option | Reached by |
|--------|-----------|
| `InputSpec` | `genspec -input <file>` (`document.input`) — the command loads the document |
| `FS` | no flag: a filesystem is a Go value. The commands read the real one; the [Playground]({{% relref "/playground" %}}) hands the browser's |
| `ExportData` | `genspec-wasi -export-data <dir\|zip>` — the command opens the path. This option is specific to the wasi CLI for now |
| `OnDiagnostic`, `OnProvenance` | no flag: the commands wire the sinks. `genspec` prints diagnostics to standard error; `genspec-wasi -format=json` puts both in its envelope. Custom hooks are only available to the library. |

Two more fields carry no flag because nothing should be reaching for them:
`DescWithRef` and `Debug` are deprecated.

## Next

- [Options reference]({{% relref "options-reference" %}}) — every field, with its
  flag and its section.
- [Usage as a headless CLI]({{% relref "usage-as-a-headless-cli" %}}) — the flags
  `genspec` adds of its own.
