# `internal/builders/responses` — maintainers' guide

Builds OAS v2 response entries (`swagger:response`) including the
response body and any header fields. One `Builder` per
declaration; one Walk pass per field doc-comment.

## Sections

- [§overview](#overview) — package shape and per-file responsibilities
- [§builder](#builder) — `Builder`, `Build`, the build chain
- [§in-discriminator](#in-discriminator) — `in:` as body/header annotation switch
- [§file-body](#file-body) — `swagger:file` and the body-only gate
- [§dispatch](#dispatch) — Walker handlers wiring for headers
- [§typable](#typable) — `responseTypable`, the `refAttempted` mechanism, body vs header
- [§alias-handling](#alias-handling) — when to `$ref` vs expand
- [§quirks-history](#quirks-history) — resolved quirks (Stream M)

---

## <a id="overview"></a>§overview — files and responsibilities

| File | Contents |
|------|----------|
| `responses.go` | `Builder`, build chain (`Build` → `buildFromType` → `buildNamedType` / `buildAlias` → `buildFromStruct` → `processResponseField` → `buildFromField*`), `buildOption` helper |
| `doc_signals.go` | `fieldDocSignals` + `scanFieldDocSignals`: pre-walk extraction of `in:`, `swagger:ignore`, `swagger:file`, `swagger:strfmt` from a field's doc comment; closed-vocabulary `in:` validation |
| `walker.go` | `applyBlockToDecl` (top-level decl), `applyBlockToHeader` + `dispatchHeaderLevel0` + `walkHeaderItemsLevel`: the grammar Walker dispatch for per-header validations / extensions |
| `typable.go` | `responseTypable` (the `ifaces.SwaggerTypable` adapter) + `headerValidations` (the `ifaces.ValidationBuilder` adapter) + `SimpleSchemaProbe` |
| `errors.go` | `ErrResponses` sentinel |

The builder embeds `*common.Builder` (Ctx, Decl, PostDeclarations,
diagnostic sink, ParseBlocks cache, MakeRef). See
[../common](../common) for the common-builder rationale.

This package's shape closely mirrors
[../parameters](../parameters) — the chain is structurally the
same. Divergences are called out below.

## <a id="builder"></a>§builder — the build chain

`Build(responses)` looks up the response by name (from
`r.Decl.ResponseNames()`), runs `applyBlockToDecl` to capture the
top-level description, then calls `buildFromType` on the declared
type. `buildFromType` unwraps pointers, dispatches named types and
aliases. Unlike parameters, **anonymous types are rejected**:
`responses_test.go` documents the rationale — the top-level
response-as-alias case under default mode is deferred to v2.

For each non-embedded exported field, `processResponseField` runs:

1. Find the AST field via `resolvers.FindASTField` (no AST → skip).
2. Pre-scan the doc-comment signals via `scanFieldDocSignals` (uses
   the `common.Builder` parse cache).
3. If `swagger:ignore` → skip.
4. Resolve JSON tag name. If `json:"-"` → skip.
5. Resolve `in:` (default `header`; see [§in-discriminator](#in-discriminator)).
6. **File-body gate**: if `swagger:file` AND `in==body` → set
   `resp.Schema = {type:"file"}` and skip the field build (see
   [§file-body](#file-body)).
7. Otherwise build the field's type into either `resp.Schema`
   (body) or a header value (non-body), through `responseTypable`.
8. Apply `swagger:strfmt <name>` override when set.
9. Walk the doc-comment block via `applyBlockToHeader` (description,
   validations, items, extensions — no `required:`).
10. If `in != body`, register the header on `resp.Headers[name]`.

After all fields are processed, `buildFromStruct` deletes header
entries for fields that were skipped (the `seen` map).

## <a id="in-discriminator"></a>§in-discriminator — `in:` as body/header annotation switch

OAS v2 has **no `in` field on the Response Object** — the location
exists at the parameter level only. This package overloads `in:` on
response fields to tell apart "this field is the body" from "this
field is a header" within a single Go struct:

- `in: body` → field's type populates `resp.Schema`
- `in: header` (or absent → defaults to `header`) → field becomes
  one entry in `resp.Headers`
- `in: query | path | formData` → recognised but unusual; not a
  response location per OAS v2. Treated as non-body (header-like)
  with no special handling.

### Default — implicit header

Pre-Stream M the implicit case fell into the `in != "body"` branch
by accident: an empty string is not `"body"` and so behaved
identically to `header`. Q1 made this default **explicit** in code
— `inHeader` is assigned when `!signals.inSet`. Observable behaviour
is unchanged; the implicit fall-through is gone.

### Invalid `in:` values

An `in:` line with a non-vocabulary value (e.g. `in: cookie`) emits
a `CodeInvalidAnnotation` warning via `Warnf` and **defaults to
header**. Author misuse surfaces in diagnostics without breaking the
build.

### Why line-scan instead of property

Same reason as parameters — `in:` may appear on either side of an
annotation. See
[../parameters/README.md#in-discriminator](../parameters/README.md#in-discriminator).

## <a id="file-body"></a>§file-body — `swagger:file` annotation

`swagger:file` on a response field marks the entire response body as
a file payload (image, PDF, raw bytes). Per OAS v2, the allowed
**header** types are `{string, number, integer, boolean, array}` —
`file` is **forbidden on a header**. The annotation must therefore
land on the body field; on a header it is misuse.

### The Q3 gate

Pre-Q3 the file branch fired unconditionally and rewrote
`resp.Schema = {file, ""}` even when `in != body`, silently
corrupting the body schema from a header-positioned annotation.
Q3 gates the branch on `in == inBody`:

```go
useFileBody := signals.file && in == inBody
```

When `signals.file` fires under a non-body `in`, the dispatcher
emits a `CodeUnsupportedInSimpleSchema` warning and falls through
to the regular field build, treating the field like any other
header. The body schema is untouched.

## <a id="dispatch"></a>§dispatch — Walker dispatch for headers

`applyBlockToHeader` is the per-field entry point for header
fields. Three phases:

1. **Prose** → `header.Description`.
2. **Level-0 dispatch** → `dispatchHeaderLevel0` wires Walker
   callbacks via the `handlers` package. Shape mirrors parameters'
   level-0 dispatcher with one omission:
   - `Number`, `Integer`, `String` (Pattern + CollectionFormat),
     `Raw` (default/example/enum) → identical to parameters
   - `Bool` → `handlers.UniqueBool(valid)` only — **no `required:`
     write**. The OAS v2 Header object simply doesn't carry
     `required:`.
   - `Extension` → `handlers.Extension(header)` — v1 had no
     header-side extension support at all; the grammar migration
     closes that gap. User-authored `Extensions:` block entries
     land on the header.
3. **Items-level dispatch** → `walkHeaderItemsLevel` for each
   `(level, items)` pair returned by `collectHeaderItemsLevels`
   (1-indexed depths matching grammar's `Property.ItemsDepth`).

`applyBlockToDecl` is the top-level (response-level) entry point.
It only writes `resp.Description` from prose — the v1
`SectionedParser` only accepted description at the top level, no
taggers. Property-level keywords on the top-level decl are silently
ignored.

### Header errSink semantics

Unlike `dispatchParamLevel0`, the response `Raw` handler is called
with a nil errSink: malformed `default:` / `example:` for a header
produces a parser diagnostic but is **not promoted to a build
error**. Headers are non-critical metadata; failing the build over
a malformed example would be surprising.

## <a id="typable"></a>§typable — `responseTypable`, body vs header

`responseTypable` adapts a header or body slot to
`ifaces.SwaggerTypable`. Single struct, polymorphic by `ht.in`:

- **Body** (`in == "body"`): `Schema()` materialises and returns
  `resp.Schema`. `Typed()` writes to the header struct, but body
  callers use `Schema()` directly. `Items()` walks `resp.Schema.Items`.
- **Header** (`in == "header"` or anything non-body): `Typed()`
  writes to the embedded `SimpleSchema` on the header.
  `Items()` builds the items chain on the header side.

### The Q2 `refAttempted` mechanism

OAS v2 response headers cannot carry `$ref`. Pre-Q2, `SetRef` wrote
the ref onto `response.Schema.Ref` unconditionally — which
corrupted the body schema with a **header field's reference** when a
header field aliased to a named type. The Q2 fix:

```go
func (ht responseTypable) SetRef(ref oaispec.Ref) {
    if ht.in == inBody {
        ht.Schema().Ref = ref
        return
    }
    if ht.refAttempted != nil {
        *ht.refAttempted = true
    }
}
```

Non-body `SetRef` calls no-op the ref write and flip a flag on a
caller-owned `bool`. `HasRef()` reads the flag for
`SimpleSchemaProbe` so the schema builder's exit validator can
detect the violation, emit `CodeUnsupportedInSimpleSchema`, and call
`ResetForViolation()` (which wipes the header's SimpleSchema back to
empty).

The flag is **caller-owned** (passed by pointer) so a single
`responseTypable` value can be shared across recursion levels
without mutation through the `ifaces.SwaggerTypable` value-receiver
methods.

### `SimpleSchemaProbe` implementation

- `SimpleSchemaShape()` — returns the header's embedded SimpleSchema
- `HasRef()` — true if a non-body `SetRef` attempt was made
- `ResetForViolation()` — wipes the header's SimpleSchema back to `{}`

## <a id="alias-handling"></a>§alias-handling

The responses builder shares the alias-handling contract with the
schema and parameters builders — annotation gates first-class
alias identity at use sites; `TransparentAliases` overrides at use
sites; mode flags only shape the alias's own definition. The full
rule lives in
[schema/README.md §aliases](../schema/README.md#aliases); below
captures the responses-specific reach contexts.

Two responses-specific use-site handlers:

**Top-level alias annotated `swagger:response` (`buildAlias`).**
The alias is **transparent re: model creation** in all modes —
neither the alias nor any chain link of its backing struct
surfaces in `definitions`. The fields of the unaliased target
become the response's body and headers. The implementation just
forwards `tpe.Rhs()` to `buildFromType`; recursion handles chains
naturally. No mode-specific behaviour at this layer.

**Alias as a field type within a response struct
(`buildFieldAlias`).** Three branches:

- `TransparentAliases=true` — dissolve via the schema sub-builder.
- Non-body field (response header) — SimpleSchema target cannot
  carry `$ref`; always expand to the unaliased target via
  `types.Unalias`. Annotation has no effect at non-body sites.
- Body field — annotation gate decides:
  - With `swagger:model`: `MakeRef` to the alias's decl —
    `$ref: <AliasName>` preserves the alias name and the alias
    surfaces in `definitions` via the discovery loop.
  - Without `swagger:model`: dissolve via `types.Unalias` (full
    chain collapse in one step) and build from the unaliased
    target.

The body-field gate is mode-agnostic: Default and Ref produce the
same `$ref` target — annotation alone decides. The mode flag
shapes only the alias decl's own definition downstream.

## <a id="quirks-history"></a>§quirks-history — resolved quirks

The Stream M merge-readiness pass closed a handful of subtle quirks
in this builder. Recorded for archaeology — none should resurface
under the current dispatch.

- **Q1: implicit header default** — `in:` absent now resolves
  explicitly to `inHeader`. Pre-Q1 the empty string fell through
  `in != "body"` by accident. Observable behaviour unchanged.
- **Q2: `$ref` leaking into response body** — non-body `SetRef`
  calls no longer write to `response.Schema.Ref`. The
  `refAttempted` flag plumbs the attempt to the SimpleSchema exit
  validator. See [§typable](#typable).
- **Q3: `swagger:file` on a header** — gated to `in == inBody` only;
  misuse emits a diagnostic and falls through to the regular field
  build.
- **Q4: alias expansion at field-level** — added the
  `In() != inBody || !RefAliases()` gate to `buildFieldAlias` so
  non-body header aliases expand instead of leaking a `$ref`. Aligns
  with parameters' gate exactly.
- **Q5 (closed, no change)** — the previously-suspected TODO around
  alias deduplication turned out to be obsolete; no code change
  needed.
- **Q6: alignment with parameters** — `buildFieldAlias` and
  `processResponseField` now mirror parameters' shape so the two
  builders behave consistently at field-level alias sites and
  during dispatch.

### Deferred to v2

- **Top-level response-as-alias under default mode**:
  `buildFromType`'s default branch rejects anonymous types with
  `"anonymous types are currently not supported for responses"`. A
  top-level alias to an anonymous struct under default mode crashes
  here. Reproduces in `fixtures/enhancements/alias-response-shapes`.
  Out of scope for v1.
