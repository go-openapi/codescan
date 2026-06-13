// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2922 reproduces go-swagger issue #2922 ("enum description:
// superfluous name&values"): the enum const-name mapping (e.g. "FIRST
// TestEnumFirst") is appended to the property description AND duplicated in
// x-go-enum-desc. The reporter finds the description pollution superfluous.
//
// The mapping is folded in at every affected target — both a schema property
// (in: body) and a non-body parameter (in: query) — so the SkipEnumDescriptions
// knob is exercised across both the schema and parameters builders.
package bug2922

// swagger:model GetEnumTestResponse
type GetEnumTestResponse struct {
	// The description of the test enum in the response body
	TestEnumInBody TestEnum `json:"testEnumInBody"`
}

// swagger:parameters GetEnumTest
type GetTestEnumParam struct {
	// The description of the test enum param
	// in: query
	TestEnumInQueryParam TestEnum `json:"testEnumInQueryParam"`
}

// EnumHeaderResponse carries an enum-typed response header.
//
// swagger:response enumHeaderResponse
type EnumHeaderResponse struct {
	// The description of the enum header
	// in: header
	TestEnumInHeader TestEnum `json:"X-Test-Enum"`
}

// swagger:route GET /enum/test things GetEnumTest
//
// responses:
//   200: GetEnumTestResponse
func GetEnumTest() {}

// swagger:enum TestEnum
type TestEnum string

const (
	TestEnumFirst  TestEnum = "FIRST"
	TestEnumSecond TestEnum = "SECOND"
)
