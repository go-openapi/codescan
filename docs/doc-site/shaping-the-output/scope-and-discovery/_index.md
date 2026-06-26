---
title: Scope & discovery
weight: 10
description: |
  Choose what gets scanned and which definitions land in the spec — package
  patterns and filters, when a type is emitted, pruning unreferenced models,
  overlaying an existing document, and build constraints.
---

These knobs decide the *inputs and the surface* of the scan: which packages
codescan reads, which types become definitions, and how that set is trimmed or
merged before anything is rendered.

{{< children type="card" description="true" >}}
