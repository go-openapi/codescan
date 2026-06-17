// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2801

// swagger:parameters MyOperation
type MyParams WrappedRequest[Request]

// swagger:route POST /op things MyOperation
// responses:
//   200: description: ok
func op() {}
