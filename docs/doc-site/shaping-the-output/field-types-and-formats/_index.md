---
title: Field types & formats
weight: 40
description: |
  Tune how an individual field renders — force a conformant format, mark
  pointer fields nullable, and control the x-go-* vendor extensions codescan
  emits.
---

These knobs act at the level of a single property: the `format` it carries,
whether a pointer is advertised as nullable, and the vendor extensions that
record its Go provenance.

{{< children type="card" description="true" >}}
