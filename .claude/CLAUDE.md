# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go source code scanner that parses Go packages and produces [Swagger 2.0](https://swagger.io/specification/v2/)
(OpenAPI 2.0) specifications. It reads specially formatted comments (annotations) in Go source files
and extracts API metadata — routes, parameters, responses, schemas, and more — to build a complete
`spec.Swagger` document. Supports Go modules (go1.11+).

See [Maintainers documentation][maintainers-doc-site] for CI/CD, release process, and repo structure details.

[maintainers-doc-site]: https://go-openapi.github.io/doc-site/maintainers/index.html

## Package layout

Single Go module `github.com/go-openapi/codescan`. Public API lives at the root; implementation is
split under `internal/` into three layers: **scanner** (package/AST ingestion), **parsers** (comment
block parsing), and **builders** (emitting Swagger objects). A thin `ifaces` package glues parsers
to builders without direct coupling.

### Root (public API — keep surface minimal)

| File | Contents |
|------|----------|
| `api.go` | `Run(*Options) (*spec.Swagger, error)` entry point; re-exports `Options = scanner.Options` |
| `diagnostics.go` | Re-exports the diagnostic surface for `Options.OnDiagnostic` callers: `Diagnostic`/`Severity`/`Code` aliases + `Severity*` constants |
| `doc.go` | Package godoc |
| `errors.go` | `ErrCodeScan` sentinel error |

### `internal/scanner/` — package loading & entity discovery

| File | Contents |
|------|----------|
| `options.go` | `Options` struct: packages, work dir, build tags, include/exclude, feature flags |
| `scan_context.go` | `ScanCtx` / `NewScanCtx`; `loadPackages` picks a loader from `Options.FS` (see `README.md#loader`) |
| `index.go` | `TypeIndex` — node classification (meta/route/operation/model/parameters/response) |
| `declaration.go` | `EntityDecl` — wraps a type/value declaration with its enclosing file/package |
| `written_rhs.go` | `WrittenRHS` — the type a declaration was *written* over (`Stamp` in `type StampResp Stamp`), which peeling to the underlying would discard |
| `enum_value.go` | `enumBasicLitValue` — converts a `const Foo Kind = "bar"` RHS into its runtime value (enum discovery) |
| `provenance.go` | `Provenance` — ties a spec JSON pointer to the source position of the Go construct that produced it; emitted via `Options.OnProvenance` (cross-ref linker, source side) |
| `classify/` | Classification predicates usable from both scanner and builders (e.g. `IsAllowedExtension`) |

### `internal/cliopts/` — the command-line option surface

Every knob on `codescan.Options` as a flag, declared once and shared by the commands (`cmd/genspec`,
`cmd/genspec-wasi`) so a flag means the same thing whichever one you reach for. Entries are keyed by
the field's *setter*, never by a name derived from the field, and a guard in the package tests fails
when a value-typed option lands with no flag (or excuse). Flag names are the kebab-case of the field
without exception. `-loader` (`go|own|auto`) is the tri-state that discharges `ToolchainFreeLoader`.
Options that are not values — `FS`, `ExportData`, `InputSpec`, the callbacks — are the command's
business, not the table's.

### `internal/packages/` — toolchain-free package loader (experimental)

Owns **both** ways of resolving a package graph, behind one `Loader`: delegate to
`golang.org/x/tools/go/packages` (default), or do the same job in pure Go with no `go list` and no
`exec`. The scanner states a preference and the loader reconciles it — see
`internal/scanner/README.md#loader`. Keeping the switch here is what makes options like `WithTarget`
mean the same thing under both. Types (`Config`, `Package`, `Error`, `LoadMode`) are aliases of the
upstream ones, so the two are interchangeable at the call site.

Split three ways, because the halves are inherited from different places and age differently: the
loader is a simplified `go/packages` (a shape we own), while `list/` is `cmd/go` (behaviour we must
reproduce exactly). Quarantining the quirks is what makes them checkable against upstream.

| File | Contents |
|------|----------|
| `strategy.go` | `Strategy` (`StrategyGoPackages` / `StrategyToolchainFree`), `WithStrategy`, the `WithFS` override, shared `loadMode` |
| `golist.go` / `golist_wasm.go` | the go/packages strategy + pinned-env propagation; refused on `wasm`, which has no process model |
| `loader.go` | `Loader` / `NewLoader` / `Load` (dispatch); `loadFromSource`, `buildContext` (GOOS/GOARCH/cgo tiering), parse + type-check |
| `goenv.go` | `GoEnv` — the pinned go environment (GOOS/GOARCH/GOFLAGS/GOWORK/GOEXPERIMENT), applied to both strategies |
| `synthesize.go` | fabricates a package from the names selected through it when its source cannot be read (incl. the cgo `"C"` pseudo-package) |
| `exportdata.go` | serves dependencies from pre-computed export data (`Options.ExportData`) |
| `options.go` / `aliases.go` | functional options; type aliases of the upstream vocabulary |

