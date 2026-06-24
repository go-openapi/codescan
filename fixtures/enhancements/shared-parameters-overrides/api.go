// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters_overrides (Fixture 5) witnesses two caveats
// of the shared-parameters feature (go-swagger#2632):
//
//   - C3: the shared key and every reference use the FINAL, overridden
//     parameter name (the `name:` keyword wins over the json tag), not
//     the Go field name.
//   - C1/C2: duplicate op-id targets and duplicate reference names are
//     deduplicated, each surplus raising a warning.
//
// See .claude/plans/features/shared-parameters-fixtures.md.
package shared_parameters_overrides

// CommonHeaders registers a shared header whose spec name is OVERRIDDEN
// by the `name:` keyword: it lands at #/parameters/X-Correlation-ID, not
// #/parameters/X-Request-ID. References must use the overridden name.
//
// swagger:parameters *
type CommonHeaders struct {
	// RequestID correlates a request across services.
	//
	// in: header
	// name: X-Correlation-ID
	RequestID string `json:"X-Request-ID"`
}

// AuthHeader registers #/parameters/X-API-Key and $ref's it into
// createThing. The op-id list repeats createThing: the duplicate is
// dropped (C1) and a scan.duplicate-target warning is raised.
//
// swagger:parameters * createThing createThing
type AuthHeader struct {
	// APIKey authorises access.
	//
	// in: header
	APIKey string `json:"X-API-Key"`
}

// ListThings lists things. The standalone reference marker repeats the
// shared name X-Correlation-ID: the duplicate is dropped (C2) with a
// scan.duplicate-ref warning, leaving a single $ref. It also proves a
// reference resolves by the OVERRIDDEN name.
//
// swagger:route GET /things things listThings
// swagger:parameters listThings X-Correlation-ID X-Correlation-ID
// Responses:
//
//	200: description: OK
func ListThings() {}

// CreateThing creates a thing; it gets the $ref'd X-API-Key (once).
//
// swagger:route POST /things things createThing
// Responses:
//
//	201: description: Created
func CreateThing() {}
