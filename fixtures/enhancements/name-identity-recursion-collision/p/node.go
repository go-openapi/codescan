// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package p

// swagger:model Node
type Node struct {
	Value string `json:"value"`
	Next  *Node  `json:"next,omitempty"`
}