#### `internal/packages/list/` — `go list` semantics

| File | Contents |
|------|----------|
| `resolve.go` | `Resolver`: patterns → directories, imports → directories; module-bounded `...` walk, nearest-module import paths, vendor mode, module cache, `replace` |
| `workspace.go` | `go.work` discovery + `use`/`replace`, so a sibling module resolves to its working copy |
| `pattern.go` | where a `...` walk starts, and which walked directories a pattern matches |
| `pkgpattern.go` | the wildcard matcher, **copied verbatim** from the Go distribution (BSD-3-Clause; see `NOTICE`) |

#### `internal/packages/vfs/` — the filesystem seam

`vfs.FS` — the one place every read goes through (build-constraint matching, directory walking, source
reading), so a virtual tree is honoured everywhere. Its own package because both halves need it and
neither owns it.

### `internal/parsers/` — scanner classification + helpers

Post grammar-migration (P6.3), `parsers/` is intentionally scanner-only. The
old regex-based comment-block parsing engine is gone; what remains are
classification helpers used by the scanner and builders, plus subpackages
for the grammar parser and its satellite helpers.

**Root — scanner classification**

| File | Contents |
|------|----------|
| `matchers.go` | `ExtractAnnotation`, `ModelOverride`, `ResponseOverride`, `ParametersOverride` — the scanner-level annotation classifiers |
| `annotation_line.go` | The string-scanning primitives behind the matchers — comment-prefix stripping, keyword and argument extraction. No regexes |
| `regexprs.go` | `rxRoute` / `rxOperation` (+ their heads) for the path-annotation parsers — the only regexes left in the codebase |
| `parsed_path_content.go` | `ParsedPathContent` + `ParseOperationPathAnnotation` / `ParseRoutePathAnnotation` |

**Subpackages**

| Package | Role |
|---------|------|
| `grammar/` | The grammar parser — `NewParser`, `Block`, `Property`, keyword tables |
| `yaml/` | YAML sub-parser used by grammar's typed-extensions surface and by operation / meta body unmarshal |
| `routebody/` | Sub-parser for the multi-line body grammar (`Parameters:` / `Responses:`) nested under `swagger:route` / `swagger:operation`; emits typed `ParamDecl` / `ResponseDecl` + a `grammar.Block` dispatched through the shared `handlers` seam |
| `security/` | Inline `Security:` block-body parser (genuine YAML) shared by `swagger:meta` / `route` / `operation`; normalises to the OpenAPI 2.0 `[]map[string][]string` shape |

### `internal/builders/` — Swagger object construction

Each sub-package owns one concern; `walker.go` carries the per-block grammar dispatch.

| Package | Contents |
|---------|----------|
| `spec` | `Builder` — top-level orchestrator producing the final `*spec.Swagger` |
| `schema` | Go type → Swagger schema conversion (the largest builder; dispatch in `schema.go`) |
| `operations` | Operation (route handler) annotation parsing |
| `parameters` | Parameter annotation parsing |
| `responses` | Response annotation parsing |
| `routes` | Route/path discovery + body parsers (`body_params.go`, `body_responses.go`) |
| `common` | `*common.Builder` embedded by every per-decl builder; `SchemesList` + `SecurityRequirements` shared by routes/spec |
| `handlers` | Walker callback factories shared across schema/parameters/responses (`Number`, `Integer`, `UniqueBool`, `PatternString`, …) |
| `resolvers` | `SwaggerSchemaForType`, identity/assertion helpers, items-chain ifaces adapters (`ItemsTypable` / `ItemsValidations`) shared by builders |
| `validations` | Type-aware coercion / shape-check primitives (`CoerceEnum`, `ParseDefault`, `IsLegalForType`) |
| `godoclink` | godoc-syntax cleanup + recomposition backing `CleanGoDoc`: drop `[text]: url` ref-def lines, humanize doc-link spans (`[CustName]` → "cust name"), and via two-phase markers recompose a resolved doc-link to the schema's final exposed name |

