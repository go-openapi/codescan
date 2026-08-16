// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package maps stands in for an external library (googlemaps.github.io/maps)
// whose exported struct carries a custom-typed field.
package maps

// RankBy is a custom string type (the field that #413 said could not be
// resolved when the enclosing struct was embedded).
type RankBy string

// NearbySearchRequest is the external struct that gets embedded.
type NearbySearchRequest struct {
	Keyword string `json:"keyword"`

	RankBy RankBy `json:"rankby"`
}
