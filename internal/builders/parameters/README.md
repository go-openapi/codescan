# `internal/builders/parameters` — maintainers' guide

Builds OAS v2 parameter entries (`swagger:parameters`) and writes them
onto matching operations. One `Builder` per declaration; one
Walk pass per field doc-comment.

## Sections

- [§overview](#overview) — package shape and per-file responsibilities
- [§builder](#builder) — `Builder`, `Build`, the build chain
- [§in-discriminator](#in-discriminator) — how `in:` is read and what it gates
- [§dispatch](#dispatch) — Walker handlers wiring at level-0 and items level
- [§typable](#typable) — `paramTypable`, `SimpleSchemaProbe`, body vs non-body
- [§simple-schema-handoff](#simple-schema-handoff) — when the schema builder runs in SimpleSchema mode
- [§quirks-history](#quirks-history) — resolved quirks (Stream M)

---

## <a id="overview"></a>§overview — files and responsibilities

| File | Contents |
|------|----------|
| `parameters.go` | `Builder`, build chain (`Build` → `buildFromType` → `buildNamedType` / `buildAlias` → `buildFromStruct` → `processParamField` → `buildFromField*`), `buildOption` helper |
| `doc_signals.go` | `fieldDocSignals` + `scanFieldDocSignals`: pre-walk extraction of `in:`, `swagger:ignore`, `swagger:file`, `swagger:strfmt` from a field's doc comment |
| `walker.go` | `applyBlockToField` + `walkParamLevel` + `walkItemsLevel`: the grammar Walker dispatch for per-field validations / extensions |
| `typable.go` | `paramTypable` (the `ifaces.SwaggerTypable` adapter) + `paramValidations` (the `ifaces.OperationValidationBuilder` adapter) + `SimpleSchemaProbe` |
| `errors.go` | `ErrParameters` sentinel |

The builder embeds `*common.Builder` (Ctx, Decl, PostDeclarations,
diagnostic sink, ParseBlocks cache, MakeRef). See
[../common](../common) for the common-builder rationale.

## <a id="builder"></a>§builder — the build chain

`Build(operations)` iterates over the declaration's `swagger:parameters
<opid>` arguments — one struct can attach to many operations — and
calls `buildFromType` for each. The chain unwraps pointers, dispatches
named types and aliases, and ultimately reaches `buildFromStruct`
which walks the struct fields.

For each non-embedded exported field, `processParamField` runs the
following ordered steps:

1. Find the AST field via `resolvers.FindASTField` (no AST → skip).
2. Pre-scan the doc-comment signals via `scanFieldDocSignals` (uses
   the `common.Builder` parse cache so `applyBlockToField`'s later
   walk hits the same parse result).
3. If `swagger:ignore` is present → skip.
4. Resolve the JSON tag name. If ignored (`json:"-"`) → skip.
5. Pick the `in:` location (default `query`; see [§in-discriminator](#in-discriminator)).
6. Build the field's type into the parameter via `buildFromField` —
   the schema builder runs in SimpleSchema mode unless `in==body`
   (see [§simple-schema-handoff](#simple-schema-handoff)).
7. Apply `swagger:strfmt <name>` override when set (collapses the
   resolved shape to `string + format`).
8. Walk the doc-comment block to apply description, validations,
   `required:`, vendor extensions, and items-level validations
   (see [§dispatch](#dispatch)).
9. Force `required: true` for `in: path` (OAS-mandated).
10. Append `x-go-name` extension when the JSON tag differs from the
    Go field name.

The order matters: pre-scanning the doc signals BEFORE building lets
the dispatch pick the right `in:` and forces the SimpleSchema mode
correctly; applying the block AFTER the type build lets validations
override the resolved defaults.

### Embedded fields and inherited `in:`/`required:` (go-swagger#2701)

`buildFromStruct` handles an embedded (anonymous) field by recursing
into its type via `buildFromType` — its promoted fields become
parameters of the outer set. An `in:`/`required:` annotation written on
the **embed itself** applies to every parameter it promotes (the embed
is the natural place to say "all of these are path params"). The
recursion threads that as inherited context via the shared
`common.EmbedInheritance` kernel (`ReadEmbedInheritance` reads the
embed's doc; the schema and responses builders use the same kernel so
the rule is identical everywhere). `processParamField` falls back to it
when a promoted field sets no `in:`/`required:` of its own (the field's
own annotation always wins). The context is saved/restored around each
embed so siblings are unaffected, and it nests — an inner embed without
its own `in:` keeps the outer one. See
[§in-discriminator](#in-discriminator) for the `in:` precedence.

Exportedness is per-field, not per-embed: only exported fields promote
(the product documents the public API surface), but exported fields
reached *through* an unexported embedded type still promote — Go
promotes them and they are reachable on the outer type. Unexported
fields never surface, at any depth.

## <a id="in-discriminator"></a>§in-discriminator — reading `in:` and what it gates

`in:` is the OAS v2 location discriminator —
`query | path | header | body | formData` (closed vocabulary; see
`grammar.NormalizeIn` for the canonical normaliser used by both
`parameters/doc_signals.go` and `responses/doc_signals.go`). It
drives three downstream
decisions:

- **Schema vs SimpleSchema mode**: `in==body` ⇒ full Schema build;
  any other `in` ⇒ SimpleSchema build (see [§simple-schema-handoff](#simple-schema-handoff)).
- **File handling**: `in==formData` + `swagger:file` ⇒ shape collapses
  to `type: file` (no further build).
- **Path required**: `in==path` ⇒ `required: true` forced after
  building, regardless of what the block said.

### Why line-scan instead of property?

`scanFieldDocSignals` reads `in:` by **scanning the doc text line by
line**, not by reading it as a grammar Property. Reason: grammar
attaches pre-annotation lines (e.g. `in: formData` preceding a
`swagger:file` annotation) to the annotation block's prose, not to
its property list. A direct line scan picks up `in:` regardless of
which side of an annotation it appears on. The line scan mirrors
v1's `rxIn` regex semantics:

```
[Ii]n\p{Zs}*:\p{Zs}*(query|path|header|body|formData)(?:\.)?$
```

Default when `in:` is absent: an enclosing embed's inherited `in:` if
any (see the embedded-fields note above), otherwise `query` (OAS v2
convention).

## <a id="dispatch"></a>§dispatch — Walker dispatch at level-0 and items level

`applyBlockToField` is the per-field entry point. It runs three
phases on the parsed grammar block:

1. **Prose** → `param.Description` (with `x-go-enum-desc` lift via
   `resolvers.GetEnumDesc`).
2. **Level-0 dispatch** → `walkParamLevel` → `dispatchParamLevel0`,
   which wires Walker callbacks via the `handlers` package:
   - `Number` → `handlers.Number(valid)` (maximum / minimum / multipleOf)
   - `Integer` → `handlers.Integer(valid)` (minLength / maxLength / minItems / maxItems)
   - `Bool` → `ComposeBool(UniqueBool, paramRequiredBool)` — splits
     `uniqueItems` (parameter-side validation) from `required:`
     (writes straight onto `param.Required`)
   - `String` → `ComposeString(PatternString, CollectionFormatString)`
     — pattern + collectionFormat
   - `Raw` → `handlers.Raw(valid, scheme, errSink)` —
     `default:` / `example:` / `enum:` as raw bodies. `errSink`
     captures the first parse error so malformed default/example
     surface as a build error (see `TestMalformed_DefaultInt` /
     `TestMalformed_ExampleInt` integration tests)
   - `Extension` → `handlers.Extension(param)` — pre-typed YAML
     extensions land directly on the parameter
3. **Items-level dispatch** → `walkItemsLevel` for each
   `(level, items)` pair returned by `collectParamItemsLevels`
   (1-indexed depths matching grammar's `Property.ItemsDepth`).
   Named/aliased array types opt out — parity with v1.

`dispatchParamLevel0` is standalone (not a method) so unit tests can
drive it without constructing a full `Builder`.

### Why `required:` is parameter-specific

Schema writes `required:` onto the **enclosing schema's** Required
slice keyed by name (because a struct field's required-ness belongs
to the parent type, not the field). Parameters write `required:`
straight onto `param.Required`. Headers don't carry `required:` at
all — the OAS v2 Header object simply doesn't have the field.

## <a id="typable"></a>§typable — `paramTypable`, body vs non-body

`paramTypable` adapts a `*spec.Parameter` to `ifaces.SwaggerTypable`.
Two shapes share the type:

- **Body parameter** (`In == "body"`): `Schema()` returns a real
  `*spec.Schema`; type writes go to `param.Schema`, not to the
  parameter's SimpleSchema. `AddExtension` lands on the schema.
- **Non-body** (path / query / header / formData): `Schema()`
  returns nil; type writes go to the parameter's embedded
  `SimpleSchema`. `Items()` builds the items chain on the parameter
  side (not on a body schema).

`Items()` switches on `param.In`: under `body`, returns a
`schema.BodyTypable` that walks down `param.Schema.Items`; under
non-body, returns an `items.NewTypable` chain that walks
`param.Items` directly. The body / non-body split is the same
fundamental gate as everywhere else in this package.

### `SimpleSchemaProbe` implementation

`paramTypable` implements the `schema.SimpleSchemaProbe` interface so
the schema builder can validate the SimpleSchema outcome after its
internal build:

- `SimpleSchemaShape() *oaispec.SimpleSchema` — returns the embedded
  SimpleSchema so the exit validator can inspect Type / Format
- `HasRef()` — non-empty `Ref` is a violation (SimpleSchema forbids
  `$ref`)
- `ResetForViolation()` — wipes SimpleSchema and Ref back to empty
  so the resulting spec is honest about the failed resolution

## <a id="simple-schema-handoff"></a>§simple-schema-handoff — SimpleSchema mode delegation

`buildOption(tpe, typable)` returns the `schema.Build` option matching
the typable's `In()`:

- `In() == body` ⇒ `schema.WithType(tpe, typable)` — full Schema
  build
- otherwise ⇒ `schema.WithSimpleSchema(tpe, typable, typable.In())` —
  SimpleSchema build with the `in` carried for keyword gating

Centralised in `buildOption` so every `buildFromFieldXxx` call site
picks the same shape uniformly. The schema builder enforces the
SimpleSchema vocabulary via `handlers.IsSimpleSchemaKeyword` (see
[../schema/README.md#simple-schema-mode](../schema/README.md#simple-schema-mode)
for the keyword surface).

### <a id="alias-handling"></a>Alias handling

The parameters builder shares the alias-handling contract with the
schema and responses builders — annotation gates first-class alias
identity at use sites; `TransparentAliases` overrides at use sites;
mode flags only shape the alias's own definition. The full rule
lives in
[schema/README.md §aliases](../schema/README.md#aliases); below
captures the parameters-specific reach contexts.

Two parameters-specific use-site handlers:

**Top-level alias annotated `swagger:parameters` (`buildAlias`).**
The alias is **transparent re: model creation** in all modes —
neither the alias nor any chain link of its backing struct
surfaces in `definitions`. The fields of the unaliased target
become the operation's parameters. The implementation just
forwards `tpe.Rhs()` to `buildFromType`; recursion handles chains
naturally. No mode-specific behaviour at this layer.

**Alias as a field type within a parameters struct
(`buildFieldAlias`).** Three branches:

- `TransparentAliases=true` — dissolve via the schema sub-builder.
- Non-body field (query / path / header / formData) — SimpleSchema
  target cannot carry `$ref`; always expand to the unaliased
  target via `types.Unalias`. Annotation has no effect at non-body
  sites.
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

- **`x-go-name` parity**: only emit when the JSON tag differs from
  the Go field name. Pre-Stream M this was sometimes emitted on
  aliases even when names matched.
- **`required:` on path parameters**: forced post-build, after the
  block walk. A user-authored `required: false` on a path parameter
  is overridden — OAS v2 requires path params to be required.
- **`swagger:strfmt` collapse**: when set, the field's resolved
  shape collapses to `string + format`, clearing `Ref` and `Items`.
  This is the single point where strfmt overrides the resolved
  build.
- **Pre-walk doc signal cache**: `scanFieldDocSignals` calls
  `p.ParseBlocks(afld.Doc)` which hits the `common.Builder` cache;
  the later `applyBlockToField` reads the same cache entry. One
  parse per comment group.
