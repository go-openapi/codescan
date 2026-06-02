# scanner — maintainer notes

This document is the long-form companion to the scanner package code.
The source files keep godoc concise; complex invariants, design
trade-offs, and known quirks live here.

The `scanner` package owns package loading and entity discovery. It
turns a set of Go package patterns into a `ScanCtx` that exposes the
classified per-decl inventory (meta, routes, operations, models,
parameters, responses) consumed by the builder layer.

---

## Table of contents

- [§options](#options) — `Options.DescWithRef` shape and rationale
- [§descwithref](#descwithref) — the description-only-decoration
  $ref shape and why it has a flag
- [§diagnostics](#diagnostics) — `OnDiagnostic` contract and
  experimental-API caveat
- [§model-lookup](#model-lookup) — `GetModel` vs `FindModel` —
  pure read vs implicit registration
- [§classifier](#classifier) — `detectNodes` bitmask semantics and
  struct-annotation exclusivity
- [§quirks-open](#quirks-open) — deferred follow-ups

---

## <a id="options"></a>§options — `Options` overview

`Options` is the externally-visible configuration struct. It is
re-exported from the package root as `codescan.Options`. The default
zero value is a valid configuration: every flag defaults to false and
every slice/map defaults to nil.

Most fields are simple toggles (scope inclusion, debug, vendor
extension suppression). Two fields carry non-trivial semantics that
warrant the inline godoc and the deeper notes below:

- `DescWithRef` — controls the `$ref` shape used when a struct field
  resolves to a named type and its only decoration is a description.
  See [§descwithref](#descwithref).
- `OnDiagnostic` — diagnostic callback hook. See
  [§diagnostics](#diagnostics).

## <a id="descwithref"></a>§descwithref — description-only-decoration $ref shape

When a struct field's Go type resolves to a named type (so the spec
emits a `$ref` to its definition) and its only field-level
decoration is a description (no validations, no user-authored
vendor extensions), the spec has two possible shapes:

1. **Bare $ref** — `{$ref: ...}`. The field's description is
   dropped. This is the conservative default when `DescWithRef` is
   false.
2. **Single-arm allOf** — `{description: "...", allOf: [{$ref}]}`.
   The description is preserved by wrapping the `$ref` in a
   single-arm `allOf` compound. This is JSON-Schema-draft-4 correct
   for sibling description.

`DescWithRef=true` opts into the second shape. The default is false
because the bare-`$ref` shape interoperates more broadly with
Swagger 2.0 tooling that does not implement the `allOf` compound.

When the field also carries validation overrides (pattern, enum,
example, etc.) or user-authored vendor extensions, the `allOf`
compound is mandatory regardless of `DescWithRef` — the override
would be lost otherwise.

## <a id="diagnostics"></a>§diagnostics — `OnDiagnostic` callback

`Options.OnDiagnostic`, when non-nil, is invoked for every
`grammar.Diagnostic` the builder layer records: lexer/parser
warnings, semantic-validation failures from the validations package,
and any future diagnostic class wired into the builder pipeline.

Contract:

- The callback fires **once per diagnostic, in source order**.
- Diagnostics **never block the build**. An invalid construct is
  silently dropped from the output spec; the explanation flows
  through this channel instead.
- The callback may be called from any per-decl builder; it is the
  caller's responsibility to make it goroutine-safe if the consumer
  ever drives `codescan.Run` concurrently (today it is single-
  goroutine, but the callback contract makes no such guarantee).

The diagnostic surface is **experimental**. Once the LSP integration
matures the shape is expected to grow: typed severity classes,
structural deduplication, per-position provenance. Callers that
adopt `OnDiagnostic` today should treat the signature as subject to
breaking change in a future minor release.

`ScanCtx.OnDiagnostic` returns the user-supplied callback verbatim;
builders pipe diagnostics through it via `common.Builder.RecordDiagnostic`.

## <a id="model-lookup"></a>§model-lookup — `GetModel` vs `FindModel`

`ScanCtx` exposes two lookup helpers with similar signatures but
different side-effect contracts. The choice between them is
load-bearing for the shape of the emitted spec.

### `GetModel(pkgPath, name)` — pure read

Looks up a model decl across three sources, in order:

1. `Models` — decls annotated with `swagger:model`. Always emitted
   as top-level definitions regardless of lookup.
2. `ExtraModels` — decls discovered as dependencies of other
   emitted shapes. Already enqueued for top-level emission.
3. `FindDecl` — fall through to a syntactic search over the
   loaded packages.

No side effect. A `FindDecl` hit through `GetModel` does **not**
register the decl in `ExtraModels`. Callers that want the lookup
to also surface the decl as a top-level definition must follow up
with `AddDiscoveredModel` explicitly.

### `FindModel(pkgPath, name)` — implicit registration

The older sibling of `GetModel`. It does the same three-source
lookup, but a `FindDecl` hit also writes the decl into `ExtraModels`
as a side effect.

`FindModel` is deprecated. The implicit registration surprises
readers and pulls stdlib types (notably `time.Time`,
`json.RawMessage`) into the spec's top-level definitions when they
should be inlined where referenced. Builders that need the
registration should use the explicit `GetModel` + `AddDiscoveredModel`
pair.

### `AddDiscoveredModel` — explicit registration

Registers a decl in `ExtraModels`. No-op for decls already in
`Models` (annotated decls are emitted unconditionally — registering
them as discovered would create a Models↔ExtraModels bouncing loop
in the spec orchestrator's `joinExtraModels` pass). Nil and
Ident-less decls are silently ignored, which is defensive against
the scanner emitting partial decls during error recovery.

## <a id="classifier"></a>§classifier — `detectNodes` bitmask

`TypeIndex.detectNodes` scans every comment group in a file and
returns a bitmask of detected annotation kinds. Each kind drives
downstream processing:

| Bit | Annotation | Downstream |
|---|---|---|
| `metaNode` | `swagger:meta` | file-level meta block |
| `routeNode` | `swagger:route` | path-level route annotations |
| `operationNode` | `swagger:operation` | path-level operation annotations |
| `modelNode` | `swagger:model` | per-decl model registration |
| `parametersNode` | `swagger:parameters` | per-decl parameter registration |
| `responseNode` | `swagger:response` | per-decl response registration |

`route`, `operation`, and `meta` accumulate freely across comment
groups in a file. The three struct-level annotations (`model`,
`parameters`, `response`) are **mutually exclusive within a single
comment group** — a struct cannot simultaneously be a model and a
parameters bag, for instance. `checkStructConflict` enforces the
rule per comment group and returns an error if the constraint is
violated.

The annotation vocabulary recognised by the classifier is a closed
set. Unknown annotations beginning with `swagger:` raise a
classifier error. A handful of annotation tokens (`strfmt`, `name`,
`enum`, `default`, `alias`, `type`, …) are recognised but produce
no bit — they are field-level decorations that downstream builders
parse out of the comment block directly.

## <a id="quirks-open"></a>§quirks-open — deferred follow-ups

- **`FindModel` deprecation.** The deprecated alias is still on the
  `ScanCtx` surface for in-tree callers. Once every builder has been
  audited and migrated to the `GetModel` + `AddDiscoveredModel` pair,
  the deprecated method can be removed in a future major release.
- **Recognised-but-unused annotation tokens.** `detectNodes`
  recognises a list of field-level tokens (`strfmt`, `name`,
  `discriminated`, `file`, `enum`, `default`, `alias`, `type`,
  `allOf`, `ignore`) only to avoid raising the "unknown annotation"
  error. Promoting them to per-file bits would let downstream
  builders skip whole files that carry no decorations — an
  optimisation, not a correctness change.
- **`shouldAcceptTag` precedence.** When both `includeTags` and
  `excludeTags` are populated, `includeTags` wins (a tag in
  `includeTags` admits the operation even if it also appears in
  `excludeTags`). This is deliberate but easy to mis-read; an
  explicit "the include list takes precedence" doc on `Options`
  would help callers, but the field-level prose is already dense.
