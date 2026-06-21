# Copilot Instructions — codescan

## Project Overview

Go source code scanner that parses Go packages and produces [Swagger 2.0](https://swagger.io/specification/v2/)
(OpenAPI 2.0) specifications. It reads specially formatted comments (annotations) in Go source files
and extracts API metadata — routes, parameters, responses, schemas, and more — to build a complete
`spec.Swagger` document. Supports Go modules (go1.11+).

Single module: `github.com/go-openapi/codescan`. Public API is a thin facade at the root; the
implementation lives under `internal/` and is split into three layers: **scanner** (package /
AST ingestion), **parsers** (annotation grammar + sub-language parsers), and **builders**
(emitting Swagger objects). A thin `ifaces` package glues parsers to builders without direct
coupling.

### Layout

Root (`codescan` package — public surface):

| File | Contents |
|------|----------|
| `api.go` | `Run(*Options) (*spec.Swagger, error)` entry point; `Options = scanner.Options` |
| `doc.go` | Package godoc |
| `errors.go` | `ErrCodeScan` sentinel error |

Internal tree:

| Package | Role |
|---------|------|
| `internal/scanner` | Package loading via `golang.org/x/tools/go/packages`, entity discovery, `ScanCtx`, `TypeIndex`, `Options` |
| `internal/scanner/classify` | Classification predicates shared with builders (e.g. `IsAllowedExtension`) |
| `internal/parsers` | Scanner-level annotation classifiers (`ExtractAnnotation`, `ModelOverride`, …) and route/operation path-annotation parsing |
| `internal/parsers/grammar` | Annotation grammar parser: preprocessor, lexer, recursive-descent parser, typed `Block` family, `Walker` visitor, diagnostics |
| `internal/parsers/yaml` | YAML sub-parser used by the grammar's typed-extensions surface and by operation / meta body unmarshal |
| `internal/parsers/routebody` | Sub-parser for the multi-line body grammar nested under `swagger:route` |
| `internal/parsers/security` | Inline security-requirement line parser shared by routes and meta |
| `internal/builders/spec` | Top-level orchestrator producing the final `*spec.Swagger` |
| `internal/builders/schema` | Go type → Swagger schema conversion (largest builder); supports full Schema and SimpleSchema modes |
| `internal/builders/{operations,parameters,responses,routes}` | Per-annotation builders |
| `internal/builders/common` | `*common.Builder` embedded by every per-decl builder; parsed-block cache, post-decl queue, diagnostic accumulator, `MakeRef` |
| `internal/builders/handlers` | Reusable Walker callback factories (`Number`, `Integer`, `UniqueBool`, `Extension`, parameter/schema dispatch) |
| `internal/builders/validations` | Type-aware coercion (`CoerceEnum`, `ParseDefault`) + shape checks (`IsLegalForType`) |
| `internal/builders/resolvers` | `SwaggerSchemaForType`, identity / assertion helpers, items-chain ifaces adapters |
| `internal/ifaces` | `SwaggerTypable`, `ValidationBuilder`, `OperationValidationBuilder`, `ValueParser`, `Objecter` — decouples parsers from builders |
| `internal/scantest` | Test utilities: golden compare, fixture loading, mocks, classification helpers |
| `internal/integration` | Black-box integration tests against `fixtures/integration/golden/*.json` |

Fixtures:

- `fixtures/goparsing/...` — classification / petstore / go118-119-123 variants / invalid inputs.
- `fixtures/enhancements/...` — one sub-directory per focused branch-coverage scenario.
- `fixtures/integration/golden/*.json` — captured Swagger output for golden comparisons.
- `fixtures/bugs/...` — minimised repros for specific upstream bug IDs.

### Key API

- `Run(*Options) (*spec.Swagger, error)` — scan Go packages and produce a Swagger spec.
- `Options` — packages, work dir, build tags, include/exclude filters, `ScanModels`, `InputSpec`
  overlay, plus feature flags (`RefAliases`, `TransparentAliases`, `DescWithRef`,
  `SetXNullableForPointers`, `SkipExtensions`, `OnDiagnostic`, `Debug`).

### Dependencies

- `github.com/go-openapi/loads` — loading base Swagger specs
- `github.com/go-openapi/spec` — Swagger 2.0 spec types
- `github.com/go-openapi/swag` — string/JSON utilities
- `golang.org/x/tools` — Go package loading
- `github.com/go-openapi/testify/v2` — test-only assertions (zero-dep fork of `stretchr/testify`)

## Building & testing

```sh
go test ./...
```

Regenerate golden files after intentional output changes:

```sh
UPDATE_GOLDEN=1 go test ./...
```

## Conventions

Coding conventions are found beneath `.github/copilot` (symlinked to `.claude/rules/`).

### Summary

- All `.go` files must have SPDX license headers (Apache-2.0).
- Commits require DCO sign-off (`git commit -s`).
- Linting: `golangci-lint run` — config in `.golangci.yml` (posture: `default: all` with explicit disables).
- Every `//nolint` directive **must** have an inline comment explaining why.
- Tests: `go test ./...`. CI runs on `{ubuntu, macos, windows} x {stable, oldstable}` with `-race`.
- Test framework: `github.com/go-openapi/testify/v2` (not `stretchr/testify`; `testifylint` does not work).
- Parsers never import builders — write into the interfaces in `internal/ifaces`. When adding a new
  annotation keyword, extend the grammar keyword table and wire a matching Walker handler in the
  relevant builder rather than reaching across the parser/builder boundary.
- Test helpers live in `internal/scantest`; do not widen production API to satisfy a test.

See `.github/copilot/` for detailed rules on Go conventions, linting, testing, and contributions.
