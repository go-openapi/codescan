// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2637 reproduces go-swagger issue #2637 ("Cyclic type definition
// for defined types using the same name"): a local type defined from a
// same-named type in another package collides on the short key and the
// definition gets a SELF-$ref alongside its properties — an invalid, self-
// cyclic definition that hangs downstream codegen. Same family as #2783.
package bug2637

import "github.com/go-openapi/codescan/fixtures/bugs/2637/mongo"

// swagger:model CreateDomainRequest
type CreateDomainRequest mongo.CreateDomainRequest

// swagger:parameters createDomain
type createDomainParams struct {
	// in: body
	Body CreateDomainRequest
}

// swagger:route POST /domains domains createDomain
//
// responses:
//   200: description: ok
func createDomain() {}
