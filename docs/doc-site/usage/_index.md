---
title: Advanced Usage
weight: 20
description: |
  All the knobs codescan takes. What they are, and how they relate.

  There are three spellings: a Go field, a command-line flag, and a key in a configuration file.
---

codescan is a go library that ships with a few utility commands.
These commands are thin wrappers around the library's scanner.

The library's behavior is tuned by fields on [`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options).
The commands — [`genspec`]({{% relref "usage-as-a-headless-cli" %}}),
[`genspec-tui`]({{% relref "usage-as-a-tui" %}}) and `genspec-wasi` — register one flag per field over that same struct.

So every available knob has **three spellings for one meaning**:

| Spelling | Looks like |
|----------|------------|
| A Go field | `opts.NameFromTags = []string{"form", "json"}` |
| A command-line flag | `genspec -name-from-tags form,json` |
| A configuration key | `emit:`<br>&nbsp;&nbsp;`name-from-tags: [form, json]` |

The mapping is mechanical: a flag is the kebab-case of the field it writes, without exception,
and a configuration key **is** the flag, spelled exactly as on the command line, under the section that flag belongs to.

That is why `genspec -h` doubles as the reference for the file — and why the one
reference below serves all three.

{{% notice style="note" %}}
The commands do not each declare that surface: it is written once, and a guard
fails the build when an option lands with no flag — so a knob added to the
library is reachable from all of them at the same moment. How that is arranged is
in [The commands]({{% relref "commands" %}}).
{{% /notice %}}

{{< children type="card" description="true" >}}
