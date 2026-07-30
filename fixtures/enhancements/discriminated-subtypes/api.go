// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package discriminatedsubtypes exercises the auto-discovery of discriminator
// subtypes of a referenced base (go-swagger#1913).
//
// A single route references base.TeslaCar — a discriminated base (its `model`
// member carries `discriminator: true`). Nothing $refs the subtypes: THEY refer
// to the base via `swagger:allOf`, so a plain top-down reachability walk never
// reaches them. The reverse `swagger:allOf` index pulls them in:
//
//   - ModelS / ModelX embed the base from this package;
//   - sub.ModelY embeds it from another package (the index is module-wide);
//   - Battery is reached transitively from ModelS (a pulled subtype keeps
//     discovering).
//
// Two negative controls guard against over-pull: Unrelated is a swagger:model
// nothing references, and PlainSub declares base.PlainBase as an allOf member,
// yet that base carries no discriminator. Neither is emitted without ScanModels.
package discriminatedsubtypes

import (
	"github.com/go-openapi/codescan/fixtures/enhancements/discriminated-subtypes/base"
)

// ModelS is a subtype of base.TeslaCar.
//
// It is reached only through the reverse index, and itself pulls Battery into the
// closure.
//
// swagger:model modelS
type ModelS struct {
	// swagger:allOf
	base.TeslaCar

	// The edition of this Model S
	Edition string `json:"edition"`

	// Battery is discovered transitively from a pulled subtype.
	Battery Battery `json:"battery"`
}

// ModelX is another subtype of base.TeslaCar.
//
// swagger:model modelX
type ModelX struct {
	// swagger:allOf
	base.TeslaCar

	// Number of doors
	Doors int `json:"doors"`
}

// Battery is reached from ModelS only.
//
// It proves a pulled subtype keeps discovering its own dependencies.
//
// swagger:model Battery
type Battery struct {
	// Capacity in kWh
	Capacity int `json:"capacity"`
}

// PlainSub declares a non-discriminated base as an allOf member.
//
// It is the negative control for the discriminated gate.
//
// swagger:model PlainSub
type PlainSub struct {
	// swagger:allOf
	base.PlainBase

	// Extra field
	Extra string `json:"extra"`
}

// Unrelated is a swagger:model nothing references.
//
// It is the negative control for over-generation.
//
// swagger:model Unrelated
type Unrelated struct {
	V string `json:"v"`
}

// carResp carries the polymorphic base in its body.
//
// The route therefore reaches the base — and only the base.
//
// swagger:response carResp
type carResp struct {
	// in: body
	Body base.TeslaCar `json:"body"`
}

// handler is the only route.
//
// swagger:route GET /cars listCars
//
// Lists tesla cars.
//
// responses:
//
//	200: carResp
func handler() {}
