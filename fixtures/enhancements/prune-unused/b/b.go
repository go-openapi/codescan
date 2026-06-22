// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package b holds the unreferenced half of the a.Thing / b.Thing collision
// pair. Nothing references b.Thing, so PruneUnusedModels drops it before name
// reduction — and the a.Thing / b.Thing collision never surfaces.
package b

// Thing is a swagger:model that nothing references — pruned.
//
// swagger:model Thing
type Thing struct {
	B string `json:"b"`
}
