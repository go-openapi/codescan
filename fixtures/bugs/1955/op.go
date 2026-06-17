// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1955

// swagger:operation GET /cross crossOp
//
// Cross-package operation.
//
// ---
// responses:
//   '200':
//     description: ok
func Cross() {}
