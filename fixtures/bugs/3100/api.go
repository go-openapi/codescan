// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3100 reproduces go-swagger issue #3100: a parameter declared
// with `in: formData` inside an inline `swagger:route` annotation lost its
// `in` field in the generated spec (the scanner only recognised `form`),
// so the parameter rendered with name/type/description but no `in`.
//
// The expected behaviour — locked by the golden — is that `in: formData`
// round-trips verbatim, for the `+name:` flush form the reporter used,
// alongside parameters in other locations (here `query`).
package bug3100

// swagger:route POST /v1/example/route operationName
// Very important operation
// consumes:
//   - application/x-www-form-urlencoded
// parameters:
//   +name: encryption_public_key
//     description: Public key of the referee client.
//     in: formData
//     type: string
//     required: true
//   +name: signature
//     description: Detached signature of the payload.
//     in: formData
//     type: string
//   +name: verbose
//     description: Echo the request back in the response.
//     in: query
//     type: boolean
// responses:
//   200: description: ok
