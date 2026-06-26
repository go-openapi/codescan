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
| `scan_context.go` | `ScanCtx` / `NewScanCtx` — loads Go packages via `golang.org/x/tools/go/packages` |
| `index.go` | `TypeIndex` — node classification (meta/route/operation/model/parameters/response) |
| `declaration.go` | `EntityDecl` — wraps a type/value declaration with its enclosing file/package |
| `classify/` | Classification predicates usable from both scanner and builders (e.g. `IsAllowedExtension`) |

### `internal/parsers/` — scanner classification + helpers

Post grammar-migration (P6.3), `parsers/` is intentionally scanner-only. The
old regex-based comment-block parsing engine is gone; what remains are
classification helpers used by the scanner and builders, plus subpackages
for the grammar parser and its satellite helpers.

**Root — scanner classification**

| File | Contents |
|------|----------|
| `matchers.go` | `ExtractAnnotation`, `ModelOverride`, `ResponseOverride`, `ParametersOverride` — the scanner-level annotation classifiers |
| `regexprs.go` | Regex definitions backing the matchers + `rxRoute` / `rxOperation` for the path-annotation parsers |
| `parsed_path_content.go` | `ParsedPathContent` + `ParseOperationPathAnnotation` / `ParseRoutePathAnnotation` |

**Subpackages**

| Package | Role |
|---------|------|
| `grammar/` | The grammar parser — `NewParser`, `Block`, `Property`, keyword tables |
| `yaml/` | YAML sub-parser used by grammar's typed-extensions surface and by operation / meta body unmarshal |

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
    `scan.renamed-definition`. See `internal/scanner/README.md#prune`.
  - `InputSpec` — overlay: merge discoveries on top of an existing spec.
  - `BuildTags`, `Include`/`Exclude`, `IncludeTags`/`ExcludeTags`, `ExcludeDeps` — scope control.
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
  - `SkipExtensions` — suppress `x-go-*` vendor extensions.
  - `OnDiagnostic` — callback sink for all scan-time observations (the only output
    channel; codescan never writes to stdout/stderr).
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
- The scanner works at the AST / `go/types` level — it never executes or compiles scanned code.
- Parsers never import builders; they write through the interfaces in `internal/ifaces`.
  When adding a new annotation, extend the relevant builder's `taggers.go` rather than reaching
  into parser internals.
- Test helpers live in `internal/scantest` and are never imported from production code (guarded by
  build-tag-free test files). Do not widen production API to satisfy a test — use `export_test.go`
  or an integration test instead.
- Golden-file comparisons go through `scantest.CompareOrDumpJSON`; regenerate with `UPDATE_GOLDEN=1 go test ./...`.
