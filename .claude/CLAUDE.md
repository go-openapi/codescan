# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go source code scanner that parses Go packages and produces [Swagger 2.0](https://swagger.io/specification/v2/)
(OpenAPI 2.0) specifications. It reads specially formatted comments (annotations) in Go source files
and extracts API metadata — routes, parameters, responses, schemas, and more — to build a complete
`spec.Swagger` document. Supports Go modules (go1.11+).

See [docs/MAINTAINERS.md](../docs/MAINTAINERS.md) for CI/CD, release process, and repo structure details.

### Package layout (single package)

| File | Contents |
|------|----------|
| `application.go` | `Options`, `Run` entry point, `appScanner` orchestration |
| `parser.go` | `sectionedParser` — comment block parsing engine (title, description, annotations) |
| `parser_helpers.go` | Helpers for tag/package filtering, value extraction |
| `meta.go` | Swagger info block parsing (title, version, license, contact, etc.) |
| `operations.go` | Operation (route handler) annotation parsing |
| `parameters.go` | Parameter annotation parsing |
| `responses.go` | Response annotation parsing |
| `routes.go` | Route/path discovery and matching |
| `route_params.go` | Route parameter extraction |
| `schema.go` | Go type → Swagger schema conversion |
| `enum.go` | Enum value extraction from Go constants |
| `spec.go` | Spec-level helpers |
| `regexprs.go` | Shared regular expressions for annotation parsing |
| `assertions.go` | Test assertion helpers |

### Key API

- `Run(*Options) (*spec.Swagger, error)` — the main entry point: scan Go packages and produce a Swagger spec
- `Options` — configuration: packages to scan, build tags, base swagger spec to overlay

### Dependencies

- `github.com/go-openapi/loads` — loading base Swagger specs
- `github.com/go-openapi/spec` — Swagger 2.0 spec types
- `github.com/go-openapi/swag` — string/JSON utilities
- `golang.org/x/tools` — Go package loading (`packages.Load`)
- `github.com/go-openapi/testify/v2` — test-only assertions

### Notable design decisions

- Uses `golang.org/x/tools/go/packages` for module-aware package loading.
- Comment annotations follow the go-swagger convention (`swagger:route`, `swagger:operation`,
  `swagger:parameters`, `swagger:response`, `swagger:model`, etc.).
- The scanner works on the AST level — it does not execute or compile the scanned code.
