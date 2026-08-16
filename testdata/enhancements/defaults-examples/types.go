// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package defaults_examples exercises default and example tag parsing
// across numeric, array and object schema types.
package defaults_examples

// Metrics aggregates fields that carry default and example tags for the
// numeric, array and object branches of parseValueFromSchema.
//
// swagger:model
type Metrics struct {
	// Ratio is a float32 value.
	//
	// default: 1.5
	// example: 2.25
	Ratio float32 `json:"ratio"`

	// Weight is a float64 value.
	//
	// default: 3.14
	// example: 9.81
	Weight float64 `json:"weight"`

	// Tags is a slice with a JSON-array default and example.
	//
	// default: ["a","b"]
	// example: ["x","y","z"]
	Tags []string `json:"tags"`

	// Counts is a slice of integers with a JSON-array default.
	//
	// default: [1,2,3]
	// example: [4,5]
	Counts []int `json:"counts"`

	// Props is a map represented as a JSON object.
	//
	// default: {"k":1}
	// example: {"q":42,"r":7}
	Props map[string]int `json:"props"`
}
