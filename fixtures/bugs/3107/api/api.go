// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package api carries the route whose response references the model in a
// sibling package, exercising cross-package type resolution for issue
// #3107.
package api

import "github.com/go-openapi/codescan/fixtures/bugs/3107/model"

// swagger:route GET /things things listThings
//
// responses:
//   200: thingResponse

// swagger:response thingResponse
type thingResponse struct {
	// in: body
	Body model.MyStruct
}
