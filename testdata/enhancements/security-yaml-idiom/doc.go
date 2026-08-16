// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package secyaml demonstrates that a Security requirements block written as
// idiomatic OpenAPI 2.0 YAML — a sequence of Security Requirement Objects —
// produces the spec-correct requirement set, identical to what a real YAML
// parse of the same text would yield. This is the form we recommend in the
// docs; the bespoke parser was retired in favour of a real YAML decode.
//
// The block below reads, in OpenAPI terms:
//
//	(oauth AND api_key) OR basic
//
// The first sequence item is one requirement object carrying TWO schemes (AND);
// the second item is an alternative (OR). Scopes use block style (the form the
// spec's own petstore uses); flow style (`[read, write]`) is equally accepted.
// The legacy newline-separated mapping (one bare `name:` per line, meaning OR)
// also still parses, but is NOT idiomatic YAML.
//
// Security:
//   - oauth:
//       - read
//       - write
//     api_key: []
//   - basic: []
//
// SecurityDefinitions:
//   oauth:
//     type: oauth2
//     flow: accessCode
//     authorizationUrl: https://example.com/oauth/authorize
//     tokenUrl: https://example.com/oauth/token
//     scopes:
//       read: read access
//       write: write access
//   api_key:
//     type: apiKey
//     name: X-API-Key
//     in: header
//   basic:
//     type: basic
//
// swagger:meta
package secyaml
