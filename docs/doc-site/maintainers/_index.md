---
title: Maintainers
weight: 100
description: |
  The complete, normative reference for the codescan annotation language.

  Every annotation, every keyword, the embedded sub-languages, the formal
  grammar the parser implements, and how the commands are put together.
---

This section is the **reference compendium**: the precise, exhaustive
description of the language codescan parses, and of the tools built around it.
It is written for people who want the full contract — annotation authors looking
up an exact rule, and contributors porting, extending, or debugging the parser.

Looking up an *option* rather than an annotation? That moved: the
[Options reference]({{% relref "options-reference" %}}) lives under
[Usage]({{% relref "/usage" %}}), beside the flag and configuration-key
spellings of the same knobs.

If you are learning codescan by example, start with the
[Tutorials]({{% relref "/tutorials" %}}) instead — they show the same concepts
as runnable Go with the spec they produce, side by side. The
[Annotation index]({{% relref "/annotation-index" %}}) cross-references every
annotation to both its tutorial and its entry here.

## The reference documents

{{< children type="card" description="true" >}}

- **[Annotations]({{% relref "annotations" %}})** — the `swagger:*` vocabulary:
  what each annotation does, where it attaches, its argument shape, and the
  keywords it admits. The author-facing normative reference.
- **[Keywords]({{% relref "keywords" %}})** — the per-keyword reference card:
  every `keyword: value` form, its value shape, and the contexts where it is
  legal.
- **[Sub-languages]({{% relref "sub-languages" %}})** — the smaller languages
  embedded inside annotation bodies (`Parameters:` / `Responses:` grammars,
  YAML surfaces, prose classification).
- **[Grammar]({{% relref "grammar" %}})** — the formal ISO-14977 EBNF the
  parser implements, from comment preprocessing through the typed walker.
- **[The commands]({{% relref "commands" %}})** — how the three CLI tools are put
  together: why each lives where it does, where their shared flag surface is
  declared, and what keeps it whole.
