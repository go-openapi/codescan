// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package inventory provides a model referenced cross-package by a doc-link.
package inventory

// Ledger records stock movements.
//
// swagger:model
type Ledger struct {
	Entries int `json:"entries"`
}
