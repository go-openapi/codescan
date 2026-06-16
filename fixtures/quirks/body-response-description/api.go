// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bodyresponsedescription exercises quirk G1 (doc-site-quirks.md): a
// `body:` response line with NO trailing description text used to fill
// response.description with the bare Go type token ("Pet", "string"), leaking
// an implementation detail into the contract. The description is now derived,
// in order of preference, from
//
//	1. the referenced model's godoc (its title, then description);
//	2. the HTTP status reason phrase for a numeric code;
//	3. a neutral "default response" placeholder for `default` / odd codes.
package bodyresponsedescription

// swagger:route GET /quirk things g1
//
// Exercises every G1 description tier in one operation.
//
// responses:
//
//	200: body:Pet
//	404: body:Blank
//	500: body:string
//	default: body:string
func g1() {}

// Pet is a documented model used as a response body.
//
// swagger:model Pet
type Pet struct {
	Name string `json:"name"`
}

// swagger:model Blank
type Blank struct {
	ID int64 `json:"id"`
}
