// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1713

// Test probes response examples keyed by mime type (examples: {application/json:
// {...}}), as the reporter wanted to label example responses.
//
// swagger:operation GET /test test testOp
//
// Test.
//
// ---
// responses:
//   '200':
//     description: Success
//     examples:
//       application/json:
//         test: blah
//         hello: world
func Test() {}
