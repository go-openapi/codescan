// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package paramsx holds a parameters struct used by a swagger:operation in
// another package (go-swagger#1955).
package paramsx

// swagger:parameters crossOp
type CrossParams struct {
	// in: query
	Q string `json:"q"`
}
