// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package api holds a parameters struct in a DIFFERENT package from the route
// that references it (go-swagger#1742).
package api

// ListParams are the query parameters for the list operation, declared in a
// separate package from the route annotation.
//
// swagger:parameters listThings
type ListParams struct {
	// Limit caps the number of results.
	// in: query
	Limit int64 `json:"limit"`
}
