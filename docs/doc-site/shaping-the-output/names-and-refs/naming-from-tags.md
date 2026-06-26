---
title: Naming from struct tags
weight: 20
description: |
  Derive property, parameter and header names from a struct tag other than
  json (form, xml, …) via NameFromTags.
---

By default codescan derives a field's spec name from its `json:` tag (then the
Go field name). `Options.NameFromTags` lets you choose which struct-tag types
supply the name, in precedence order — handy when your structs are tagged for
another binding library (for example gin's `form:`). It applies everywhere a
name is derived from a field: schema properties, parameters, and response
headers. The model below tags every field with both `json:` and `form:`:

{{< code file="shaping/naming-from-tags/naming.go" lang="go" region="model" >}}

Scanned with the default (`["json"]`) and with `["form","json"]`, the property
names differ — `form:` wins because it is listed first:

{{< compare left="shaping/naming-from-tags/testdata/default.json" leftlabel="Default (json)"
            right="shaping/naming-from-tags/testdata/form.json" rightlabel="NameFromTags: [form, json]" >}}

```go
codescan.Run(&codescan.Options{
    Packages:     []string{"./..."},
    ScanModels:   true,
    NameFromTags: []string{"form", "json"},
})
```

The first listed tag that supplies a usable name wins; a tag that is absent or
carries only options (e.g. `,omitempty`) is skipped and the next is tried. An
explicit empty list (`NameFromTags: []string{}`) consults no tag and falls back
to the Go field name.

{{% notice style="info" %}}
**Name only.** `NameFromTags` changes only the *name*. The encoding/json
directives — `json:"-"` (exclude), `,omitempty`, `,string` — are always read
from the `json` tag, whatever names the field. Targeted renames (the `name:`
keyword, `swagger:name`, and `swagger:model {name}`) still take precedence over
any tag-derived name.
{{% /notice %}}
