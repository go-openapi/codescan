// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2002

import "github.com/go-openapi/codescan/fixtures/bugs/2002/api"

// swagger:route POST /foobar foobar idOfFoobar
//
// Foobar does some amazing stuff.
//
// responses:
//
//	200: foobarResponse
func Foobar() {}

// swagger:response foobarResponse
type foobarResponseWrapper struct {
	// in:body
	Body api.FooBarResponse
}
