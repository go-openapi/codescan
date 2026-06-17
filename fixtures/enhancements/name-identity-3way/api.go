// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package threeway is a name-identity corpus member: THREE packages each
// declare a `swagger:model Widget` with distinct fields, all referenced from
// one container. Today they collide on the short key "Widget" and merge into a
// single definition (union of fields, last package wins, non-deterministic).
// Target: three distinct definitions.
package threeway

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-3way/a"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-3way/b"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-3way/c"
)

// swagger:model Container
type Container struct {
	A a.Widget `json:"a"`
	B b.Widget `json:"b"`
	C c.Widget `json:"c"`
}
