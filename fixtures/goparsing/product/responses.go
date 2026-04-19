// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package product

// swagger:response GetProductsResponse
type GetProductsResponse struct {
	// in:body
	Body map[string]Product `json:"body"`
}
