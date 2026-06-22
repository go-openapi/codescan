// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package a holds the referenced half of the a.Thing / b.Thing collision pair.
package a

// Thing is referenced from Used.Thing, so it survives a prune. With b.Thing
// pruned away first, this keeps the bare name "Thing".
//
// swagger:model Thing
type Thing struct {
	A string `json:"a"`
}
