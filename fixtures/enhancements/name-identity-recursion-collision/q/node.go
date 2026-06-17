// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package q

// swagger:model Node
type Node struct {
	ID   int64 `json:"id"`
	Next *Node `json:"next,omitempty"`
}
