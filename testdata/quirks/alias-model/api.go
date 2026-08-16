// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package aliasmodel is the F9 regression lock (doc-site-quirks.md): a
// model-annotated Go type alias once sent the scanner into an infinite loop
// when built (default / RefAliases modes); TransparentAliases dissolved it
// before the looping path. The hang is gone on the current base — this
// fixture keeps it gone. A regression would resurface as a test timeout.
package aliasmodel

// Money is the underlying first-class model.
//
// swagger:model
type Money struct {
	// Cents is the amount in cents.
	Cents int64 `json:"cents"`
	// Currency is the ISO currency code.
	Currency string `json:"currency"`
}

// Price is a Go type alias of Money that itself carries the model annotation
// — the first-class-alias form that used to hang the build.
//
// swagger:model
type Price = Money

// Invoice references the aliased Price, so the alias is reachable and built.
//
// swagger:model
type Invoice struct {
	// Total is the invoice total.
	Total Price `json:"total"`
}