### `internal/ifaces/` — cross-package interfaces

`SwaggerTypable`, `ValidationBuilder`, `OperationValidationBuilder`, `ValueParser`, `Objecter` —
the glue that lets `parsers` write into any builder's target without importing concrete builders.

### `internal/scantest/` — test utilities (do **not** import from production code)

| File | Contents |
|------|----------|
| `load.go` | `FixturesDir`, package-loading helpers |
| `golden.go` | `CompareOrDumpJSON` — golden-file comparison honoring `UPDATE_GOLDEN=1` |
| `property.go` | Assertion helpers for property-shape checks |
| `classification/` | Reusable assertions over the classification fixture |
| `mocks/` | Minimal mock implementations of `ifaces` interfaces |

### `internal/integration/` — black-box integration tests

Scans fixture trees and compares against `fixtures/integration/golden/*.json`. Tests for enhancements,
malformed input, the petstore, aliased schemas, go123-specific forms, and cross-feature coverage.

### `fixtures/`

- `fixtures/goparsing/...` — historic corpus: classification, petstore, go118/go119/go123 variants, invalid inputs.
- `fixtures/enhancements/...` — one sub-directory per isolated branch-coverage scenario (e.g. `swagger-type-array`,
  `alias-expand`, `allof-edges`, `named-basic`, `interface-methods`).
- `fixtures/integration/golden/*.json` — captured Swagger output for golden comparisons.
- `fixtures/bugs/...` — minimised repros for specific upstream bug IDs.

## Key API

