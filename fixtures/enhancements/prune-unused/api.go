// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package pruneunused exercises Options.PruneUnusedModels.
//
// A single route references Used, which transitively reaches a chain (B -> C),
// a self-recursive model (Node), and a cross-package model (a.Thing). These are
// the reachable set and must survive a prune.
//
// Unused (a swagger:model nothing references) and OnlyByUnused (referenced only
// by Unused) must be pruned. b.Thing is a swagger:model in a sibling package
// that nothing references and must also be pruned — and because it is pruned
// BEFORE name reduction, its collision with a.Thing never materialises, so
// a.Thing keeps the bare name "Thing" instead of being deconflicted to AThing /
// BThing. That is the headline property of pruning before reduction.
package pruneunused

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/prune-unused/a"
	"github.com/go-openapi/codescan/fixtures/enhancements/prune-unused/b"
)

// Used is referenced from the route response — the reachability root.
//
// swagger:model Used
type Used struct {
	B     B       `json:"b"`
	Thing a.Thing `json:"thing"`
	Node  *Node   `json:"node"`
}

// B is reached via Used -> B -> C.
//
// swagger:model B
type B struct {
	C C `json:"c"`
}

// C is the chain tail.
//
// swagger:model C
type C struct {
	Name string `json:"name"`
}

// Node is self-recursive and reached from Used; the reachability walk must
// terminate on the cycle, not loop.
//
// swagger:model Node
type Node struct {
	Next *Node `json:"next"`
}

// Unused is a swagger:model that nothing references — pruned.
//
// swagger:model Unused
type Unused struct {
	Only OnlyByUnused `json:"only"`
}

// OnlyByUnused is referenced only by Unused (itself unreferenced), so it is
// reachable only through dead nodes — pruned too.
//
// swagger:model OnlyByUnused
type OnlyByUnused struct {
	V string `json:"v"`
}

// usedResp carries Used in its body so the route reaches it.
//
// swagger:response usedResp
type usedResp struct {
	// in: body
	Body Used `json:"body"`
}

// handler is the only route.
//
// swagger:route GET /used usedOp
//
// Used.
//
// responses:
//
//	200: usedResp
func handler() {}

// keep the imported packages referenced so the tree type-checks even though
// b is only reached by the scanner, not by Go code.
var (
	_ a.Thing
	_ b.Thing
)
