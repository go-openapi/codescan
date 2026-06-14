// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1711

// listParams carries a query parameter with an explicit description, distinct
// from the Go field name. The reporter wanted the param description to be the
// comment text, not a field-name concatenation.
//
// swagger:parameters listThings
type listParams struct {
	// the maximum number of items to return
	//
	// in: query
	Limit int64 `json:"limit"`
}

// ListThings uses listThings.
//
// swagger:route GET /things things listThings
//
// List things.
//
//	Responses:
//	  200: description: ok
func ListThings() {}
