# Copilot Instructions — codescan

## Project Overview

Go source code scanner that parses Go packages and produces [Swagger 2.0](https://swagger.io/specification/v2/)
(OpenAPI 2.0) specifications. It reads specially formatted comments (annotations) in Go source files
and extracts API metadata — routes, parameters, responses, schemas, and more — to build a complete
`spec.Swagger` document. Supports Go modules (go1.11+).

Single module: `github.com/go-openapi/codescan`. Public API is a thin facade at the root; the
implementation lives under `internal/`.

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
| `internal/parsers` | Comment-block parsing engine: sectioned parser, meta, responses, route params, validations, extensions, YAML body parser, enum extraction, security definitions |
| `internal/builders/spec` | Top-level `Builder` orchestrating the final `*spec.Swagger` |
| `internal/builders/schema` | Go type → Swagger schema conversion (largest builder) |
| `internal/builders/{operations,parameters,responses,routes,items}` | Per-concern builders |
| `internal/builders/resolvers` | `SwaggerSchemaForType` and shared assertion helpers |
| `internal/ifaces` | `SwaggerTypable`, `ValidationBuilder`, `ValueParser`, `Objecter` — decouples parsers from builders |
| `internal/logger` | Debug logging (gated on `Options.Debug`) |
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
  `SetXNullableForPointers`, `SkipExtensions`, `Debug`).

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
  annotation, extend the relevant builder's `taggers.go` rather than reaching into parser internals.
- Test helpers live in `internal/scantest`; do not widen production API to satisfy a test.

See `.github/copilot/` for detailed rules on Go conventions, linting, testing, and contributions.
