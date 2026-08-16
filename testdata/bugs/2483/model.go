// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2483

// SimpleOne is a model with a few simple fields.
//
// swagger:model SimpleOne
type SimpleOne struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Age  int32  `json:"age"`
}

// Something is used by other structs (not itself annotated).
type Something struct {
	DID int64  `json:"did"`
	Cat string `json:"cat"`
}

// Pet embeds SimpleOne as an allOf base and Something inline. The reporter saw
// the allOf arms duplicated (SimpleOne $ref + the inline object each appearing
// twice), which also tripped a "circular ancestry" validation error.
//
// swagger:model Pet
type Pet struct {
	// swagger:allOf
	SimpleOne

	Something

	Notes string `json:"notes"`

	Extra string `json:"extra"`
}
