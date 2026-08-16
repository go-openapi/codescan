// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1279

// GetCountryDetails probes inline path-parameter declaration in a swagger:route
// block (no wrapper swagger:parameters struct), as the reporter wished:
// "Parameters: country string in:path".
//
// swagger:route GET /{country} getCountryDetails
//
// Returns details for the given country.
//
//	Parameters:
//	  + name: country
//	    in: path
//	    type: string
//	    required: true
//
//	Responses:
//	  200: description: ok
func GetCountryDetails() {}
