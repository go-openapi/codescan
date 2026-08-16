---
title: "The commands"
weight: 50
description: |
  How the three CLI tools are put together.

  Why there are three of them, where their shared flag surface is declared, and what keeps it whole.
---

Three commands ship from this repository — [`genspec`]({{% relref "usage-as-a-headless-cli" %}}),
[`genspec-tui`]({{% relref "usage-as-a-tui" %}}) and `genspec-wasi` — and they run
the same scan. This page is about how they are arranged, which matters to anyone
adding an option, adding a command, or wondering why the library stays as light
as it does.

For what the flags and configuration keys *mean*, see
[Setting options]({{% relref "setting-options" %}}) and the
[Options reference]({{% relref "options-reference" %}}).

## Why three, and where each one lives

The split is about dependencies, not about features.

| Command | Module | Because |
|---------|--------|---------|
| `genspec-wasi` | the **main** module | it must add nothing to what the library already needs — that is what lets it cross-compile to `wasip1/wasm` and run with no toolchain and no subprocess |
| `genspec` | its **own** module | it takes koanf, for the configuration sources it will be asked for next |
| `genspec-tui` | its **own** module | it takes bubbletea and the tree that comes with it |

So installing the terminal UI pulls none of bubbletea into *your* project, and a
program importing the library gets neither. A command that needed a dependency
the library does not want is a reason to give it a module, not a reason to argue
about the dependency.

{{% notice style="note" %}}
The submodules are tagged and have no `replace` directive, so
`go install github.com/go-openapi/codescan/cmd/genspec-tui@latest` really works
from a clean environment.
{{% /notice %}}

## One flag surface, declared once

Every knob the library takes is declared as a flag in **`cmd/internal/cliopts`**,
once, and each command registers the whole of it. That is what makes
`-name-from-tags` mean the same thing whichever one you reach for, and what makes
a knob added to the library reachable from all three at the same moment.

Written per command instead, the mapping would be partial in a different way each
time, and an option added to `Options` would reach whichever command somebody
remembered.

Three properties hold it together:

- **Entries are keyed by the field's setter**, never by a name derived from the
  field. Naming the field in a string would let a rename pass the compiler and
  leave the flag writing nowhere.
- **A guard in the package tests fails** when a value-typed option lands with no
  flag — or no recorded excuse. A caller cannot use an option that has no flag,
  and would find out by meeting `flag provided but not defined` after writing
  something against a surface that was never there. The pull request that adds
  the option is a better place to find out.
- **A flag is the kebab-case of the field**, without exception. No rule can derive
  that `JSONify` is one word where `HTTPServer` is two, so coverage is decided by
  writing through a setter and seeing what moved — never by mangling a name.

The package sits under `cmd/` rather than under the root `internal/`, which is
what lets the commands that are modules of their own import it.

Options that are not values — a filesystem, a document to merge into, the
callbacks — are deliberately absent from the table. Those are the command's
business, and the tests carry the list with a reason for each. See
[What has no flag]({{% relref "setting-options#what-has-no-flag" %}}).

## One configuration contract, two readers

**`cmd/internal/cliconf`** owns where a `.codescan.yaml` is found, what may be in
it, and how it loses to anything typed — and nothing else. The values themselves
arrive as a plain flat map, so how they are *read* stays the command's business:

- `genspec` feeds the file through **koanf**, for the environment variables and
  further formats it will be asked for next;
- `genspec-tui` — which reads one file, once, to decide what a session starts
  with — calls **`cliconf.Parse`**, which needs nothing this repository does not
  already have.

That seam is why the package can be shared with a command that must
cross-compile to WebAssembly: `cliconf.YAML` satisfies koanf's parser interface
*structurally*, so the package owes koanf no import.

Sections come from `cliopts.ConfigSchema()` merged with each command's own, which
is why a section one command does not know is skipped rather than refused, and a
key inside a section it *does* know must name one of its flags.

Everything a file sets lands through `flag.FlagSet.Set` — the same path the
command line takes — so a value is parsed and validated exactly once, and a file
cannot express anything an argument could not.

## See also

- [Setting options]({{% relref "setting-options" %}}) — the same contract from
  the reader's side.
- [`genspec` README](https://github.com/go-openapi/codescan/blob/master/cmd/genspec/README.md),
  [`genspec-tui` README](https://github.com/go-openapi/codescan/blob/master/cmd/genspec-tui/README.md),
  [`genspec-wasi` README](https://github.com/go-openapi/codescan/blob/master/cmd/genspec-wasi/README.md)
  — each command's own deep documentation, including the TUI's full keymap and the
  WASI mount and export-data recipes.
