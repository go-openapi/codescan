// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2783 reproduces go-swagger issue #2783 ("Models get mixed when
// using structs from several packages"): two packages each declare a
// swagger:model named Test. They collide on the short definition key "Test"
// and silently merge into one definition (union of fields, last package wins),
// instead of being disambiguated.
package bug2783

import (
	"github.com/go-openapi/codescan/fixtures/bugs/2783/b"
	"github.com/go-openapi/codescan/fixtures/bugs/2783/c"
)

// swagger:model TestResponseBody
type TestResponseBody struct {
	Test1 b.Test `json:"test1"`
	Test2 c.Test `json:"test2"`
}
