// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2899

// swagger:route POST /login auth doLogin
//
// Logs in.
//
// parameters:
//   - name: twoFactorCode
//     description: Six digit SMS code
//     in: body
//     type: string
//     example: "123456"
//
// responses:
//
//	200: description: ok
func doLogin() {}
