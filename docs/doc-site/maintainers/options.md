---
title: "Options reference"
weight: 25
description: |
  Every field of codescan.Options — its type, default, and effect — grouped by
  concern and cross-linked to the how-to that shows it in action.
---

[`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options)
is the single configuration struct passed to
[`codescan.Run`](https://pkg.go.dev/github.com/go-openapi/codescan#Run). **The
zero value is a valid configuration** — every flag defaults to `false`, every
slice/map to `nil`, every numeric tunable to its built-in default. You set only
what you need.

This page is the field-by-field catalogue. The
[godoc](https://pkg.go.dev/github.com/go-openapi/codescan#Options) is the
normative source; each field here links to the
[how-to guide]({{% relref "/shaping-the-output" %}}) that shows it on real
input where one exists.

{{% notice style="note" %}}
codescan never writes to stdout or stderr. Every scan-time observation — a
dropped construct, a rename, a prune — flows through the `OnDiagnostic`
callback. See [Diagnostics & observability](#diagnostics--observability) below.
{{% /notice %}}

## Inputs & scope

What gets loaded and which packages and types are in play. See
[Scope & discovery]({{% relref "/shaping-the-output/scope-and-discovery" %}}).

| Option | Type | Default | Effect |
|--------|------|---------|--------|
| `Packages` | `[]string` | `nil` | Package patterns to scan (e.g. `./...`), resolved relative to `WorkDir`. |
| `WorkDir` | `string` | `""` (cwd) | Working directory the package patterns and module resolution are rooted at. |
| `BuildTags` | `string` | `""` | Go build tags to activate while loading, so tag-guarded source is scanned. See [Build tags]({{% relref "build-tags" %}}). |
| `Include` | `[]string` | `nil` | Allow-list of package path patterns; when non-empty only matching packages are scanned. See [Scoping the scan]({{% relref "scoping-the-scan" %}}). |
| `Exclude` | `[]string` | `nil` | Deny-list of package path patterns, applied after `Include`. See [Scoping the scan]({{% relref "scoping-the-scan" %}}). |
| `IncludeTags` | `[]string` | `nil` | Allow-list filtering routes/operations by their swagger `tags`. |
| `ExcludeTags` | `[]string` | `nil` | Deny-list filtering routes/operations by their swagger `tags`. |
| `ExcludeDeps` | `bool` | `false` | Skip types reached through module dependencies, keeping the scan to first-party packages. |
| `ScanModels` | `bool` | `false` | Also emit a definition for every `swagger:model` type, not just route-reachable ones. See [When the scanner emits a type]({{% relref "type-discovery" %}}). |
| `PruneUnusedModels` | `bool` | `false` | With `ScanModels`, drop discovered definitions not transitively reachable from a path, shared response/parameter, or `InputSpec` root. Runs before name reduction; `InputSpec` definitions are pinned. No-op without `ScanModels`. See [Pruning unused models]({{% relref "pruning-unused-models" %}}). |
| `InputSpec` | `*spec.Swagger` | `nil` | Base document to overlay scanned discoveries onto; its definitions are pinned and seed pruning roots. See [Overlaying a spec]({{% relref "overlaying-a-spec" %}}). |

## Names & references

How definitions are named and how `$ref`s render. See
[Names & `$ref`s]({{% relref "/shaping-the-output/names-and-refs" %}}).

| Option | Type | Default | Effect |
|--------|------|---------|--------|
| `NameFromTags` | `[]string` | `nil` (⇒ `["json"]`) | Ordered struct-tag types a property/parameter/header name is derived from; first that supplies a name wins. Explicit empty slice ⇒ Go field name. Only the name — `json` encoding directives (`-`, `,omitempty`, `,string`) always come from `json`. See [Naming from struct tags]({{% relref "naming-from-tags" %}}). |
| `RefAliases` | `bool` | `false` | Render Go type aliases as a first-class `$ref` (via `swagger:model`) instead of expanding them inline. See [Alias rendering]({{% relref "alias-rendering" %}}). |
| `TransparentAliases` | `bool` | `false` | Make aliases fully transparent — never creating a definition. See [Alias rendering]({{% relref "alias-rendering" %}}). |
| `NameConcatBudget` | `float64` | `0` (⇒ `0.65`) | Readability cutoff `[0,1]` for the package-segment concatenation that deconflicts colliding definition names; lower scores are more readable. A group whose best concat scores above the budget is a candidate for the hierarchical fallback. See [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}). |
| `EmitHierarchicalNames` | `bool` | `false` | For the rare collision group whose best flat concat exceeds `NameConcatBudget`, emit nested container definitions (`#/definitions/<pkg>/<Name>`) instead of a long flat concat, with an explanatory diagnostic. The always-correct flat concat is the default. See [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}). |
| `EmitRefSiblings` | `bool` | `false` | Emit a `$ref`'d field's description and vendor extensions as direct `$ref` siblings (`{$ref, description, x-*}`) instead of an `allOf` wrap. Validations/externalDocs still force a compound. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |
| `SkipAllOfCompounding` | `bool` | `false` | Never emit an `allOf` compound for a `$ref`'d field. Validations/externalDocs are dropped (description/extensions too, unless `EmitRefSiblings` keeps them as siblings); each drop raises a diagnostic. `required` is unaffected. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |
| `DescWithRef` | `bool` | `false` | **Deprecated** — prefer `EmitRefSiblings`. In the description-only case, wrap the `$ref` in a single-arm `allOf` to preserve the description (strict draft-4 shape). No-op when `EmitRefSiblings` is set. See [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}). |

