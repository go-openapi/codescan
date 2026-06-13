// SPDX-License-Identifier: Apache-2.0

// Package validations holds the annotated declarations used by the
// "Validations" tutorial. validations_test.go scans this package and writes the
// per-feature golden fragments the tutorial renders.
package validations

// snippet:field

// Product is a model whose fields carry the full JSON-schema validation surface.
//
// swagger:model
type Product struct {
	// SKU is the stock code.
	//
	// required: true
	// pattern: ^[A-Z]{3}-[0-9]{4}$
	SKU string `json:"sku"`

	// Price is the price in cents.
	//
	// minimum: 1
	// maximum: 1000000
	// multipleOf: 1
	Price int64 `json:"price"`

	// Name is the display name.
	//
	// min length: 1
	// max length: 120
	Name string `json:"name"`

	// Grade is a quality band.
	//
	// enum: A,B,C
	Grade string `json:"grade"`

	// Tags label the product.
	//
	// min items: 1
	// max items: 10
	// unique: true
	Tags []string `json:"tags"`
}

// endsnippet:field

// swagger:route GET /products/search search searchProducts
//
// responses:
//
//	200: productList
//	429: rateLimited

// snippet:param

// SearchParams is the simple-schema parameter set for searchProducts. Query
// parameters accept the reduced OAS 2.0 validation surface.
//
// swagger:parameters searchProducts
type SearchParams struct {
	// Q is the search text.
	//
	// in: query
	// min length: 3
	// max length: 50
	Q string `json:"q"`

	// Limit caps the number of results.
	//
	// in: query
	// minimum: 1
	// maximum: 100
	Limit int32 `json:"limit"`

	// Sort lists the sort fields.
	//
	// in: query
	// collection format: csv
	// unique: true
	Sort []string `json:"sort"`
}

// endsnippet:param

// snippet:header

// RateLimited is a response carrying a validated header (a simple schema).
//
// swagger:response rateLimited
type RateLimited struct {
	// XRateRemaining is the remaining request budget.
	//
	// minimum: 0
	XRateRemaining int32 `json:"X-Rate-Remaining"` //nolint:tagliatelle // this is the canonicalized header name
}

// endsnippet:header

// ProductList is the 200 response body for searchProducts.
//
// swagger:response productList
type ProductList struct {
	// in: body
	Body []Product
}
