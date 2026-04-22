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
| `matchers.go` | Classification helpers: `HasAnnotation`, `ExtractAnnotation`, `ModelOverride`, `StrfmtName`, `ParamLocation`, etc. |
| `regexprs.go` | Regex definitions for the `swagger:<name>` annotations + `in:` / `required:` classification |
| `parsed_path_content.go` | `ParsedPathContent` + `ParseOperationPathAnnotation` / `ParseRoutePathAnnotation` |

**Subpackages**

| Package | Role |
|---------|------|
| `grammar/` | The new grammar-based comment parser — `NewParser`, `Block`, `Property`, keyword tables |
| `grammar/gen/`, `grammar/grammar_test/` | Generator + external grammar tests |
| `helpers/` | Bridge-consumed helpers: `JoinDropLast`, `CollectScannerTitleDescription`, `RemoveIndent`, `ParseEnum`, `ParseValueFromSchema`, `YAMLListBody`, `SecurityRequirements`, `SchemesList`, enum-desc extension handling |
| `yaml/` | Grammar's companion YAML sub-parser — reads `---`-fenced bodies into generic values |
| `enum/` | Experimental enum body sub-parser (not yet activated — see `.claude/plans/workshops/w2-enum.md`) |
| `routebody/` | **The last v1 regex-era body parsers.** `SetOpParams` / `SetOpResponses` / `SetOpExtensions` consume the indented `parameters:` / `responses:` / `extensions:` blocks inside `swagger:route` docs. Consumed exclusively by `internal/builders/routes/bridge.go` — deleteable as a unit once routes grows a grammar-native body pipeline |

### `internal/builders/` — Swagger object construction

Each sub-package owns one concern and a `taggers.go` file wiring parsers to its targets.

| Package | Contents |
|---------|----------|
| `spec` | `Builder` — top-level orchestrator producing the final `*spec.Swagger` |
| `schema` | Go type → Swagger schema conversion (the largest builder; dispatch in `schema.go`) |
| `operations` | Operation (route handler) annotation parsing |
| `parameters` | Parameter annotation parsing |
| `responses` | Response annotation parsing |
| `routes` | Route/path discovery and matching |
| `items` | Array-item targets (typable + validations, no own annotations) |
| `resolvers` | `SwaggerSchemaForType`, identity/assertion helpers shared by builders |

### `internal/ifaces/` — cross-package interfaces

`SwaggerTypable`, `ValidationBuilder`, `OperationValidationBuilder`, `ValueParser`, `Objecter` —
the glue that lets `parsers` write into any builder's target without importing concrete builders.

### `internal/logger/` — debug logging

`debug.go` — gated on `Options.Debug`.

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
  - `InputSpec` — overlay: merge discoveries on top of an existing spec.
  - `BuildTags`, `Include`/`Exclude`, `IncludeTags`/`ExcludeTags`, `ExcludeDeps` — scope control.
  - `RefAliases`, `TransparentAliases`, `DescWithRef` — alias handling knobs.
  - `SetXNullableForPointers` — emit `x-nullable: true` on pointer fields.
  - `SkipExtensions` — suppress `x-go-*` vendor extensions.
  - `Debug` — verbose logging via `internal/logger`.

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