- `codescan.Run(*Options) (*spec.Swagger, error)` — the main entry point.
- `codescan.Options` — configuration. Notable fields beyond `Packages`/`WorkDir`:
  - `ScanModels` — also emit definitions for `swagger:model` types.
  - `PruneUnusedModels` — with `ScanModels`, prune discovered definitions not
    transitively referenced from any path/response/parameter/overlay root.
    Runs before name reduction (so an unused model can't force a spurious
    collision rename on a used one); `InputSpec` definitions are pinned. Each
    drop raises a `scan.pruned-unused` Hint; collision renames raise
    `scan.renamed-definition`. A reachable discriminated base keeps its whole
    subtype family (see the subtype-discovery note under "Notable design
    decisions"). See `internal/scanner/README.md#prune`.
  - `InputSpec` — overlay: merge discoveries on top of an existing spec.
  - `BuildTags`, `Include`/`Exclude`, `IncludeTags`/`ExcludeTags`, `ExcludeDeps` — scope control.
  - `GOOS`/`GOARCH`/`GOFLAGS`/`GOWORK`/`GOEXPERIMENT` — the go environment that
    decides *what* is built. Each changes the emitted spec just as `BuildTags`
    does, and each is an option rather than inherited state so a scan is
    reproducible. Empty means "whatever the environment says". Honoured by both
    loaders. `GOWORK` matters most: inside a workspace a sibling module resolves
    to its working copy, and missing it synthesizes the sibling empty rather than
    failing. See `internal/scanner/README.md#loader`.
  - `SkipCompiledDependencies` — opt out (default false) of taking dependency
    types from the compiler's export data under the go/packages loader. Unset is
    the default since v0.36.4 and costs no meaning: a dependency's annotations are
    read back after the load, and a declaration the spec needs is fetched at the
    lookup that wants it. Set it for cost alone — `go list -export` compiles the
    closure, so a cold cache is an order of magnitude slower and writes ~229 MB.
    Code that fails to build needs no opt-out: that load is retried from source,
    which is what keeps go-swagger#2874 fixed. See
    `internal/scanner/README.md#compiled-dependencies`.
  - Toolchain-free loading (experimental; see `internal/scanner/README.md#loader`):
    - `ToolchainFreeLoader` — scan through `internal/packages` instead of
      `golang.org/x/tools/go/packages`. Same job, same output; it just needs no
      installed toolchain and no `exec`, since it never runs `go list`. False
      (default) keeps the historic behaviour.
    - `FS` — read source through an `io/fs` filesystem instead of the real one.
      Implies `ToolchainFreeLoader`: `go list` runs against the real filesystem,
      so it could not honour `FS` even if asked. **`FS` is the whole world the
      scan can read**, not just the module: dependencies and GOROOT come through
      it too, absolute paths map by dropping the leading separator (so a tree
      serving GOROOT must mirror its absolute layout), and anything unreachable
      is synthesized — a valid, quietly thinner spec plus `scan.degraded-load` /
      `scan.synthesized-import`. See `internal/scanner/README.md#virtual-filesystem`.
    - `StubStdlib` — synthesize the stdlib from the names selected through it
      rather than reading GOROOT. Trades fidelity for reach: a synthesized type
      has no fields and no method set, so `json.RawMessage` stops rendering as a
      byte array and nothing is seen to implement `encoding.TextMarshaler`. Each
      fabricated import raises `scan.synthesized-import`.
    - `ExportData` — serve dependencies from pre-computed export data. Costs no
      fidelity, but is valid only for the toolchain that produced it; uncovered
      packages fall back to source, then to synthesis.
  - `RefAliases`, `TransparentAliases`, `DescWithRef` — alias handling knobs
    (`DescWithRef` is deprecated; see `EmitRefSiblings`).
  - `$ref`-sibling rendering (see `internal/builders/schema/README.md#ref-override`):
    - `EmitRefSiblings` — emit a `$ref`'d field's description & extensions as direct
      siblings (`{$ref, description, x-*}`) instead of an `allOf` wrap; validations/
      externalDocs still force a compound.
    - `SkipAllOfCompounding` — never emit an `allOf` compound; validations/externalDocs
      dropped (description/extensions too, unless `EmitRefSiblings` keeps them as
      siblings), each with a diagnostic. For consumers (e.g. go-swagger) wanting bare refs.
  - `DefaultAllOfForEmbeds` — opt-in (default false): render a plain
    (non-`swagger:allOf`) struct embed as allOf composition instead of inlining
    its properties — a `$ref` allOf member for a model embed, an inline member
    otherwise, with the embedding struct's own fields in a sibling member.
    Json-named embeds (go-swagger#2038) and interface embeds are unaffected;
    `swagger:allOf` already wins. See `internal/builders/schema/README.md#allof`.
  - `SetXNullableForPointers` — emit `x-nullable: true` on pointer fields.
  - `NameFromTags` — ordered list of struct-tag types a field's emitted name is
    derived from (schema properties, parameters, response headers). First listed
    tag that supplies a usable name wins. nil/unset ⇒ `["json"]` (historic);
    explicit empty slice ⇒ Go field name. Only the name; encoding/json directives
    (`-`, `,omitempty`, `,string`) always come from the `json` tag. e.g.
    `["form","json"]` for gin (go-swagger#2912/#1391).
  - `SkipJSONifyInterfaceMethods` — opt out (default false) of the auto-jsonify
    mangler on interface-method property names (`ID`→`id`, `CreatedAt`→`createdAt`).
    When true the Go method name is emitted verbatim; `swagger:name` still wins
    verbatim regardless. Does not affect struct fields. See
    `internal/builders/schema/README.md#interface-naming`.
  - `SingleLineCommentAsDescription` — opt-in (default false): a single-line doc
    comment always becomes `description`, never `title`/`summary`, regardless of
    trailing punctuation. Multi-line comments keep the title/description split.
    go-swagger#2626.
  - `AfterDeclComments` — opt-in (default false): allow swagger annotations INSIDE
    a declaration (struct-body leading comment) or INLINED as a trailing comment,
    keeping the godoc above the decl clean. Scanner-only; same annotation grammar.
    v0.36 scope: type decls (struct/alias); fields & const enums are follow-ups.
  - `CleanGoDoc` — opt-in (default false): rewrite godoc-only syntax (doc-link
    brackets, `[text]: url` ref-def lines) carried into a title/description, and
    recompose a resolved doc-link to the schema's final exposed name. Touches only
    godoc-derived prose, never `swagger:title`/`swagger:description` overrides. See
    the `godoclink` package.
  - `SkipEnumDescriptions` — opt-in (default false): keep the per-enum-value
    const-name mapping on the `x-go-enum-desc` extension only, instead of also
    appending it to the authored description. go-swagger#2922.
  - `EmitXGoType` — opt-in (default false): stamp `x-go-type` (`<pkg path>.<type>`)
    on every emitted definition, alongside `x-go-name`/`x-go-package`. Under the
    `SkipExtensions` umbrella. go-swagger#2924.
  - `NameConcatBudget` — readability cutoff (default 0.65) for the name-identity
    reduce stage when deconflicting collisions by concatenating package segments
    (`b.Test`/`c.Test` → `BTest`/`CTest`). A group whose best concat scores above
    the budget is a candidate for the hierarchical fallback.
  - `EmitHierarchicalNames` — opt-in (default false): emit over-budget collision
    groups as nested container definitions (`#/definitions/<pkg>/<Name>`) instead
    of long flat concats. Default off because nested pointers only resolve under
    `ExpandSpec` and confuse definition-enumerating consumers (e.g. go-swagger codegen).
  - `SkipExtensions` — suppress `x-go-*` vendor extensions.
  - `OnDiagnostic` — callback sink for all scan-time observations (the only output
    channel; codescan never writes to stdout/stderr).
  - `OnProvenance` — callback invoked once per anchor node (type decls, fields,
    values, route/meta blocks) with its JSON pointer + source position; powers the
    cross-ref linker (LSP/TUI). Experimental. See `internal/scanner/provenance.go`.
  - `Debug` — deprecated no-op (the legacy stderr debug logger was retired; wire
    `OnDiagnostic` instead).

## Dependencies

- `github.com/go-openapi/loads` — loading base Swagger specs
- `github.com/go-openapi/spec` — Swagger 2.0 spec types
- `github.com/go-openapi/swag` — string/JSON utilities
- `golang.org/x/tools` — Go package loading (`packages.Load`)
- `github.com/go-openapi/testify/v2` — test-only assertions (zero-dep fork of `stretchr/testify`)

## Notable design decisions

- Uses `golang.org/x/tools/go/packages` for module-aware package loading.
- Comment annotations follow the go-swagger convention (`swagger:route`, `swagger:operation`,
  `swagger:parameters`, `swagger:response`, `swagger:model`, etc.).
- `swagger:description |` (YAML literal block-scalar marker) captures a verbatim
  markdown body — blank lines, indentation, table pipes preserved — until the next
  line-leading annotation or EOF; reframes go-swagger#3211. Plain `swagger:description`
  stays blank-terminated. See `internal/parsers/grammar/README.md#literal-description`.
- Discriminator subtypes are discovered **backwards**: a subtype `$ref`s its base and
  nothing `$ref`s the subtype, so a reachable definition carrying `discriminator` pulls
  in the `swagger:model` types that embed it under `swagger:allOf` — no `ScanModels`
  needed, and a discriminated family is never pruned down to its base. Always on (an
  incomplete polymorphic family is unusable, so this is a fix, not a knob); each pull
  raises a `scan.discovered-subtype` Hint. go-swagger#1913. See
  `internal/scanner/README.md#subtypes`.
- `swagger:omit <name>[,<name>…]` is the author's escape hatch for an embed that promotes more
  than the API should carry (go-swagger#1992): a **pre-filter** on the promotion walk, so the
  listed Go fields are never written and the annotation reads identically whether the embed is
  inlined or composed into an `allOf` member. Targets resolve against the *type*
  (`types.LookupFieldOrMethod`), never against the emitted names. Placed on the embed (plain field
  names) or on the declaration (dotted embed paths); embeds only. Unresolved / behind-`$ref`
  targets raise Hints. See `internal/builders/schema/README.md#omit`.
- A `swagger:enum` schema takes its `type`/`format` from the **declared** Go type (`int8` →
  `integer/int8`, `float32` → `number/float`), never from the parsed const values, and each member is
  normalised to that type — typing from the first value let the const block's declaration order
  decide the schema type. Member *values* come from the type-checker
  (`TypesInfo.Defs[name].(*types.Const).Val()`), never from the literal syntax, and membership is
  decided per name from the constant's own type — which is what makes `iota` blocks (where only the
  first spec carries a type and a value) visible at all. So `iota`, expressions (`1 << 3`),
  references to earlier members, rune literals (`'a'` → 97), `true`/`false` (identifiers, not
  literals), raw/escaped strings, every integer base and above-`MaxInt64` members all resolve
  (go-swagger#3412). A literal reader survives only as the degraded-load fallback. See
  `internal/scanner/README.md#enum-values`, `internal/builders/schema/README.md#enum-typing` and
  `internal/builders/validations/README.md#enum-const-values`.
- The scanner works at the AST / `go/types` level — it never executes or compiles scanned code.
- Parsers never import builders; they write through the interfaces in `internal/ifaces`.
  When adding a new annotation, extend the relevant builder's `taggers.go` rather than reaching
  into parser internals.
- Test helpers live in `internal/scantest` and are never imported from production code (guarded by
  build-tag-free test files). Do not widen production API to satisfy a test — use `export_test.go`
  or an integration test instead.
- Golden-file comparisons go through `scantest.CompareOrDumpJSON`; regenerate with `UPDATE_GOLDEN=1 go test ./...`.