## Titles & descriptions

The human-readable text the spec carries. See
[Titles & descriptions]({{% relref "/shaping-the-output/titles-and-descriptions" %}}).

| Option | Type | Default | Effect |
|--------|------|---------|--------|
| `SingleLineCommentAsDescription` | `bool` | `false` | Route every single-line doc comment to `description`, never to `title`/`summary` (the first-sentence convention otherwise applies). Multi-line comments keep the title/description split. See [Single-line comments as descriptions]({{% relref "single-line-comments" %}}). |
| `AfterDeclComments` | `bool` | `false` | Let swagger annotations live inside a struct body or as a trailing comment, in addition to the doc comment above the declaration, so the godoc stays clean. v0.36 scope: type declarations (struct inside-body + alias trailing comment). See [Keeping annotations out of the godoc]({{% relref "keeping-annotations-out-of-the-godoc" %}}). |
| `CleanGoDoc` | `bool` | `false` | Strip godoc doc-link brackets from generated `title`/`description` (humanizing unresolved ones, dropping reference-definition lines, recomposing resolved links to each schema's exposed name). Applies only to godoc-derived prose; overrides are untouched. See [Cleaning godoc doc-links]({{% relref "cleaning-godoc-doc-links" %}}). |

## Field types, formats & extensions

How an individual property renders. See
[Field types & formats]({{% relref "/shaping-the-output/field-types-and-formats" %}}).

| Option | Type | Default | Effect |
|--------|------|---------|--------|
| `SetXNullableForPointers` | `bool` | `false` | Emit `x-nullable: true` on pointer-typed fields. See [Nullable pointers]({{% relref "nullable-pointers" %}}). |
| `SkipExtensions` | `bool` | `false` | Suppress all `x-go-*` vendor extensions in the output. See [Vendor extensions]({{% relref "vendor-extensions" %}}). |
| `EmitXGoType` | `bool` | `false` | Stamp an `x-go-type` extension (fully-qualified originating Go type) on every emitted definition, for round-tripping a spec back to its Go types. Suppressed under `SkipExtensions`. See [Vendor extensions]({{% relref "vendor-extensions" %}}). |
| `SkipEnumDescriptions` | `bool` | `false` | Keep the per-enum-value const-name mapping (from `swagger:enum`) out of the `description`, exposing it only via the `x-go-enum-desc` extension. Suppressed entirely under `SkipExtensions`. |

## Diagnostics & observability

Channels for what the scan observed; these do not change the output spec.

| Option | Type | Default | Effect |
|--------|------|---------|--------|
| `OnDiagnostic` | `func(Diagnostic)` | `nil` | Invoked once per diagnostic in source order (parser warnings, validation failures, prunes, renames). Diagnostics never block the build — invalid constructs are dropped from the spec while their explanation flows here. The only output channel. **Experimental** while LSP integration matures. |
| `OnProvenance` | `func(Provenance)` | `nil` | Invoked once per anchor node in the produced spec, carrying its JSON pointer and the source position of the Go construct that produced it. Never blocks the build. **Experimental** while LSP/TUI integration matures. |
| `Debug` | `bool` | `false` | **Deprecated, ignored.** The legacy stderr debug logger was retired; wire `OnDiagnostic` instead. Retained for API compatibility. |

## See also

- [Annotations]({{% relref "annotations" %}}) — the `swagger:*` vocabulary the
  scanner reads from comments.
- [Keyword reference]({{% relref "keywords" %}}) — the `keyword: value` forms
  inside annotation bodies.
- [Shaping the output]({{% relref "/shaping-the-output" %}}) — task-oriented
  how-tos that put these options to work on real input.
