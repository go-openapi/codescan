// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package recursion is a name-identity corpus member: the G3 control. It holds
// LEGITIMATE recursion that must keep working unchanged through every stage of
// the deconfliction engine — a property/items back-$ref to an ancestor
// definition is valid OAS, unlike #2637's top-level self-$ref body.
//
//   - Node: direct self-recursion (Node.next -> #/definitions/Node)
//   - Loop/Knot: mutual recursion (Loop.knot -> Knot, Knot.loop -> Loop)
package recursion

// swagger:model Node
type Node struct {
	Value string `json:"value"`
	Next  *Node  `json:"next,omitempty"`
}

// swagger:model Loop
type Loop struct {
	Name string `json:"name"`
	Knot *Knot  `json:"knot,omitempty"`
}

// swagger:model Knot
type Knot struct {
	ID   int64 `json:"id"`
	Loop *Loop `json:"loop,omitempty"`
}
