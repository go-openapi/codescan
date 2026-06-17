// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package recursioncollision is the name-identity witness that combines the two
// hard cases: a type that is BOTH self-recursive AND name-colliding across
// packages. p.Node and q.Node each declare `swagger:model Node` with a
// self-reference (Next *Node). The reduce stage must qualify the colliding
// names (PNode / QNode) AND rewrite each self-`$ref` in lockstep with its own
// renamed key — never dangling at the pre-reduce deep key, never pointing at
// the sibling package's Node.
package recursioncollision

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-recursion-collision/p"
	"github.com/go-openapi/codescan/fixtures/enhancements/name-identity-recursion-collision/q"
)

// swagger:model Tree
type Tree struct {
	P p.Node `json:"p"`
	Q q.Node `json:"q"`
}
