// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package examplesnomix is the name-identity witness for go-swagger #2126
// ("reference swagger models under a specific package"): two packages each
// declare `swagger:model Widget` with the SAME json field but DIFFERENT
// example values. Before the fix the colliding definitions merged and one
// package's example clobbered the other's; now each Widget keeps its own
// example, proving the per-package identity (and its example) is preserved.
package examplesnomix

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-examples-no-mix/a"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-examples-no-mix/b"
)

// swagger:model Catalog
type Catalog struct {
	A a.Widget `json:"a"`
	B b.Widget `json:"b"`
}
