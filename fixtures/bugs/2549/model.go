// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2549

import "github.com/go-openapi/codescan/fixtures/bugs/2549/sub"

// Property has an example on a field of an imported type (#2549).
//
// swagger:model
type Property struct {
	// Initial Market Value
	//
	// required: true
	// example: 210000
	MarketValue *sub.Decimal `json:"marketValue"`
}
