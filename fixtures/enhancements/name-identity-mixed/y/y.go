// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package y

// swagger:model Item
type Item struct {
	Y1 int64 `json:"y1"`
}

// Record is intentionally NOT annotated: it becomes a definition only because
// it is referenced from mixed.Root (the implicit-collision arm).
type Record struct {
	RY bool `json:"ry"`
}
