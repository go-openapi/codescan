# Copilot Instructions — codescan

## Project Overview

Go source code scanner that parses Go packages and produces [Swagger 2.0](https://swagger.io/specification/v2/)
(OpenAPI 2.0) specifications. It reads specially formatted comments (annotations) in Go source files
and extracts API metadata — routes, parameters, responses, schemas, and more — to build a complete
`spec.Swagger` document. Supports Go modules (go1.11+).

Single module: `github.com/go-openapi/codescan`.

### Package layout (single package)

| File | Contents |
|------|----------|
| `application.go` | `Options`, `Run` entry point, `appScanner` orchestration |
| `parser.go` | `sectionedParser` — comment block parsing engine |
| `schema.go` | Go type → Swagger schema conversion |
| `operations.go` | Operation annotation parsing |
| `parameters.go` | Parameter annotation parsing |
| `responses.go` | Response annotation parsing |
| `routes.go` | Route/path discovery and matching |
| `meta.go` | Swagger info block parsing |

### Key API

- `Run(*Options) (*spec.Swagger, error)` — scan Go packages and produce a Swagger spec
- `Options` — configuration: packages to scan, build tags, base swagger spec

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

## Conventions

Coding conventions are found beneath `.github/copilot`

### Summary

- All `.go` files must have SPDX license headers (Apache-2.0).
- Commits require DCO sign-off (`git commit -s`).
- Linting: `golangci-lint run` — config in `.golangci.yml` (posture: `default: all` with explicit disables).
- Every `//nolint` directive **must** have an inline comment explaining why.
- Tests: `go test ./...`. CI runs on `{ubuntu, macos, windows} x {stable, oldstable}` with `-race`.
- Test framework: `github.com/go-openapi/testify/v2` (not `stretchr/testify`; `testifylint` does not work).

See `.github/copilot/` (symlinked to `.claude/rules/`) for detailed rules on Go conventions, linting, testing, and contributions.
