// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2403 is the witness for go-swagger#2403: a global Security
// requirement in swagger:meta written as a YAML sequence. The `- auth0: []`
// list item must parse as a security requirement naming `auth0` with empty
// scopes — NOT as a property keyed by the literal string "- auth0".
//
// Security:
//   - auth0: []
//
// SecurityDefinitions:
//   auth0:
//     type: oauth2
//     flow: implicit
//     authorizationUrl: https://example.auth0.com/authorize
//
// swagger:meta
package bug2403
