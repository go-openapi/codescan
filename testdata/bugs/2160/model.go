// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2160

// FooBarStruct is an element.
//
// swagger:model
type FooBarStruct struct {
	Amount int `json:"amount"`
	Value  int `json:"value"`
}

// ResponseBody has an array-of-structs example (#2160).
//
// swagger:model
type ResponseBody struct {
	// The foo value
	// example: 123456
	Foo int64 `json:"foo"`
	// FooBars
	// example: [{"amount":1,"value":4900},{"amount":15,"value":4500}]
	FooBars []FooBarStruct `json:"fooBars"`
}
