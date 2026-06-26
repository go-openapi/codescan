---
title: Maintainers
weight: 50
description: |
  The complete, normative reference for the codescan annotation language —
  every annotation, every keyword, the embedded sub-languages, and the formal
  grammar the parser implements.
---

This section is the **reference compendium**: the precise, exhaustive
description of the language codescan parses and the options that drive it. It is
written for people who want the full contract — annotation authors looking up an
exact rule, library callers looking up an option, and contributors porting,
extending, or debugging the parser.

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
- **[Options]({{% relref "options" %}})** — every field of `codescan.Options`:
  its type, default, and effect, cross-linked to the how-to that shows it in
  action. The library-caller reference.
- **[Sub-languages]({{% relref "sub-languages" %}})** — the smaller languages
  embedded inside annotation bodies (`Parameters:` / `Responses:` grammars,
  YAML surfaces, prose classification).
- **[Grammar]({{% relref "grammar" %}})** — the formal ISO-14977 EBNF the
  parser implements, from comment preprocessing through the typed walker.
