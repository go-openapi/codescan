// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2294 is the witness for go-swagger#2294: a global Security
// requirement that combines TWO schemes with AND logic — both must be satisfied.
// In OpenAPI 2.0 that is a single requirement object with two keys:
// `security: [{x_client_id: [], access_token: []}]`. Written as a YAML sequence
// item with two mapping keys, it must NOT split into two separate (OR)
// requirements.
//
// Security:
//   - x_client_id: []
//     access_token: []
//
// SecurityDefinitions:
//   x_client_id:
//     type: apiKey
//     name: x_client_id
//     in: header
//   access_token:
//     type: apiKey
//     name: Authorization
//     in: header
//
// swagger:meta
package bug2294
