---
title: Titles & descriptions
weight: 30
description: |
  Shape the human-readable text — override godoc with API-facing title and
  description, route single-line comments to the description, keep annotations
  out of the godoc, and clean godoc doc-links out of generated prose.
---

The same Go doc comments feed both `pkg.go.dev` and your API documentation, and
the two audiences rarely want the exact same words. These knobs let you keep a
concise godoc while curating the `title` / `description` text the spec carries.

{{< children type="card" description="true" >}}
