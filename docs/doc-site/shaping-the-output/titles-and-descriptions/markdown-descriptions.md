---
title: Markdown descriptions
weight: 15
description: |
  Carry a verbatim markdown body — tables, blank lines, indentation and all —
  into a description with the swagger:description | literal block-scalar marker,
  instead of letting Option B fold it.
---

A multi-line `swagger:description` normally folds its body with the **Option B**
rule: contiguous prose lines up to the first blank line, each trimmed. That's
right for a paragraph of prose, but it destroys markdown — a blank line ends the
description, and leading indentation and table pipes are stripped. So a table or
a multi-paragraph body never survives the trip into the spec.

Ending the annotation line with a lone `|` — the YAML **literal block-scalar
marker** — opts the body into *verbatim* capture instead. Everything below is
taken exactly as written — blank lines, indentation, table pipes and `---` all
preserved — until the next annotation or the end of the comment. It is opt-in
per annotation; a plain `swagger:description` (no `|`) keeps the Option B
behaviour unchanged.

## Plain prose vs a verbatim body

These two models carry the *same* markdown body. The first uses an ordinary
annotation; the second adds the `|` marker:

{{< code file="shaping/markdowndesc/markdowndesc.go" lang="go" region="plain" >}}

{{< code file="shaping/markdowndesc/markdowndesc.go" lang="go" region="markdown" >}}

The emitted descriptions diverge sharply:

{{< compare left="shaping/markdowndesc/testdata/plain.json" leftlabel="Plain — Option B folds"
            right="shaping/markdowndesc/testdata/markdown.json" rightlabel="swagger:description | — verbatim" >}}

- **Option B stops at the first blank line.** The plain model's description is
  just the opening sentence — the table that follows the blank line is dropped
  entirely (the original go-swagger#3211 grievance).
- **The `|` body is captured whole.** Table leading pipes, the significant blank
  line, and the bullet list after it all ride through verbatim.
- **The marker never leaks.** The trailing `|`, the `swagger:description` line
  itself, and the single godoc `// ` convention space per line are all stripped;
  interior indentation and trailing whitespace (markdown hard breaks) are kept.
- **The title is unaffected.** It still comes from the godoc preamble above the
  annotation — only the description body becomes verbatim.

It works on a **field** description just the same — the `name` property above
keeps the indentation of its ordered list (`  1. unique`).

## Where the block ends

The literal block runs until the **next annotation at the start of a line**, or
the end of the doc comment. In the examples above, the trailing
`swagger:model Widget` line closes the block.

A `swagger:` token *mid-line* is ordinary prose and stays in the body — only a
line that *begins* with an annotation terminates. Indentation doesn't shield
such a line, though: the comment prefix and leading whitespace are stripped
before the check, so a line-leading `swagger:` inside an indented markdown code
block still ends the block. Keep annotation-looking lines out of the verbatim
body, or place them before the `|` annotation.

{{% notice style="note" %}}
This reframes [go-swagger#3211](https://github.com/go-swagger/go-swagger/issues/3211):
markdown is authored **explicitly** via `swagger:description |`, never recovered
from ambient godoc prose. A plain doc comment stays plain — godoc and the spec
keep their separate conventions.
{{% /notice %}}

## What's next

- [Overriding titles & descriptions]({{% relref "overriding-titles-and-descriptions" %}})
  — the `swagger:title` / `swagger:description` overrides this builds on.
- [Single-line comments as descriptions]({{% relref "single-line-comments" %}})
  — route short comments to the description without a marker.
