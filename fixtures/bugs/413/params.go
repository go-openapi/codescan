// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug413

import "github.com/go-openapi/codescan/fixtures/bugs/413/maps"

// NearbySearchRequest embeds an external-package struct that itself contains a
// custom-typed field. The legacy scanner failed with "unable to resolve
// embedded struct for: RankBy".
//
// swagger:parameters Nearby
type NearbySearchRequest struct {
	// in: query
	maps.NearbySearchRequest
}

// PlacesNearby exposes the Nearby operation so the embedded params bind to a path.
//
// swagger:route POST /places/nearby places Nearby
//
// Search for a place nearby.
//
//	Responses:
//	  200: description: ok
func PlacesNearby() {}
