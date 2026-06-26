---
title: Single-line comments as descriptions
weight: 20
description: |
  Route every single-line doc comment to description instead of title/summary
  with the SingleLineCommentAsDescription option.
---

By the first-sentence convention, a **single-line** doc comment that ends in
punctuation becomes the object's `title` (on a model or the `info` block) or
`summary` (on an operation); without trailing punctuation it is a `description`.
That is the right default for most codebases, but some use single-line comments
purely as prose — and then a stray period silently promotes the line to a title.

`Options.SingleLineCommentAsDescription` opts out of the promotion: a single-line
comment is **always** a `description`, never a `title` / `summary`.

```go
codescan.Run(&codescan.Options{
    Packages:                       []string{"./..."},
    ScanModels:                     true,
    SingleLineCommentAsDescription: true,
})
```

The witness pairs a model and an operation, each with a single-line comment that
ends in a period:

{{< code file="shaping/singleline/singleline.go" lang="go" region="model" >}}

{{< code file="shaping/singleline/singleline.go" lang="go" region="route" >}}

The same source, scanned both ways — the comment moves from `title` / `summary`
to `description` uniformly:

{{< compare left="shaping/singleline/testdata/off.json" leftlabel="Default — title / summary"
            right="shaping/singleline/testdata/on.json" rightlabel="SingleLineCommentAsDescription: true" >}}

{{% notice style="info" %}}
**Only the single-line case changes.** A multi-line doc comment keeps the
existing split — the first line (or the paragraph before the first blank line)
stays the `title`, and the rest becomes the `description`. Reach for this option
when your house style writes one-line prose descriptions and you don't want them
landing in `title` / `summary`; otherwise leave it off (the default) and write a
two-line comment, or drop the trailing period, when you want a description.
{{% /notice %}}

## What's next

- [Document metadata]({{% relref "/tutorials/document-metadata" %}}) — the
  `info` title/description split this option also governs.
- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) — the
  title/description convention on a `swagger:model`.
- [Vendor extensions]({{% relref "vendor-extensions" %}}) —
  other `Options` knobs that reshape the emitted spec.
